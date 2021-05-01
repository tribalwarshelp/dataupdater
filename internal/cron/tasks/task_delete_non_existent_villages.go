package tasks

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

type taskDeleteNonExistentVillages struct {
	*task
}

func (t *taskDeleteNonExistentVillages) execute() error {
	var servers []*models.Server
	err := t.db.
		Model(&servers).
		Relation("Version").
		Where("status = ?", models.ServerStatusOpen).
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
		s := server
		err := t.queue.Add(
			queue.MainQueue,
			Get(TaskNameServerDeleteNonExistentVillages).
				WithArgs(
					context.Background(),
					fmt.Sprintf("https://%s.%s", server.Key, server.Version.Host),
					s,
				),
		)
		if err != nil {
			log.Warn(
				errors.Wrapf(
					err,
					"taskDeleteNonExistentVillages.execute: %s: Couldn't add the task '%s' for this server",
					server.Key,
					TaskNameServerDeleteNonExistentVillages,
				),
			)
		}
	}
	return nil
}
