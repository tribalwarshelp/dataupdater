package cron

import (
	"context"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/tw/twmodel"

	"github.com/robfig/cron/v3"

	"github.com/tribalwarshelp/cron/pkg/cron/queue"
)

type Cron struct {
	*cron.Cron
	queue     queue.Queue
	db        *pg.DB
	runOnInit bool
	log       logrus.FieldLogger
}

func New(cfg *Config) (*Cron, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	q, err := initializeQueue(cfg)
	if err != nil {
		return nil, err
	}
	log := logrus.WithField("package", "pkg/cron")
	c := &Cron{
		Cron: cron.New(cron.WithChain(
			cron.SkipIfStillRunning(
				cron.PrintfLogger(log),
			),
		)),
		queue:     q,
		db:        cfg.DB,
		runOnInit: cfg.RunOnInit,
		log:       log,
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cron) init() error {
	var versions []*twmodel.Version
	if err := c.db.Model(&versions).DistinctOn("timezone").Select(); err != nil {
		return errors.Wrap(err, "couldn't load versions")
	}

	var updateHistoryFuncs []func()
	var updateStatsFuncs []func()
	for _, version := range versions {
		updateHistory := createFnWithTimezone(version.Timezone, c.updateHistory)
		updateHistoryFuncs = append(updateHistoryFuncs, updateHistory)

		updateStats := createFnWithTimezone(version.Timezone, c.updateStats)
		updateStatsFuncs = append(updateStatsFuncs, updateStats)

		if _, err := c.AddFunc(fmt.Sprintf("CRON_TZ=%s 30 1 * * *", version.Timezone), updateHistory); err != nil {
			return err
		}
		if _, err := c.AddFunc(fmt.Sprintf("CRON_TZ=%s 45 1 * * *", version.Timezone), updateStats); err != nil {
			return err
		}
	}
	if _, err := c.AddFunc("0 * * * *", c.updateServerData); err != nil {
		return err
	}
	if _, err := c.AddFunc("20 1 * * *", c.vacuumDatabase); err != nil {
		return err
	}
	if _, err := c.AddFunc("10 1 * * *", c.deleteNonExistentVillages); err != nil {
		return err
	}
	if _, err := c.AddFunc("@every 1m", c.updateEnnoblements); err != nil {
		return err
	}
	if c.runOnInit {
		go func() {
			c.updateServerData()
			c.vacuumDatabase()
			for _, fn := range updateHistoryFuncs {
				fn()
			}
			for _, fn := range updateStatsFuncs {
				fn()
			}
		}()
	}
	return nil
}

func (c *Cron) Start(ctx context.Context) error {
	if err := c.queue.Start(ctx); err != nil {
		return err
	}
	c.Cron.Start()
	return nil
}

func (c *Cron) Stop() error {
	c.Cron.Stop()
	if err := c.queue.Close(); err != nil {
		return err
	}
	return nil
}

func (c *Cron) updateServerData() {
	err := c.queue.Add(queue.Main, queue.GetTask(queue.LoadVersionsAndUpdateServerData).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.updateServerData", queue.LoadVersionsAndUpdateServerData, err)
	}
}

func (c *Cron) updateEnnoblements() {
	err := c.queue.Add(queue.Ennoblements, queue.GetTask(queue.UpdateEnnoblements).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.updateEnnoblements", queue.UpdateEnnoblements, err)
	}
}

func (c *Cron) updateHistory(timezone string) {
	err := c.queue.Add(queue.Main, queue.GetTask(queue.UpdateHistory).WithArgs(context.Background(), timezone))
	if err != nil {
		c.logError("Cron.updateHistory", queue.UpdateHistory, err)
	}
}

func (c *Cron) updateStats(timezone string) {
	err := c.queue.Add(queue.Main, queue.GetTask(queue.UpdateStats).WithArgs(context.Background(), timezone))
	if err != nil {
		c.logError("Cron.updateStats", queue.UpdateStats, err)
	}
}

func (c *Cron) vacuumDatabase() {
	err := c.queue.Add(queue.Main, queue.GetTask(queue.Vacuum).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.vacuumDatabase", queue.Vacuum, err)
	}
}

func (c *Cron) deleteNonExistentVillages() {
	err := c.queue.Add(queue.Main, queue.GetTask(queue.DeleteNonExistentVillages).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.deleteNonExistentVillages", queue.DeleteNonExistentVillages, err)
	}
}

func (c *Cron) logError(prefix string, taskName string, err error) {
	c.log.Error(
		errors.Wrapf(
			err,
			"%s: Couldn't add the task '%s' to the queue",
			prefix,
			taskName,
		),
	)
}

func initializeQueue(cfg *Config) (queue.Queue, error) {
	q, err := queue.New(&queue.Config{
		WorkerLimit: cfg.WorkerLimit,
		Redis:       cfg.Redis,
		DB:          cfg.DB,
	})
	return q, errors.Wrap(err, "couldn't initialize a queue")
}

func createFnWithTimezone(timezone string, fn func(timezone string)) func() {
	return func() {
		fn(timezone)
	}
}
