package task

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

type taskLoadVersionsAndUpdateServerData struct {
	*task
}

func (t *taskLoadVersionsAndUpdateServerData) execute() error {
	var versions []*twmodel.Version
	log.Debug("taskLoadVersionsAndUpdateServerData.execute: Loading versions...")
	if err := t.db.Model(&versions).Relation("SpecialServers").Select(); err != nil {
		err = errors.Wrap(err, "taskLoadVersionsAndUpdateServerData.execute: Couldn't load versions")
		log.Fatal(err)
		return err
	}
	log.Debug("taskLoadVersionsAndUpdateServerData.execute: Versions have been loaded")
	for _, version := range versions {
		err := t.queue.Add(queue.Main, Get(LoadServersAndUpdateData).WithArgs(context.Background(), version))
		if err != nil {
			log.
				WithField("code", version.Code).
				Warn(
					errors.Wrapf(
						err,
						"taskLoadVersionsAndUpdateServerData.execute: %s: Couldn't add the task '%s' for this version",
						version.Code,
						LoadServersAndUpdateData,
					),
				)
		}
	}
	return nil
}
