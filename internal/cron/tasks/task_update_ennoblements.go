package tasks

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
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
	log.WithField("numberOfServers", len(servers)).Info("taskUpdateEnnoblements.execute: Update of the ennoblements has started...")
	for _, server := range servers {
		s := server
		err := t.queue.Add(
			queue.EnnoblementsQueue,
			Get(TaskUpdateServerEnnoblements).
				WithArgs(context.Background(), fmt.Sprintf("https://%s.%s", server.Key, server.Version.Host), s),
		)
		if err != nil {
			log.Warn(
				errors.Wrapf(
					err,
					"taskUpdateEnnoblements.execute: %s: Couldn't add the task '%s' for this server",
					server.Key,
					TaskUpdateServerEnnoblements,
				),
			)
		}
	}
	return nil
}
