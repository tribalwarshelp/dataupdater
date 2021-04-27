package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
	"time"
)

type taskUpdateServerStats struct {
	*task
}

func (t *taskUpdateServerStats) execute(timezone string, server *models.Server) error {
	if err := t.validatePayload(server); err != nil {
		log.Debug(err)
		return nil
	}
	location, err := t.loadLocation(timezone)
	if err != nil {
		err = errors.Wrap(err, "taskUpdateServerStats.execute")
		log.Error(err)
		return err
	}
	entry := log.WithField("key", server.Key)
	entry.Infof("taskUpdateServerStats.execute: %s: update of the stats has started...", server.Key)
	err = (&workerUpdateServerStats{
		db:       t.db.WithParam("SERVER", pg.Safe(server.Key)),
		server:   server,
		location: location,
	}).update()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateServerStats.execute")
		entry.Error(err)
		return err
	}
	entry.Infof("taskUpdateServerStats.execute: %s: stats have been updated", server.Key)

	return nil
}

func (t *taskUpdateServerStats) validatePayload(server *models.Server) error {
	if server == nil {
		return errors.Errorf("taskUpdateServerStats.validatePayload: Expected *models.Server, got nil")
	}

	return nil
}

type workerUpdateServerStats struct {
	db       *pg.DB
	server   *models.Server
	location *time.Location
}

func (w *workerUpdateServerStats) prepare() (*models.ServerStats, error) {
	activePlayers, err := w.db.Model(&models.Player{}).Where("exists = true").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count active players")
	}
	inactivePlayers, err := w.db.Model(&models.Player{}).Where("exists = false").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count inactive players")
	}
	players := activePlayers + inactivePlayers

	activeTribes, err := w.db.Model(&models.Tribe{}).Where("exists = true").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count active tribes")
	}
	inactiveTribes, err := w.db.Model(&models.Tribe{}).Where("exists = false").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count inactive tribes")
	}
	tribes := activeTribes + inactiveTribes

	barbarianVillages, err := w.db.Model(&models.Village{}).Where("player_id = 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count barbarian villages")
	}
	bonusVillages, err := w.db.Model(&models.Village{}).Where("bonus <> 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count bonus villages")
	}
	playerVillages, err := w.db.Model(&models.Village{}).Where("player_id <> 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count player villages")
	}
	villages, err := w.db.Model(&models.Village{}).Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count villages")
	}

	now := time.Now().In(w.location)
	createDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return &models.ServerStats{
		ActivePlayers:   activePlayers,
		InactivePlayers: inactivePlayers,
		Players:         players,

		ActiveTribes:   activeTribes,
		InactiveTribes: inactiveTribes,
		Tribes:         tribes,

		BarbarianVillages: barbarianVillages,
		BonusVillages:     bonusVillages,
		PlayerVillages:    playerVillages,
		Villages:          villages,
		CreateDate:        createDate,
	}, nil
}

func (w *workerUpdateServerStats) update() error {
	stats, err := w.prepare()
	if err != nil {
		return err
	}

	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer func(s *models.Server) {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrapf(err, "%s: Couldn't rollback the transaction", s.Key))
		}
	}(w.server)

	if _, err := tx.Model(stats).Insert(); err != nil {
		return errors.Wrap(err, "cannot insert server stats")
	}

	_, err = tx.Model(w.server).
		Set("stats_updated_at = ?", time.Now()).
		WherePK().
		Returning("*").
		Update()
	if err != nil {
		return errors.Wrap(err, "cannot update server")
	}

	return tx.Commit()
}
