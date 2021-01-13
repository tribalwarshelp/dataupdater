package cron

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/models"
	"github.com/tribalwarshelp/shared/tw/dataloader"

	phpserialize "github.com/Kichiyaki/go-php-serialize"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
)

const (
	endpointGetServers = "/backend/get_servers.php"
)

type handler struct {
	db                   *pg.DB
	maxConcurrentWorkers int
	pool                 *pool
}

func (h *handler) init() error {
	if h.maxConcurrentWorkers <= 0 {
		h.maxConcurrentWorkers = runtime.NumCPU()
	}

	if h.pool == nil {
		h.pool = newPool(h.maxConcurrentWorkers)
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	dbModels := []interface{}{
		(*models.SpecialServer)(nil),
		(*models.Server)(nil),
		(*models.Version)(nil),
		(*models.PlayerToServer)(nil),
		(*models.PlayerNameChange)(nil),
	}

	for _, model := range dbModels {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	type statementWithParams struct {
		statement string
		params    []interface{}
	}

	for _, s := range []statementWithParams{
		{
			statement: pgDefaultValues,
		},
		{
			statement: allVersionsPGInsertStatements,
		},
		{
			statement: allSpecialServersPGInsertStatements,
		},
		{
			statement: pgDropSchemaFunctions,
			params:    []interface{}{pg.Safe("public")},
		},
		{
			statement: pgFunctions,
		},
	} {
		if _, err := tx.Exec(s.statement, s.params...); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	servers := []*models.Server{}
	if err := h.db.Model(&servers).Select(); err != nil {
		return err
	}

	for _, server := range servers {
		if err := h.createSchema(server, true); err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) createSchema(server *models.Server, init bool) error {
	if !init {
		exists, err := h.db.Model().Table("information_schema.schemata").Where("schema_name = ?", server.Key).Exists()
		if err != nil {
			return err
		}

		if exists {
			return nil
		}
	}

	tx, err := h.db.WithParam("SERVER", pg.Safe(server.Key)).Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if _, err := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", server.Key)); err != nil {
		return err
	}

	dbModels := []interface{}{
		(*models.Tribe)(nil),
		(*models.Player)(nil),
		(*models.Village)(nil),
		(*models.Ennoblement)(nil),
		(*models.ServerStats)(nil),
		(*models.TribeHistory)(nil),
		(*models.PlayerHistory)(nil),
		(*models.TribeChange)(nil),
		(*models.DailyPlayerStats)(nil),
		(*models.DailyTribeStats)(nil),
	}

	for _, model := range dbModels {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	statements := []string{
		serverPGFunctions,
		serverPGTriggers,
		serverPGDefaultValues,
	}
	if init {
		statements = append([]string{pgDropSchemaFunctions}, statements...)
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement, pg.Safe(server.Key), server.VersionCode); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (h *handler) getServers() ([]*models.Server, map[string]string, error) {
	versions := []*models.Version{}
	if err := h.db.Model(&versions).Relation("SpecialServers").Order("code ASC").Select(); err != nil {
		return nil, nil, errors.Wrap(err, "getServers")
	}

	serverKeys := []string{}
	servers := []*models.Server{}
	urls := make(map[string]string)
	loadedVersions := []models.VersionCode{}
	for _, version := range versions {
		log := log.WithField("host", version.Host)
		log.Infof("Loading servers from %s", version.Host)
		resp, err := http.Get(fmt.Sprintf("https://%s%s", version.Host, endpointGetServers))
		if err != nil {
			log.Errorln(errors.Wrapf(err, "fetching servers from %s", version.Host))
			continue
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorln(errors.Wrapf(err, "reading response body from %s", version.Host))
			continue
		}
		body, err := phpserialize.Decode(string(bodyBytes))
		if err != nil {
			log.Errorln(errors.Wrapf(err, "serializing body from %s into go value", version.Host))
			continue
		}
		for serverKey, url := range body.(map[interface{}]interface{}) {
			serverKeyStr := serverKey.(string)
			if version.SpecialServers.Contains(serverKeyStr) {
				continue
			}
			server := &models.Server{
				Key:         serverKeyStr,
				Status:      models.ServerStatusOpen,
				VersionCode: version.Code,
				Version:     version,
			}
			if err := h.createSchema(server, false); err != nil {
				log.WithField("serverKey", serverKey).Errorln(errors.Wrapf(err, "cannot create schema for %s", serverKey))
				continue
			}
			serverKeys = append(serverKeys, serverKeyStr)
			urls[serverKeyStr] = url.(string)
			servers = append(servers, server)
		}
		loadedVersions = append(loadedVersions, version.Code)
	}

	if len(servers) > 0 {
		if _, err := h.db.Model(&servers).
			OnConflict("(key) DO UPDATE").
			Set("status = ?", models.ServerStatusOpen).
			Set("version_code = EXCLUDED.version_code").
			Returning("*").
			Insert(); err != nil {
			return nil, nil, err
		}
	}

	if _, err := h.db.Model(&models.Server{}).
		Set("status = ?", models.ServerStatusClosed).
		Where("key NOT IN (?) AND version_code IN (?)", pg.In(serverKeys), pg.In(loadedVersions)).
		Update(); err != nil {
		return nil, nil, err
	}

	return servers, urls, nil
}

func (h *handler) updateServerData() {
	servers, urls, err := h.getServers()
	if err != nil {
		log.Errorln("updateServerData:", err.Error())
		return
	}
	log.
		WithField("numberOfServers", len(servers)).
		Info("updateServerData: servers loaded")

	var wg sync.WaitGroup

	for _, server := range servers {
		log := log.WithField("serverKey", server.Key)
		url, ok := urls[server.Key]
		if !ok {
			log.Warnf("No one URL associated with key: %s, skipping...", server.Key)
			continue
		}
		h.pool.waitForWorker()
		wg.Add(1)
		sh := &updateServerDataWorker{
			db:     h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server: server,
			dataloader: dataloader.New(&dataloader.Config{
				BaseURL: url,
			}),
		}
		go func(worker *updateServerDataWorker, server *models.Server, url string, log *logrus.Entry) {
			defer func() {
				h.pool.releaseWorker()
				wg.Done()
			}()
			log.Infof("updateServerData: %s: updating data", server.Key)
			err := sh.update()
			if err != nil {
				log.Errorln("updateServerData:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("updateServerData: %s: data updated", server.Key)
		}(sh, server, url, log)
	}

	wg.Wait()
}

func (h *handler) updateHistory(location *time.Location) {
	servers := []*models.Server{}
	log := log.WithField("timezone", location.String())
	err := h.db.
		Model(&servers).
		Where("status = ? AND (history_updated_at IS NULL OR now() - history_updated_at > '23 hours') AND timezone = ?",
			models.ServerStatusOpen,
			location.String()).
		Relation("Version").
		Select()
	if err != nil {
		log.Errorln(errors.Wrap(err, "updateHistory"))
		return
	}
	log.
		WithField("numberOfServers", len(servers)).
		Info("updateHistory: servers loaded")

	var wg sync.WaitGroup

	for _, server := range servers {
		h.pool.waitForWorker()
		wg.Add(1)
		worker := &updateServerHistoryWorker{
			db:       h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server:   server,
			location: location,
		}
		go func(server *models.Server, worker *updateServerHistoryWorker) {
			defer func() {
				h.pool.releaseWorker()
				wg.Done()
			}()
			log := log.WithField("serverKey", server.Key)
			log.Infof("updateHistory: %s: updating history", server.Key)
			if err := worker.update(); err != nil {
				log.Errorln("updateHistory:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("updateHistory: %s: history updated", server.Key)
		}(server, worker)
	}

	wg.Wait()
}

func (h *handler) updateStats(location *time.Location) {
	servers := []*models.Server{}
	log := log.WithField("timezone", location.String())
	err := h.db.
		Model(&servers).
		Where("status = ? AND (stats_updated_at IS NULL OR now() - stats_updated_at > '23 hours') AND timezone = ?",
			models.ServerStatusOpen,
			location.String()).
		Relation("Version").
		Select()
	if err != nil {
		log.Errorf(errors.Wrap(err, "updateServerStats").Error())
		return
	}
	log.WithField("numberOfServers", len(servers)).Info("updateServerStats: servers loaded")

	var wg sync.WaitGroup

	for _, server := range servers {
		h.pool.waitForWorker()
		wg.Add(1)
		worker := &updateServerStatsWorker{
			db:       h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server:   server,
			location: location,
		}
		go func(server *models.Server, worker *updateServerStatsWorker) {
			defer func() {
				h.pool.releaseWorker()
				wg.Done()
			}()
			log := log.WithField("serverKey", server.Key)
			log.Infof("updateServerStats: %s: updating stats", server.Key)
			if err := worker.update(); err != nil {
				log.Errorln("updateServerStats:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("updateServerStats: %s: stats updated", server.Key)
		}(server, worker)
	}

	wg.Wait()
}

func (h *handler) vacuumDatabase() {
	servers := []*models.Server{}
	err := h.db.
		Model(&servers).
		Select()
	if err != nil {
		log.Errorln(errors.Wrap(err, "vacuumDatabase"))
		return
	}

	var wg sync.WaitGroup

	for _, server := range servers {
		h.pool.waitForWorker()
		wg.Add(1)
		worker := &vacuumServerDBWorker{
			db: h.db.WithParam("SERVER", pg.Safe(server.Key)),
		}
		go func(server *models.Server, worker *vacuumServerDBWorker) {
			defer func() {
				h.pool.releaseWorker()
				wg.Done()
			}()
			log := log.WithField("serverKey", server.Key)
			log.Infof("vacuumDatabase: %s: vacuuming database", server.Key)
			if err := worker.vacuum(); err != nil {
				log.Errorln("vacuumDatabase:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("vacuumDatabase: %s: database vacuumed", server.Key)
		}(server, worker)
	}

	wg.Wait()
}
