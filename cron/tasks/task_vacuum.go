package tasks

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/cron/queue"
)

type taskVacuum struct {
	*task
}

func (t *taskVacuum) execute() error {
	var servers []*models.Server
	err := t.db.
		Model(&servers).
		Select()
	if err != nil {
		err = errors.Wrap(err, "taskVacuum.execute")
		log.Errorln(err)
		return err
	}
	log.Infof("Start database vacuuming...")
	for _, server := range servers {
		s := server
		t.queue.Add(queue.MainQueue, Get(TaskNameVacuumServerDB).WithArgs(context.Background(), s))
	}
	return nil
}
