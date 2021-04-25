package tasks

import (
	"fmt"
	phpserialize "github.com/Kichiyaki/go-php-serialize"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/models"
	"io/ioutil"
	"net/http"

	"github.com/tribalwarshelp/cron/db"
)

const (
	endpointGetServers = "/backend/get_servers.php"
)

type taskLoadServers struct {
	*task
}

func (t *taskLoadServers) execute(version *models.Version) error {
	if err := t.validatePayload(version); err != nil {
		return nil
	}
	entry := log.WithField("host", version.Host)
	entry.Infof("%s: Loading servers", version.Host)
	data, err := t.getServers(version)
	if err != nil {
		log.Errorln(err)
		return err
	}

	var serverKeys []string
	var servers []*models.Server
	for serverKey := range data {
		if version.SpecialServers.Contains(serverKey) {
			continue
		}
		server := &models.Server{
			Key:         serverKey,
			Status:      models.ServerStatusOpen,
			VersionCode: version.Code,
			Version:     version,
		}
		if err := db.CreateSchema(t.db, server); err != nil {
			logrus.Warn(errors.Wrapf(err, "%s: couldn't create the schema", server.Key))
			continue
		}
		servers = append(servers, server)
		serverKeys = append(serverKeys, serverKey)
	}

	if len(servers) > 0 {
		if _, err := t.db.Model(&servers).
			OnConflict("(key) DO UPDATE").
			Set("status = ?", models.ServerStatusOpen).
			Set("version_code = EXCLUDED.version_code").
			Returning("*").
			Insert(); err != nil {
			err = errors.Wrap(err, "couldn't insert/update servers")
			logrus.Fatal(err)
			return err
		}
	}

	if _, err := t.db.Model(&models.Server{}).
		Set("status = ?", models.ServerStatusClosed).
		Where("key NOT IN (?) AND version_code =", pg.In(serverKeys), version.Code).
		Update(); err != nil {
		err = errors.Wrap(err, "couldn't update server statuses")
		logrus.Fatal(err)
		return err
	}

	entry.Infof("%s: Servers have been loaded", version.Host)
	return nil
}

func (t *taskLoadServers) validatePayload(version *models.Version) error {
	if version == nil {
		return errors.Errorf("taskLoadServers.validatePayload: Expected *models.Version, got nil")
	}
	return nil
}

func (t *taskLoadServers) getServers(version *models.Version) (map[string]string, error) {
	resp, err := http.Get(fmt.Sprintf("https://%s%s", version.Host, endpointGetServers))
	if err != nil {
		return nil, errors.Wrapf(err, "%s: taskLoadServers.loadServers couldn't load servers", version.Host)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: taskLoadServers.loadServers couldn't read the body", version.Host)
	}
	body, err := phpserialize.Decode(string(bodyBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "%s: taskLoadServers.loadServers couldn't decode the body into the go value", version.Host)
	}
	return body.(map[string]string), nil
}