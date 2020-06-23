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
	serverPGFunctions  = `
		CREATE OR REPLACE FUNCTION ?0.log_tribe_change()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.tribe_id <> OLD.tribe_id THEN
				INSERT INTO ?0.tribe_changes(player_id,old_tribe_id,new_tribe_id,created_at)
				VALUES(OLD.id,OLD.tribe_id,NEW.tribe_id,now());
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;
		CREATE OR REPLACE FUNCTION ?0.get_old_and_new_owner_tribe_id()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.old_owner_id <> 0 THEN
				SELECT tribe_id INTO NEW.old_owner_tribe_id
					FROM ?0.players
					WHERE id = NEW.old_owner_id;
			END IF;
			IF NEW.old_owner_tribe_id IS NULL THEN
				NEW.old_owner_tribe_id = 0;
			END IF;
			IF NEW.new_owner_id <> 0 THEN
				SELECT tribe_id INTO NEW.new_owner_tribe_id
					FROM ?0.players
					WHERE id = NEW.new_owner_id;
			END IF;
			IF NEW.new_owner_tribe_id IS NULL THEN
				NEW.new_owner_tribe_id = 0;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql VOLATILE;
		CREATE OR REPLACE FUNCTION check_daily_growth()
			RETURNS trigger AS
		$BODY$
		BEGIN
			IF NEW.exist = false THEN
				NEW.daily_growth = 0;
			END IF;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;
		CREATE OR REPLACE FUNCTION ?0.insert_to_player_to_servers()
			RETURNS trigger AS
		$BODY$
		BEGIN
			INSERT INTO player_to_servers(server_key,player_id)
				VALUES('?0', NEW.id)
				ON CONFLICT DO NOTHING;

			RETURN NEW;
		END;
		$BODY$
		LANGUAGE plpgsql;
	`
	serverPGTriggers = `
		DROP TRIGGER IF EXISTS ?0_tribe_changes ON ?0.players;
		CREATE TRIGGER ?0_tribe_changes
			AFTER UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.log_tribe_change();
		DROP TRIGGER IF EXISTS ?0_check_daily_growth ON ?0.players;
		CREATE TRIGGER ?0_check_daily_growth
			BEFORE UPDATE
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE check_daily_growth();
		DROP TRIGGER IF EXISTS ?0_update_ennoblement_old_and_new_owner_tribe_id ON ?0.ennoblements;
		CREATE TRIGGER ?0_update_ennoblement_old_and_new_owner_tribe_id
			BEFORE INSERT
			ON ?0.ennoblements
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.get_old_and_new_owner_tribe_id();
		DROP TRIGGER IF EXISTS ?0_insert_to_player_to_servers ON ?0.players;
		CREATE TRIGGER ?0_insert_to_player_to_servers
			AFTER INSERT
			ON ?0.players
			FOR EACH ROW
			EXECUTE PROCEDURE ?0.insert_to_player_to_servers();
	`
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
	go func() {
		h.updateServersData()
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
		(*models.Ennoblement)(nil),
		(*models.ServerStats)(nil),
		(*models.TribeHistory)(nil),
		(*models.PlayerHistory)(nil),
		(*models.TribeChange)(nil),
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
		if _, err := tx.Exec(statement, pg.Safe(key)); err != nil {
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
			if err := h.createSchema(serverKeyStr); err != nil {
				log.Print(errors.Wrapf(err, "Cannot create schema for %s", serverKey))
				continue
			}
			serverKeys = append(serverKeys, serverKeyStr)
			urls[serverKeyStr] = url.(string)
			servers = append(servers, &models.Server{
				Key:            serverKeyStr,
				Status:         models.ServerStatusOpen,
				LangVersionTag: langVersion.Tag,
				LangVersion:    langVersion,
			})
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
		Where("status = ? AND (stats_updated_at < ? OR stats_updated_at IS NULL)", models.ServerStatusOpen, t).
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
