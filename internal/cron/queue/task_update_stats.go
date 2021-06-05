package queue

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"time"
)

type taskUpdateStats struct {
	*task
}

func (t *taskUpdateStats) execute(timezone string) error {
	entry := log.WithField("timezone", timezone)
	location, err := t.loadLocation(timezone)
	if err != nil {
		err = errors.Wrap(err, "taskUpdateStats.execute")
		entry.Error(err)
		return err
	}
	year, month, day := time.Now().In(location).Date()
	date := time.Date(year, month, day, 1, 45, 0, 0, location)
	var servers []*twmodel.Server
	err = t.db.
		Model(&servers).
		Where(
			"status = ? AND (stats_updated_at IS NULL OR stats_updated_at < ?) AND timezone = ?",
			twmodel.ServerStatusOpen,
			date,
			location.String(),
		).
		Relation("Version").
		Select()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateStats.execute")
		entry.Errorln(err)
		return err
	}
	entry.
		WithField("numberOfServers", len(servers)).
		Info("taskUpdateStats.execute: Update of the stats has started")
	for _, server := range servers {
		err := t.queue.Add(Main, GetTask(UpdateServerStats).WithArgs(context.Background(), timezone, server))
		if err != nil {
			log.
				WithField("key", server.Key).
				Warn(
					errors.Wrapf(
						err,
						"taskUpdateStats.execute: %s: Couldn't add the task '%s' for this server",
						server.Key,
						UpdateServerStats,
					),
				)
		}
	}
	return nil
}
