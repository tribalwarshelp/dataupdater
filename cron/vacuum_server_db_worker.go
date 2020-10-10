package cron

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
)

const (
	day = 24 * time.Hour
)

type vacuumServerDBWorker struct {
	db *pg.DB
}

func (h *vacuumServerDBWorker) vacuum() error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	withNonExistentPlayers := h.db.Model(&models.Player{}).Where("exists = false")
	withNonExistentTribes := h.db.Model(&models.Tribe{}).Where("exists = false")

	_, err = tx.Model(&models.PlayerHistory{}).
		With("players", withNonExistentPlayers).
		Where("player_id IN (Select id FROM players) OR player_history.create_date < ?", time.Now().Add(-1*day*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old player history records")
	}

	_, err = tx.Model(&models.TribeHistory{}).
		With("tribes", withNonExistentTribes).
		Where("tribe_id IN (Select id FROM tribes) OR tribe_history.create_date < ?", time.Now().Add(-1*day*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old tribe history records")
	}

	_, err = tx.Model(&models.DailyPlayerStats{}).
		With("players", withNonExistentPlayers).
		Where("player_id IN (Select id FROM players) OR daily_player_stats.create_date < ?", time.Now().Add(-1*day*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old player stats records")
	}

	_, err = tx.Model(&models.DailyTribeStats{}).
		With("tribes", withNonExistentTribes).
		Where("tribe_id IN (Select id FROM tribes) OR daily_tribe_stats.create_date < ?", time.Now().Add(-1*day*90)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "cannot delete old tribe stats records")
	}

	return tx.Commit()
}
