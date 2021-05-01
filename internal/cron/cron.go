package cron

import (
	"context"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/robfig/cron/v3"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
	"github.com/tribalwarshelp/cron/internal/cron/tasks"
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
	c := &Cron{
		Cron:      cron.New(cfg.Opts...),
		queue:     q,
		db:        cfg.DB,
		runOnInit: cfg.RunOnInit,
		log:       logrus.WithField("package", "cron"),
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cron) init() error {
	var versions []*models.Version
	if err := c.db.Model(&versions).DistinctOn("timezone").Select(); err != nil {
		return errors.Wrap(err, "Cron.init: couldn't load versions")
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
		return errors.Wrap(err, "Cron.Start")
	}
	c.Cron.Start()
	return nil
}

func (c *Cron) Stop() error {
	c.Cron.Stop()
	if err := c.queue.Close(); err != nil {
		return errors.Wrap(err, "Cron.Stop")
	}
	return nil
}

func (c *Cron) updateServerData() {
	err := c.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameLoadVersionsAndUpdateServerData).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.updateServerData", tasks.TaskNameLoadVersionsAndUpdateServerData, err)
	}
}

func (c *Cron) updateEnnoblements() {
	err := c.queue.Add(queue.EnnoblementsQueue, tasks.Get(tasks.TaskUpdateEnnoblements).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.updateEnnoblements", tasks.TaskUpdateEnnoblements, err)
	}
}

func (c *Cron) updateHistory(timezone string) {
	err := c.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskUpdateHistory).WithArgs(context.Background(), timezone))
	if err != nil {
		c.logError("Cron.updateHistory", tasks.TaskUpdateHistory, err)
	}
}

func (c *Cron) updateStats(timezone string) {
	err := c.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskUpdateStats).WithArgs(context.Background(), timezone))
	if err != nil {
		c.logError("Cron.updateStats", tasks.TaskUpdateStats, err)
	}
}

func (c *Cron) vacuumDatabase() {
	err := c.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameVacuum).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.vacuumDatabase", tasks.TaskNameVacuum, err)
	}
}

func (c *Cron) deleteNonExistentVillages() {
	err := c.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameDeleteNonExistentVillages).WithArgs(context.Background()))
	if err != nil {
		c.logError("Cron.deleteNonExistentVillages", tasks.TaskNameDeleteNonExistentVillages, err)
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
	})
	if err != nil {
		return nil, errors.Wrap(err, "initializeQueue: Couldn't create the task queue")
	}
	err = tasks.RegisterTasks(&tasks.Config{
		DB:    cfg.DB,
		Queue: q,
	})
	if err != nil {
		return nil, errors.Wrap(err, "initializeQueue: Couldn't create the task queue")
	}
	return q, nil
}

func createFnWithTimezone(timezone string, fn func(timezone string)) func() {
	return func() {
		fn(timezone)
	}
}