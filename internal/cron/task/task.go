package task

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/taskq/v3"
	"sync"
	"time"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

const (
	LoadVersionsAndUpdateServerData = "loadVersionsAndUpdateServerData"
	LoadServersAndUpdateData        = "loadServersAndUpdateData"
	UpdateServerData                = "updateServerData"
	Vacuum                          = "vacuum"
	VacuumServerDB                  = "vacuumServerDB"
	UpdateEnnoblements              = "updateEnnoblements"
	UpdateServerEnnoblements        = "updateServerEnnoblements"
	UpdateHistory                   = "updateHistory"
	UpdateServerHistory             = "updateServerHistory"
	UpdateStats                     = "updateStats"
	UpdateServerStats               = "updateServerStats"
	DeleteNonExistentVillages       = "deleteNonExistentVillages"
	ServerDeleteNonExistentVillages = "serverDeleteNonExistentVillages"
	defaultRetryLimit               = 3
)

var log = logrus.WithField("package", "internal/cron/task")

type task struct {
	db              *pg.DB
	queue           queue.Queue
	cachedLocations sync.Map
}

func (t *task) loadLocation(timezone string) (*time.Location, error) {
	val, ok := t.cachedLocations.Load(timezone)
	if ok {
		return val.(*time.Location), nil
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, errors.Wrap(err, "task.loadLocation")
	}
	t.cachedLocations.Store(timezone, location)
	return location, nil
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
			Name:    LoadVersionsAndUpdateServerData,
			Handler: (&taskLoadVersionsAndUpdateServerData{t}).execute,
		},
		{
			Name:    LoadServersAndUpdateData,
			Handler: (&taskLoadServersAndUpdateData{t}).execute,
		},
		{
			Name:    UpdateServerData,
			Handler: (&taskUpdateServerData{t}).execute,
		},
		{
			Name:    Vacuum,
			Handler: (&taskVacuum{t}).execute,
		},
		{
			Name:    VacuumServerDB,
			Handler: (&taskVacuumServerDB{t}).execute,
		},
		{
			Name:    UpdateEnnoblements,
			Handler: (&taskUpdateEnnoblements{t}).execute,
		},
		{
			Name:    UpdateServerEnnoblements,
			Handler: (&taskUpdateServerEnnoblements{t}).execute,
		},
		{
			Name:    UpdateHistory,
			Handler: (&taskUpdateHistory{t}).execute,
		},
		{
			Name:       UpdateServerHistory,
			RetryLimit: defaultRetryLimit,
			Handler:    (&taskUpdateServerHistory{t}).execute,
		},
		{
			Name:    UpdateStats,
			Handler: (&taskUpdateStats{t}).execute,
		},
		{
			Name:    UpdateServerStats,
			Handler: (&taskUpdateServerStats{t}).execute,
		},
		{
			Name:    DeleteNonExistentVillages,
			Handler: (&taskDeleteNonExistentVillages{t}).execute,
		},
		{
			Name:    ServerDeleteNonExistentVillages,
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
