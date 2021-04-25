package tasks

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/cron/queue"
)

type taskLoadVersions struct {
	*task
}

func (t *taskLoadVersions) execute() error {
	var versions []*models.Version
	log.Debug("taskLoadVersions.execute: Loading versions...")
	if err := t.db.Model(&versions).Relation("SpecialServers").Select(); err != nil {
		err = errors.Wrap(err, "taskLoadVersions.execute: couldn't load versions")
		log.Fatal(err)
		return err
	}
	for _, version := range versions {
		t.queue.Add(queue.MainQueue, Get(TaskNameLoadServers).WithArgs(context.Background(), version))
	}
	log.Debug("taskLoadVersions.execute: Versions have been loaded")
	return nil
}
