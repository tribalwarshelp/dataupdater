package tasks

import (
	"context"
	"fmt"
	phpserialize "github.com/Kichiyaki/go-php-serialize"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"io/ioutil"
	"net/http"

	"github.com/tribalwarshelp/cron/internal/cron/queue"
	"github.com/tribalwarshelp/cron/internal/postgres"
)

const (
	endpointGetServers = "/backend/get_servers.php"
)

type taskLoadServersAndUpdateData struct {
	*task
}

func (t *taskLoadServersAndUpdateData) execute(version *twmodel.Version) error {
	if err := t.validatePayload(version); err != nil {
		log.Debug(err)
		return nil
	}
	entry := log.WithField("host", version.Host)
	entry.Infof("taskLoadServersAndUpdateData.execute: %s: Loading servers", version.Host)
	data, err := t.getServers(version)
	if err != nil {
		log.Errorln(err)
		return err
	}

	var serverKeys []string
	var servers []*twmodel.Server
	for serverKey := range data {
		if version.SpecialServers.Contains(serverKey) {
			continue
		}
		server := &twmodel.Server{
			Key:         serverKey,
			Status:      twmodel.ServerStatusOpen,
			VersionCode: version.Code,
			Version:     version,
		}
		if err := postgres.CreateSchema(t.db, server); err != nil {
			logrus.Warn(errors.Wrapf(err, "taskLoadServersAndUpdateData.execute: %s: couldn't create the schema", server.Key))
			continue
		}
		servers = append(servers, server)
		serverKeys = append(serverKeys, serverKey)
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
		s := server
		t.queue.Add(queue.MainQueue, Get(TaskNameUpdateServerData).WithArgs(context.Background(), data[s.Key], s))
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

func (t *taskLoadServersAndUpdateData) getServers(version *twmodel.Version) (map[string]string, error) {
	resp, err := http.Get(fmt.Sprintf("https://%s%s", version.Host, endpointGetServers))
	if err != nil {
		return nil, errors.Wrapf(err, "%s: taskLoadServersAndUpdateData.loadServers couldn't load servers", version.Host)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: taskLoadServersAndUpdateData.loadServers couldn't read the response body", version.Host)
	}
	body, err := phpserialize.Decode(string(bodyBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "%s: taskLoadServersAndUpdateData.loadServers couldn't decode the response body into the go value", version.Host)
	}

	result := make(map[string]string)
	for serverKey, url := range body.(map[interface{}]interface{}) {
		serverKeyStr := serverKey.(string)
		urlStr := url.(string)
		if serverKeyStr != "" && urlStr != "" {
			result[serverKeyStr] = urlStr
		}
	}
	return result, nil
}
