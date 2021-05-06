package tasks

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/tw/twdataloader"
	"github.com/tribalwarshelp/shared/tw/twmodel"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
	"github.com/tribalwarshelp/cron/internal/postgres"
)

type taskLoadServersAndUpdateData struct {
	*task
}

type serverWithURL struct {
	*twmodel.Server `pg:",inherit"`
	url             string
}

func (t *taskLoadServersAndUpdateData) execute(version *twmodel.Version) error {
	if err := t.validatePayload(version); err != nil {
		log.Debug(err)
		return nil
	}
	entry := log.WithField("host", version.Host)
	entry.Infof("taskLoadServersAndUpdateData.execute: %s: Loading servers", version.Host)
	loadedServers, err := twdataloader.
		NewVersionDataLoader(&twdataloader.VersionDataLoaderConfig{
			Host:   version.Host,
			Client: newHTTPClient(),
		}).
		LoadServers()
	if err != nil {
		log.Errorln(err)
		return err
	}

	var serverKeys []string
	var servers []*serverWithURL
	for _, loadedServer := range loadedServers {
		if version.SpecialServers.Contains(loadedServer.Key) {
			continue
		}
		server := &twmodel.Server{
			Key:         loadedServer.Key,
			Status:      twmodel.ServerStatusOpen,
			VersionCode: version.Code,
			Version:     version,
		}
		if err := postgres.CreateSchema(t.db, server); err != nil {
			logrus.Warn(errors.Wrapf(err, "taskLoadServersAndUpdateData.execute: %s: couldn't create the schema", server.Key))
			continue
		}
		servers = append(servers, &serverWithURL{
			Server: server,
			url:    loadedServer.URL,
		})
		serverKeys = append(serverKeys, server.Key)
	}

	if len(servers) > 0 {
		if _, err := t.db.Model(&servers).
			OnConflict("(key) DO UPDATE").
			Set("status = ?", twmodel.ServerStatusOpen).
			Set("version_code = EXCLUDED.version_code").
			Returning("*").
			Insert(); err != nil {
			err = errors.Wrap(err, "taskLoadServersAndUpdateData.execute: couldn't insert/update servers")
			logrus.Error(err)
			return err
		}
	}

	if _, err := t.db.Model(&twmodel.Server{}).
		Set("status = ?", twmodel.ServerStatusClosed).
		Where("key NOT IN (?) AND version_code = ?", pg.In(serverKeys), version.Code).
		Update(); err != nil {
		err = errors.Wrap(err, "taskLoadServersAndUpdateData.execute: couldn't update server statuses")
		logrus.Error(err)
		return err
	}

	for _, server := range servers {
		t.queue.Add(queue.MainQueue, Get(TaskNameUpdateServerData).WithArgs(context.Background(), server.url, server.Server))
	}

	entry.Infof("%s: Servers have been loaded", version.Host)
	return nil
}

func (t *taskLoadServersAndUpdateData) validatePayload(version *twmodel.Version) error {
	if version == nil {
		return errors.New("taskLoadServersAndUpdateData.validatePayload: Expected *twmodel.Version, got nil")
	}
	return nil
}
