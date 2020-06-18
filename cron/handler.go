package cron

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sync"

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

	if _, err := c.AddFunc("@every 1h", h.updateData); err != nil {
		return err
	}
	go h.updateData()

	return nil
}

func (h *handler) init() error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	models := []interface{}{
		(*models.LangVersion)(nil),
		(*models.Server)(nil),
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

func (h *handler) createSchema(key string) error {
	tx, err := h.db.WithParam("SERVER", pg.Safe(key)).Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if _, err := tx.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", key)); err != nil {
		return err
	}

	models := []interface{}{
		(*models.Tribe)(nil),
		(*models.Player)(nil),
		(*models.Village)(nil),
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

func (h *handler) getServers() ([]*models.Server, map[string]string, error) {
	versions := []*models.LangVersion{}
	if err := h.db.Model(&versions).Select(); err != nil {
		return nil, nil, errors.Wrap(err, "getServers")
	}

	serverKeys := []string{}
	servers := []*models.Server{}
	urls := make(map[string]string)
	for _, version := range versions {
		resp, err := http.Get(fmt.Sprintf("https://%s%s", version.Host, endpointGetServers))
		if err != nil {
			log.Print(errors.Wrapf(err, "Cannot fetch servers from %s", version.Host))
			continue
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Print(errors.Wrapf(err, "Cannot read response body from %s", version.Host))
			continue
		}
		body, err := phpserialize.Decode(string(bodyBytes))
		if err != nil {
			log.Print(errors.Wrapf(err, "Cannot serialize body from %s into go value", version.Host))
			continue
		}
		for serverKey, url := range body.(map[interface{}]interface{}) {
			serverKeyStr := serverKey.(string)
			if err := h.createSchema(serverKeyStr); err != nil {
				log.Print(errors.Wrapf(err, "Cannot create schema for %s", serverKey))
				continue
			}
			serverKeys = append(serverKeys, serverKeyStr)
			urls[serverKeyStr] = url.(string)
			servers = append(servers, &models.Server{
				Key:            serverKeyStr,
				Status:         models.ServerStatusOpen,
				LangVersionTag: version.Tag,
				LangVersion:    version,
			})
		}
	}

	if _, err := h.db.Model(&servers).
		OnConflict("(key) DO UPDATE").
		Set("status = ?", models.ServerStatusOpen).
		Set("lang_version_tag = EXCLUDED.lang_version_tag").
		Returning("*").
		Insert(); err != nil {
		return nil, nil, err
	}

	if _, err := h.db.Model(&models.Server{}).
		Set("status = ?", models.ServerStatusClosed).
		Where("key NOT IN (?)", pg.In(serverKeys)).
		Update(); err != nil {
		return nil, nil, err
	}

	return servers, urls, nil
}

func (h *handler) updateData() {
	servers, urls, err := h.getServers()
	if err != nil {
		log.Println(err.Error())
		return
	}

	var wg sync.WaitGroup
	max := runtime.NumCPU() * 5
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
		}
		count++
		wg.Add(1)
		go func(server *models.Server, sh *updateServerDataHandler) {
			defer wg.Done()
			log.Printf("%s: Updating", server.Key)
			if err := sh.update(); err != nil {
				log.Println(errors.Wrap(err, server.Key))
			} else {
				log.Printf("%s: updated", server.Key)
			}
		}(server, sh)
	}

	wg.Wait()
}
