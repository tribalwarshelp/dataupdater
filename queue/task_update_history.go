package queue

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"time"
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
	var servers []*twmodel.Server
	err = t.db.
		Model(&servers).
		Where(
			"status = ? AND (history_updated_at IS NULL OR history_updated_at < ?) AND timezone = ?",
			twmodel.ServerStatusOpen,
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
		err := t.queue.Add(GetTask(UpdateServerHistory).WithArgs(context.Background(), timezone, server))
		if err != nil {
			log.
				WithField("key", server.Key).
				Warn(
					errors.Wrapf(
						err,
						"taskUpdateHistory.execute: %s: Couldn't add the task '%s' for this server",
						server.Key,
						UpdateServerHistory,
					),
				)
		}
	}
	return nil
}
