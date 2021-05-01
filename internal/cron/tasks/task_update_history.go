package tasks

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
	"time"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

type taskUpdateHistory struct {
	*task
}

func (t *taskUpdateHistory) execute(timezone string) error {
	entry := log.WithField("timezone", timezone)
	location, err := t.loadLocation(timezone)
	if err != nil {
		err = errors.Wrap(err, "taskUpdateHistory.execute")
		entry.Error(err)
		return err
	}
	year, month, day := time.Now().In(location).Date()
	date := time.Date(year, month, day, 1, 30, 0, 0, location)
	var servers []*models.Server
	err = t.db.
		Model(&servers).
		Where(
			"status = ? AND (history_updated_at IS NULL OR history_updated_at < ?) AND timezone = ?",
			models.ServerStatusOpen,
			date,
			timezone,
		).
		Relation("Version").
		Select()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateHistory.execute")
		entry.Errorln(err)
		return err
	}
	entry.
		WithField("numberOfServers", len(servers)).
		Info("taskUpdateHistory.execute: Update of the history has started")
	for _, server := range servers {
		s := server
		err := t.queue.Add(queue.MainQueue, Get(TaskUpdateServerHistory).WithArgs(context.Background(), timezone, s))
		if err != nil {
			log.Warn(
				errors.Wrapf(
					err,
					"taskUpdateHistory.execute: %s: Couldn't add the task '%s' for this server",
					server.Key,
					TaskUpdateServerHistory,
				),
			)
		}
	}
	return nil
}
