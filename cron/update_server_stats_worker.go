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

func (h *updateServerStatsWorker) prepare() (*models.ServerStats, error) {
	activePlayers, err := h.db.Model(&models.Player{}).Where("exists = true").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count active players")
	}
	inactivePlayers, err := h.db.Model(&models.Player{}).Where("exists = false").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count inactive players")
	}
	players := activePlayers + inactivePlayers

	activeTribes, err := h.db.Model(&models.Tribe{}).Where("exists = true").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count active tribes")
	}
	inactiveTribes, err := h.db.Model(&models.Tribe{}).Where("exists = false").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count inactive tribes")
	}
	tribes := activeTribes + inactiveTribes

	barbarianVillages, err := h.db.Model(&models.Village{}).Where("player_id = 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count barbarian villages")
	}
	bonusVillages, err := h.db.Model(&models.Village{}).Where("bonus <> 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count bonus villages")
	}
	playerVillages, err := h.db.Model(&models.Village{}).Where("player_id <> 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count player villages")
	}
	villages, err := h.db.Model(&models.Village{}).Count()
	if err != nil {
		return nil, errors.Wrap(err, "couldnt count villages")
	}

	now := time.Now().In(h.location)
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

func (h *updateServerStatsWorker) update() error {
	stats, err := h.prepare()
	if err != nil {
		return err
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if _, err := tx.Model(stats).Insert(); err != nil {
		return errors.Wrap(err, "couldnt insert server stats")
	}

	_, err = tx.Model(h.server).
		Set("stats_updated_at = ?", time.Now()).
		WherePK().
		Returning("*").
		Update()
	if err != nil {
		return errors.Wrap(err, "couldnt update server")
	}

	return tx.Commit()
}
