package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/taskq/v3"

	"github.com/tribalwarshelp/cron/cron/queue"
)

const (
	TaskNameLoadVersionsAndUpdateServerData = "loadVersionsAndUpdateServerData"
	TaskNameLoadServersAndUpdateData        = "loadServersAndUpdateData"
	TaskNameUpdateServerData                = "updateServerData"
	TaskNameVacuum                          = "vacuum"
	TaskNameVacuumServerDB                  = "vacuumServerDB"
	TaskUpdateEnnoblements                  = "updateEnnoblements"
	TaskUpdateServerEnnoblements            = "updateServerEnnoblements"
	TaskUpdateHistory                       = "updateHistory"
	TaskUpdateServerHistory                 = "updateServerHistory"
	TaskUpdateStats                         = "updateStats"
	TaskUpdateServerStats                   = "updateServerStats"
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
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskNameVacuum,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskVacuum{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskNameVacuumServerDB,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskVacuumServerDB{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskUpdateEnnoblements,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskUpdateEnnoblements{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskUpdateServerEnnoblements,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskUpdateServerEnnoblements{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskUpdateHistory,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskUpdateHistory{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskUpdateServerHistory,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskUpdateServerHistory{t}).execute,
	})
	taskq.RegisterTask(&taskq.TaskOptions{
		Name:       TaskUpdateStats,
		RetryLimit: defaultRetryLimit,
		Handler:    (&taskUpdateStats{t}).execute,
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
