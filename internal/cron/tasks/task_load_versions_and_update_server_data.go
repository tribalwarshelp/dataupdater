package tasks

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
		err = errors.Wrap(err, "taskLoadVersionsAndUpdateServerData.execute: couldn't load versions")
		log.Fatal(err)
		return err
	}
	for _, version := range versions {
		t.queue.Add(queue.MainQueue, Get(TaskNameLoadServersAndUpdateData).WithArgs(context.Background(), version))
	}
	log.Debug("taskLoadVersionsAndUpdateServerData.execute: Versions have been loaded")
	return nil
}
