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
}

func (h *handler) init() error {
	if h.maxConcurrentWorkers <= 0 {
		h.maxConcurrentWorkers = 1
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	models := []interface{}{
		(*models.SpecialServer)(nil),
		(*models.Server)(nil),
		(*models.LangVersion)(nil),
		(*models.PlayerToServer)(nil),
		(*models.PlayerNameChange)(nil),
	}

	for _, model := range models {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	for _, statement := range []string{
		pgDefaultValues,
		allVersionsPGInsertStatements,
		allSpecialServersPGInsertStatements,
	} {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (h *handler) createSchema(server *models.Server) error {
	tx, err := h.db.WithParam("SERVER", pg.Safe(server.Key)).Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if _, err := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", server.Key)); err != nil {
		return err
	}

	models := []interface{}{
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

	for _, model := range models {
		err := tx.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	for _, statement := range []string{
		serverPGFunctions,
		serverPGTriggers,
		serverPGDefaultValues,
	} {
		if _, err := tx.Exec(statement, pg.Safe(server.Key), server.LangVersionTag); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (h *handler) getServers() ([]*models.Server, map[string]string, error) {
	langVersions := []*models.LangVersion{}
	if err := h.db.Model(&langVersions).Relation("SpecialServers").Order("tag ASC").Select(); err != nil {
		return nil, nil, errors.Wrap(err, "getServers")
	}

	serverKeys := []string{}
	servers := []*models.Server{}
	urls := make(map[string]string)
	loadedLangVersions := []models.LanguageTag{}
	for _, langVersion := range langVersions {
		log := log.WithField("host", langVersion.Host)
		log.Infof("Loading servers from %s", langVersion.Host)
		resp, err := http.Get(fmt.Sprintf("https://%s%s", langVersion.Host, endpointGetServers))
		if err != nil {
			log.Errorln(errors.Wrapf(err, "Cannot fetch servers from %s", langVersion.Host))
			continue
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorln(errors.Wrapf(err, "Cannot read response body from %s", langVersion.Host))
			continue
		}
		body, err := phpserialize.Decode(string(bodyBytes))
		if err != nil {
			log.Errorln(errors.Wrapf(err, "Cannot serialize body from %s into go value", langVersion.Host))
			continue
		}
		for serverKey, url := range body.(map[interface{}]interface{}) {
			serverKeyStr := serverKey.(string)
			if langVersion.SpecialServers.Contains(serverKeyStr) {
				continue
			}
			server := &models.Server{
				Key:            serverKeyStr,
				Status:         models.ServerStatusOpen,
				LangVersionTag: langVersion.Tag,
				LangVersion:    langVersion,
			}
			if err := h.createSchema(server); err != nil {
				log.WithField("serverKey", serverKey).Errorln(errors.Wrapf(err, "Cannot create schema for %s", serverKey))
				continue
			}
			serverKeys = append(serverKeys, serverKeyStr)
			urls[serverKeyStr] = url.(string)
			servers = append(servers, server)
		}
		loadedLangVersions = append(loadedLangVersions, langVersion.Tag)
	}

	if len(servers) > 0 {
		if _, err := h.db.Model(&servers).
			OnConflict("(key) DO UPDATE").
			Set("status = ?", models.ServerStatusOpen).
			Set("lang_version_tag = EXCLUDED.lang_version_tag").
			Returning("*").
			Insert(); err != nil {
			return nil, nil, err
		}
	}

	if _, err := h.db.Model(&models.Server{}).
		Set("status = ?", models.ServerStatusClosed).
		Where("key NOT IN (?) AND lang_version_tag IN (?)", pg.In(serverKeys), pg.In(loadedLangVersions)).
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
	p := newPool(h.maxConcurrentWorkers)
	defer p.close()

	for _, server := range servers {
		log := log.WithField("serverKey", server.Key)
		url, ok := urls[server.Key]
		if !ok {
			log.Warnf("No one URL associated with key: %s, skipping...", server.Key)
			continue
		}
		p.waitForWorker()
		wg.Add(1)
		sh := &updateServerDataWorker{
			db:     h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server: server,
			dataloader: dataloader.New(&dataloader.Config{
				BaseURL: url,
			}),
		}
		go func(worker *updateServerDataWorker, server *models.Server, url string, log *logrus.Entry) {
			defer p.releaseWorker()
			defer wg.Done()
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

func (h *handler) updateHistory() {
	servers := []*models.Server{}
	now := time.Now()
	t1 := time.Date(now.Year(), now.Month(), now.Day(), 0, 30, 0, 0, time.UTC)
	err := h.db.
		Model(&servers).
		Where("status = ? AND (history_updated_at < ? OR history_updated_at IS NULL)", models.ServerStatusOpen, t1).
		Select()
	if err != nil {
		log.Errorln(errors.Wrap(err, "updateHistory"))
		return
	}
	log.
		WithField("numberOfServers", len(servers)).
		Info("updateHistory: servers loaded")

	var wg sync.WaitGroup
	p := newPool(runtime.NumCPU())
	defer p.close()

	for _, server := range servers {
		p.waitForWorker()
		wg.Add(1)
		worker := &updateServerHistoryWorker{
			db:     h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server: server,
		}
		go func(server *models.Server, worker *updateServerHistoryWorker) {
			defer p.releaseWorker()
			defer wg.Done()
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

func (h *handler) updateServerStats(t time.Time) error {
	servers := []*models.Server{}
	err := h.db.
		Model(&servers).
		Where("status = ?  AND (stats_updated_at < ? OR stats_updated_at IS NULL)", models.ServerStatusOpen, t).
		Select()
	if err != nil {
		return errors.Wrap(err, "updateServerStats")
	}
	log.WithField("numberOfServers", len(servers)).Info("updateServerStats: servers loaded")

	var wg sync.WaitGroup
	p := newPool(runtime.NumCPU())
	defer p.close()

	for _, server := range servers {
		p.waitForWorker()
		wg.Add(1)
		worker := &updateServerStatsWorker{
			db:     h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server: server,
		}
		go func(server *models.Server, worker *updateServerStatsWorker) {
			defer p.releaseWorker()
			defer wg.Done()
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
	return nil
}

func (h *handler) updateStats() {
	now := time.Now()
	t1 := time.Date(now.Year(), now.Month(), now.Day(), 1, 30, 0, 0, time.UTC)
	if err := h.updateServerStats(t1); err != nil {
		log.Error(errors.Wrap(err, "updateStats"))
	}
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
	p := newPool(runtime.NumCPU())
	defer p.close()

	for _, server := range servers {
		p.waitForWorker()
		wg.Add(1)
		worker := &vacuumServerDBWorker{
			db: h.db.WithParam("SERVER", pg.Safe(server.Key)),
		}
		go func(server *models.Server, worker *vacuumServerDBWorker, p *pool) {
			defer p.releaseWorker()
			defer wg.Done()
			log := log.WithField("serverKey", server.Key)
			log.Infof("vacuumDatabase: %s: vacuuming database", server.Key)
			if err := worker.vacuum(); err != nil {
				log.Errorln("vacuumDatabase:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("vacuumDatabase: %s: database vacuumed", server.Key)
		}(server, worker, p)
	}

	wg.Wait()
}
