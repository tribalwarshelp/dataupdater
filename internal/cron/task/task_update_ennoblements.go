package task

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"github.com/tribalwarshelp/shared/tw/twurlbuilder"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

type taskUpdateEnnoblements struct {
	*task
}

func (t *taskUpdateEnnoblements) execute() error {
	var servers []*twmodel.Server
	err := t.db.
		Model(&servers).
		Relation("Version").
		Where("status = ?", twmodel.ServerStatusOpen).
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
			queue.Ennoblements,
			Get(UpdateServerEnnoblements).
				WithArgs(context.Background(), twurlbuilder.BuildServerURL(server.Key, server.Version.Host), s),
		)
		if err != nil {
			log.Warn(
				errors.Wrapf(
					err,
					"taskUpdateEnnoblements.execute: %s: Couldn't add the task '%s' for this server",
					server.Key,
					UpdateServerEnnoblements,
				),
			)
		}
	}
	return nil
}
