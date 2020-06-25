package cron

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/tribalwarshelp/shared/models"

	phpserialize "github.com/Kichiyaki/go-php-serialize"
	"github.com/robfig/cron/v3"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/pkg/errors"
)

const (
	endpointGetServers = "/backend/get_servers.php"
)

type handler struct {
	db *pg.DB
}

func Attach(c *cron.Cron, db *pg.DB) error {
	h := &handler{db}
	if err := h.init(); err != nil {
		return err
	}

	if _, err := c.AddFunc("0 * * * *", h.updateServersData); err != nil {
		return err
	}
	if _, err := c.AddFunc("30 0 * * *", h.updateServersHistory); err != nil {
		return err
	}
	if _, err := c.AddFunc("30 1 * * *", h.updateStats); err != nil {
		return err
	}
	if _, err := c.AddFunc("30 2 * * *", h.vacuumDatabase); err != nil {
		return err
	}
	go func() {
		h.updateServersData()
		h.vacuumDatabase()
		h.updateServersHistory()
		h.updateStats()
	}()

	return nil
}

func (h *handler) init() error {
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
		err := tx.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
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
		err := tx.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	for _, statement := range []string{
		serverPGFunctions,
		serverPGTriggers,
	} {
		if _, err := tx.Exec(statement, pg.Safe(server.Key), server.LangVersionTag); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (h *handler) getServers() ([]*models.Server, map[string]string, error) {
	langVersions := []*models.LangVersion{}
	if err := h.db.Model(&langVersions).Relation("SpecialServers").Select(); err != nil {
		return nil, nil, errors.Wrap(err, "getServers")
	}

	serverKeys := []string{}
	servers := []*models.Server{}
	urls := make(map[string]string)
	for _, langVersion := range langVersions {
		resp, err := http.Get(fmt.Sprintf("https://%s%s", langVersion.Host, endpointGetServers))
		if err != nil {
			log.Print(errors.Wrapf(err, "Cannot fetch servers from %s", langVersion.Host))
			continue
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Print(errors.Wrapf(err, "Cannot read response body from %s", langVersion.Host))
			continue
		}
		body, err := phpserialize.Decode(string(bodyBytes))
		if err != nil {
			log.Print(errors.Wrapf(err, "Cannot serialize body from %s into go value", langVersion.Host))
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
				log.Print(errors.Wrapf(err, "Cannot create schema for %s", serverKey))
				continue
			}
			serverKeys = append(serverKeys, serverKeyStr)
			urls[serverKeyStr] = url.(string)
			servers = append(servers, server)
		}
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
		Where("key NOT IN (?)", pg.In(serverKeys)).
		Update(); err != nil {
		return nil, nil, err
	}

	return servers, urls, nil
}

func (h *handler) updateServersData() {
	servers, urls, err := h.getServers()
	if err != nil {
		log.Println(err.Error())
		return
	}

	var wg sync.WaitGroup
	max := runtime.NumCPU() * 2
	count := 0

	for _, server := range servers {
		url, ok := urls[server.Key]
		if !ok {
			log.Printf("No one URL associated with key: %s, skipping...", server.Key)
			continue
		}
		if count >= max {
			wg.Wait()
			count = 0
		}
		sh := &updateServerDataHandler{
			db:      h.db.WithParam("SERVER", pg.Safe(server.Key)),
			baseURL: url,
			server:  server,
		}
		count++
		wg.Add(1)
		go func(server *models.Server, sh *updateServerDataHandler) {
			defer wg.Done()
			log.Printf("%s: updating data", server.Key)
			if err := sh.update(); err != nil {
				log.Println(errors.Wrap(err, server.Key))
				return
			} else {
				log.Printf("%s: data updated", server.Key)
			}
		}(server, sh)
	}

	wg.Wait()
}

func (h *handler) updateServersHistory() {
	servers := []*models.Server{}
	now := time.Now()
	t1 := time.Date(now.Year(), now.Month(), now.Day(), 0, 30, 0, 0, time.UTC)
	err := h.db.
		Model(&servers).
		Where("status = ? AND (history_updated_at < ? OR history_updated_at IS NULL)", models.ServerStatusOpen, t1).
		Select()
	if err != nil {
		log.Println(errors.Wrap(err, "updateServersHistory"))
		return
	}

	var wg sync.WaitGroup
	max := runtime.NumCPU() * 5
	count := 0

	for _, server := range servers {
		if count >= max {
			wg.Wait()
			count = 0
		}
		sh := &updateServerHistoryHandler{
			db:     h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server: server,
		}
		count++
		wg.Add(1)
		go func(server *models.Server, sh *updateServerHistoryHandler) {
			defer wg.Done()
			log.Printf("%s: updating history", server.Key)
			if err := sh.update(); err != nil {
				log.Println(errors.Wrap(err, server.Key))
				return
			} else {
				log.Printf("%s: history updated", server.Key)
			}
		}(server, sh)
	}

	wg.Wait()
}

func (h *handler) updateServersStats(t time.Time) error {
	servers := []*models.Server{}
	err := h.db.
		Model(&servers).
		Where("status = ?  AND (stats_updated_at < ? OR stats_updated_at IS NULL)", models.ServerStatusOpen, t).
		Select()
	if err != nil {
		return errors.Wrap(err, "updateServersStats")
	}

	var wg sync.WaitGroup
	max := runtime.NumCPU() * 5
	count := 0

	for _, server := range servers {
		if count >= max {
			wg.Wait()
			count = 0
		}
		sh := &updateServerStatsHandler{
			db:     h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server: server,
		}
		count++
		wg.Add(1)
		go func(server *models.Server, sh *updateServerStatsHandler) {
			defer wg.Done()
			log.Printf("%s: updating stats", server.Key)
			if err := sh.update(); err != nil {
				log.Println(errors.Wrap(err, server.Key))
				return
			} else {
				log.Printf("%s: stats updated", server.Key)
			}
		}(server, sh)
	}

	wg.Wait()
	return nil
}

func (h *handler) updateStats() {
	now := time.Now()
	t1 := time.Date(now.Year(), now.Month(), now.Day(), 1, 30, 0, 0, time.UTC)
	if err := h.updateServersStats(t1); err != nil {
		log.Println(err)
		return
	}
}

func (h *handler) vacuumDatabase() {
	servers := []*models.Server{}
	err := h.db.
		Model(&servers).
		Select()
	if err != nil {
		log.Fatal(errors.Wrap(err, "vacuumDatabase"))
		return
	}

	var wg sync.WaitGroup
	max := runtime.NumCPU() * 5
	count := 0

	for _, server := range servers {
		if count >= max {
			wg.Wait()
			count = 0
		}
		sh := &vacuumServerDBHandler{
			db: h.db.WithParam("SERVER", pg.Safe(server.Key)),
		}
		count++
		wg.Add(1)
		go func(server *models.Server, sh *vacuumServerDBHandler) {
			defer wg.Done()
			log.Printf("%s: vacuuming database", server.Key)
			if err := sh.vacuum(); err != nil {
				log.Println(errors.Wrap(err, server.Key))
				return
			} else {
				log.Printf("%s: database vacuumed", server.Key)
			}
		}(server, sh)
	}

	wg.Wait()
}
