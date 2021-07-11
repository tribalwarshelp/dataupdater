package queue

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/vmihailenco/taskq/v3"
	"sync"
	"time"
)

const (
	LoadVersionsAndUpdateServerData = "loadVersionsAndUpdateServerData"
	LoadServersAndUpdateData        = "loadServersAndUpdateData"
	UpdateServerData                = "updateServerData"
	Vacuum                          = "vacuum"
	VacuumServerData                = "vacuumServerData"
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

type task struct {
	db              *pg.DB
	queue           Queue
	cachedLocations sync.Map
}

func (t *task) loadLocation(timezone string) (*time.Location, error) {
	val, ok := t.cachedLocations.Load(timezone)
	if ok {
		return val.(*time.Location), nil
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't load location for the timezone '"+timezone+"'")
	}
	t.cachedLocations.Store(timezone, location)
	return location, nil
}

func registerTasks(cfg *registerTasksConfig) error {
	if err := validateRegisterTasksConfig(cfg); err != nil {
		return errors.Wrap(err, "config is invalid")
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
			Name:    VacuumServerData,
			Handler: (&taskVacuumServerData{t}).execute,
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

func GetTask(taskName string) *taskq.Task {
	return taskq.Tasks.Get(taskName)
}
