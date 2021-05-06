package tasks

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

type taskVacuum struct {
	*task
}

func (t *taskVacuum) execute() error {
	var servers []*twmodel.Server
	err := t.db.
		Model(&servers).
		Select()
	if err != nil {
		err = errors.Wrap(err, "taskVacuum.execute")
		log.Errorln(err)
		return err
	}
	log.Infof("taskVacuum.execute: Start database vacumming...")
	for _, server := range servers {
		s := server
		err := t.queue.Add(queue.MainQueue, Get(TaskNameVacuumServerDB).WithArgs(context.Background(), s))
		if err != nil {
			log.Warn(
				errors.Wrapf(
					err,
					"taskVacuum.execute: %s: Couldn't add the task '%s' for this server",
					server.Key,
					TaskUpdateServerEnnoblements,
				),
			)
		}
	}
	return nil
}
