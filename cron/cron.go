package cron

import (
	"fmt"

	"github.com/tribalwarshelp/shared/utils"

	"github.com/go-pg/pg/v10"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("package", "cron")

type Config struct {
	DB                   *pg.DB
	MaxConcurrentWorkers int
	RunOnStartup         bool
}

func Attach(c *cron.Cron, cfg Config) error {
	if cfg.DB == nil {
		return fmt.Errorf("cfg.DB cannot be nil, expected go-pg database")
	}

	h := &handler{cfg.DB, cfg.MaxConcurrentWorkers}
	if err := h.init(); err != nil {
		return err
	}

	updateServerData := utils.TrackExecutionTime(log, h.updateServerData, "updateServerData")
	updateHistory := utils.TrackExecutionTime(log, h.updateHistory, "updateHistory")
	vacuumDatabase := utils.TrackExecutionTime(log, h.vacuumDatabase, "vacuumDatabase")
	updateStats := utils.TrackExecutionTime(log, h.updateStats, "updateStats")
	if _, err := c.AddFunc("0 * * * *", updateServerData); err != nil {
		return err
	}
	if _, err := c.AddFunc("30 0 * * *", updateHistory); err != nil {
		return err
	}
	if _, err := c.AddFunc("30 1 * * *", vacuumDatabase); err != nil {
		return err
	}
	if _, err := c.AddFunc("30 2 * * *", updateStats); err != nil {
		return err
	}
	if cfg.RunOnStartup {
		go func() {
			updateServerData()
			vacuumDatabase()
			updateHistory()
			updateStats()
		}()
	}

	return nil
}
