package task

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
	log.Infof("taskVacuum.execute: The database vacumming process has started...")
	for _, server := range servers {
		err := t.queue.Add(queue.Main, Get(VacuumServerDB).WithArgs(context.Background(), server))
		if err != nil {
			log.
				WithField("key", server.Key).
				Warn(
					errors.Wrapf(
						err,
						"taskVacuum.execute: %s: Couldn't add the task '%s' for this server",
						server.Key,
						UpdateServerEnnoblements,
					),
				)
		}
	}
	return nil
}
