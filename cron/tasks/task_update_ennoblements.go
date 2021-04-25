package tasks

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/cron/queue"
)

type taskUpdateEnnoblements struct {
	*task
}

func (t *taskUpdateEnnoblements) execute() error {
	var servers []*models.Server
	err := t.db.
		Model(&servers).
		Relation("Version").
		Where("status = ?", models.ServerStatusOpen).
		Select()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateEnnoblements.execute")
		log.Errorln(err)
		return err
	}
	log.Debug("Updating ennoblements...")
	for _, server := range servers {
		s := server
		t.queue.Add(
			queue.MainQueue,
			Get(TaskNameVacuumServerDB).
				WithArgs(context.Background(), fmt.Sprintf("https://%s.%s", server.Key, server.Version.Host), s),
		)
	}
	return nil
}
