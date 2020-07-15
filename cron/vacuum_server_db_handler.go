package cron

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
)

type vacuumServerDBHandler struct {
	db *pg.DB
}

func (h *vacuumServerDBHandler) vacuum() error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	withNotExistedPlayers := h.db.Model(&models.Player{}).Where("exists = false")
	withNotExistedTribes := h.db.Model(&models.Tribe{}).Where("exists = false")

	_, err = tx.Model(&models.PlayerHistory{}).
		With("players", withNotExistedPlayers).
		Where("player_id IN (Select id FROM players) OR player_history.create_date < ?", time.Now().Add(-1*24*time.Hour*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old player history")
	}

	_, err = tx.Model(&models.TribeHistory{}).
		With("tribes", withNotExistedTribes).
		Where("tribe_id IN (Select id FROM tribes) OR tribe_history.create_date < ?", time.Now().Add(-1*24*time.Hour*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old tribe history")
	}

	_, err = tx.Model(&models.DailyPlayerStats{}).
		With("players", withNotExistedPlayers).
		Where("player_id IN (Select id FROM players) OR daily_player_stats.create_date < ?", time.Now().Add(-1*24*time.Hour*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old player stats")
	}

	_, err = tx.Model(&models.DailyTribeStats{}).
		With("tribes", withNotExistedTribes).
		Where("tribe_id IN (Select id FROM tribes) OR daily_tribe_stats.create_date < ?", time.Now().Add(-1*24*time.Hour*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old tribe stats")
	}

	return tx.Commit()
}
