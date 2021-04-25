package cron

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/cron/queue"
)

var log = logrus.WithField("package", "cron")

type Config struct {
	DB                   *pg.DB
	MaxConcurrentWorkers int
	RunOnStartup         bool
	Queue                queue.Queue
}

func Attach(c *cron.Cron, cfg Config) error {
	if cfg.DB == nil {
		return fmt.Errorf("cfg.DB cannot be nil, expected *pg.DB")
	}
	if cfg.Queue == nil {
		return fmt.Errorf("cfg.Queue cannot be nil, expected queue.Queue")
	}

	h := &handler{
		db:                   cfg.DB,
		maxConcurrentWorkers: cfg.MaxConcurrentWorkers,
		queue:                cfg.Queue,
	}
	if err := h.init(); err != nil {
		return err
	}

	var versions []*models.Version
	if err := cfg.DB.Model(&versions).DistinctOn("timezone").Select(); err != nil {
		return err
	}

	var updateHistoryFuncs []func()
	var updateStatsFuncs []func()
	for _, version := range versions {
		updateHistory := createFnWithTimezone(version.Timezone, h.updateHistory)
		updateHistoryFuncs = append(updateHistoryFuncs, updateHistory)

		updateStats := createFnWithTimezone(version.Timezone, h.updateStats)
		updateStatsFuncs = append(updateStatsFuncs, updateStats)

		if _, err := c.AddFunc(fmt.Sprintf("CRON_TZ=%s 30 1 * * *", version.Timezone), updateHistory); err != nil {
			return err
		}
		if _, err := c.AddFunc(fmt.Sprintf("CRON_TZ=%s 45 1 * * *", version.Timezone), updateStats); err != nil {
			return err
		}
	}
	if _, err := c.AddFunc("0 * * * *", h.updateServerData); err != nil {
		return err
	}
	if _, err := c.AddFunc("20 1 * * *", h.vacuumDatabase); err != nil {
		return err
	}
	if _, err := c.AddFunc("@every 1m", h.updateEnnoblements); err != nil {
		return err
	}
	if cfg.RunOnStartup {
		go func() {
			h.updateEnnoblements()
			//h.updateServerData()
			//h.vacuumDatabase()
			//for _, fn := range updateHistoryFuncs {
			//	go fn()
			//}
			//for _, fn := range updateStatsFuncs {
			//	go fn()
			//}
		}()
	}

	return nil
}
