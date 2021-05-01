package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"sync"
	"time"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

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
