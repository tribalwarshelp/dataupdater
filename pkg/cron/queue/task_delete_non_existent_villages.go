package queue

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"github.com/tribalwarshelp/shared/tw/twurlbuilder"
)

type taskDeleteNonExistentVillages struct {
	*task
}

func (t *taskDeleteNonExistentVillages) execute() error {
	var servers []*twmodel.Server
	err := t.db.
		Model(&servers).
		Relation("Version").
		Where("status = ?", twmodel.ServerStatusOpen).
		Relation("Version").
		Select()
	if err != nil {
		err = errors.Wrap(err, "taskDeleteNonExistentVillages.execute")
		log.Errorln(err)
		return err
	}
	log.
		WithField("numberOfServers", len(servers)).
		Info("taskDeleteNonExistentVillages.execute: Servers have been loaded and added to the queue")
	for _, server := range servers {
		err := t.queue.Add(
			Main,
			GetTask(ServerDeleteNonExistentVillages).
				WithArgs(
					context.Background(),
					twurlbuilder.BuildServerURL(server.Key, server.Version.Host),
					server,
				),
		)
		if err != nil {
			log.
				WithField("key", server.Key).
				Warn(
					errors.Wrapf(
						err,
						"taskDeleteNonExistentVillages.execute: %s: Couldn't add the task '%s' for this server",
						server.Key,
						ServerDeleteNonExistentVillages,
					),
				)
		}
	}
	return nil
}
