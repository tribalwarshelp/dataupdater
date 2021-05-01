package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/taskq/v3"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
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
	TaskNameDeleteNonExistentVillages       = "deleteNonExistentVillages"
	TaskNameServerDeleteNonExistentVillages = "serverDeleteNonExistentVillages"
	defaultRetryLimit                       = 3
)

var log = logrus.WithField("package", "cron/tasks")

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
	options := []*taskq.TaskOptions{
		{
			Name:    TaskNameLoadVersionsAndUpdateServerData,
			Handler: (&taskLoadVersionsAndUpdateServerData{t}).execute,
		},
		{
			Name:    TaskNameLoadServersAndUpdateData,
			Handler: (&taskLoadServersAndUpdateData{t}).execute,
		},
		{
			Name:    TaskNameUpdateServerData,
			Handler: (&taskUpdateServerData{t}).execute,
		},
		{
			Name:    TaskNameVacuum,
			Handler: (&taskVacuum{t}).execute,
		},
		{
			Name:    TaskNameVacuumServerDB,
			Handler: (&taskVacuumServerDB{t}).execute,
		},
		{
			Name:    TaskUpdateEnnoblements,
			Handler: (&taskUpdateEnnoblements{t}).execute,
		},
		{
			Name:    TaskUpdateServerEnnoblements,
			Handler: (&taskUpdateServerEnnoblements{t}).execute,
		},
		{
			Name:    TaskUpdateHistory,
			Handler: (&taskUpdateHistory{t}).execute,
		},
		{
			Name:       TaskUpdateServerHistory,
			RetryLimit: defaultRetryLimit,
			Handler:    (&taskUpdateServerHistory{t}).execute,
		},
		{
			Name:    TaskUpdateStats,
			Handler: (&taskUpdateStats{t}).execute,
		},
		{
			Name:    TaskUpdateServerStats,
			Handler: (&taskUpdateServerStats{t}).execute,
		},
		{
			Name:    TaskNameDeleteNonExistentVillages,
			Handler: (&taskDeleteNonExistentVillages{t}).execute,
		},
		{
			Name:    TaskNameServerDeleteNonExistentVillages,
			Handler: (&taskServerDeleteNonExistentVillages{t}).execute,
		},
	}
	for _, taskOptions := range options {
		opts := taskOptions
		if opts.RetryLimit == 0 {
			opts.RetryLimit = defaultRetryLimit
		}
		taskq.RegisterTask(opts)
	}

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
