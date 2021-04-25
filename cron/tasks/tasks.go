package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/taskq/v3"

	"github.com/tribalwarshelp/cron/cron/queue"
)

const (
	TaskNameLoadVersionsAndUpdateServerData = "loadVersions"
	TaskNameLoadServersAndUpdateData        = "loadServers"
	TaskNameUpdateServerData                = "updateServerData"
	defaultRetryLimit                       = 3
)

var log = logrus.WithField("package", "tasks")

type Config struct {
	DB    *pg.DB
	Queue queue.Queue
}

func RegisterTasks(cfg *Config) error {
	if err := validateConfig(cfg); err != nil {
		return errors.Wrap(err, "RegisterTasks")
	}

	t := &task{
		db:    cfg.DB,
		queue: cfg.Queue,
	}
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskNameLoadVersionsAndUpdateServerData,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskLoadVersionsAndUpdateServerData{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskNameLoadServersAndUpdateData,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskLoadServersAndUpdateData{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskNameUpdateServerData,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskUpdateServerData{t}).execute,
	})

	return nil
}

func Get(taskName string) *taskq.Task {
	return taskq.Tasks.Get(taskName)
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("Config hasn't been provided")
	}
	if cfg.DB == nil {
		return errors.New("cfg.DB is required")
	}
	if cfg.Queue == nil {
		return errors.New("cfg.Queue is required")
	}
	return nil
}
