package cron

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
)

type updateServerStatsWorker struct {
	db       *pg.DB
	server   *models.Server
	location *time.Location
}

func (w *updateServerStatsWorker) prepare() (*models.ServerStats, error) {
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

func (w *updateServerStatsWorker) update() error {
	stats, err := w.prepare()
	if err != nil {
		return err
	}

	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

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
