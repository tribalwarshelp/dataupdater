package cron

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
)

type updateServerStatsHandler struct {
	db     *pg.DB
	server *models.Server
}

func (h *updateServerStatsHandler) prepare() (*models.ServerStats, error) {
	activePlayers, err := h.db.Model(&models.Player{}).Where("exists = true").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count active players")
	}
	inactivePlayers, err := h.db.Model(&models.Player{}).Where("exists = false").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count inactive players")
	}
	players := activePlayers + inactivePlayers

	activeTribes, err := h.db.Model(&models.Tribe{}).Where("exists = true").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count active tribes")
	}
	inactiveTribes, err := h.db.Model(&models.Tribe{}).Where("exists = false").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count inactive tribes")
	}
	tribes := activeTribes + inactiveTribes

	barbarianVillages, err := h.db.Model(&models.Village{}).Where("player_id = 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count barbarian villages")
	}
	bonusVillages, err := h.db.Model(&models.Village{}).Where("bonus <> 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count bonus villages")
	}
	playerVillages, err := h.db.Model(&models.Village{}).Where("player_id <> 0").Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count player villages")
	}
	villages, err := h.db.Model(&models.Village{}).Count()
	if err != nil {
		return nil, errors.Wrap(err, "cannot count villages")
	}

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
	}, nil
}

func (h *updateServerStatsHandler) update() error {
	stats, err := h.prepare()
	if err != nil {
		return err
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if err := tx.Insert(stats); err != nil {
		return errors.Wrap(err, "cannot insert server stats")
	}

	_, err = tx.Model(h.server).
		Set("stats_updated_at = ?", time.Now()).
		WherePK().
		Returning("*").
		Update()
	if err != nil {
		return errors.Wrap(err, "cannot update server")
	}

	return tx.Commit()
}
