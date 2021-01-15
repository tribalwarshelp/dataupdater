package cron

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
)

type updateServerHistoryWorker struct {
	db       *pg.DB
	server   *models.Server
	location *time.Location
}

func (w *updateServerHistoryWorker) update() error {
	players := []*models.Player{}
	if err := w.db.Model(&players).Where("exists = true").Select(); err != nil {
		return errors.Wrap(err, "cannot load players")
	}

	now := time.Now().In(w.location)
	createDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	ph := []*models.PlayerHistory{}
	for _, player := range players {
		ph = append(ph, &models.PlayerHistory{
			OpponentsDefeated: player.OpponentsDefeated,
			PlayerID:          player.ID,
			TotalVillages:     player.TotalVillages,
			Points:            player.Points,
			Rank:              player.Rank,
			TribeID:           player.TribeID,
			CreateDate:        createDate,
		})
	}

	tribes := []*models.Tribe{}
	if err := w.db.Model(&tribes).Where("exists = true").Select(); err != nil {
		return errors.Wrap(err, "cannot load tribes")
	}
	th := []*models.TribeHistory{}
	for _, tribe := range tribes {
		th = append(th, &models.TribeHistory{
			OpponentsDefeated: tribe.OpponentsDefeated,
			TribeID:           tribe.ID,
			TotalMembers:      tribe.TotalMembers,
			TotalVillages:     tribe.TotalVillages,
			Points:            tribe.Points,
			AllPoints:         tribe.AllPoints,
			Rank:              tribe.Rank,
			Dominance:         tribe.Dominance,
			CreateDate:        createDate,
		})
	}

	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Close()

	if len(ph) > 0 {
		if _, err := w.db.Model(&ph).Insert(); err != nil {
			return errors.Wrap(err, "cannot insert players history")
		}
	}

	if len(th) > 0 {
		if _, err := w.db.Model(&th).Insert(); err != nil {
			return errors.Wrap(err, "cannot insert tribes history")
		}
	}

	if _, err := tx.Model(w.server).
		Set("history_updated_at = ?", time.Now()).
		WherePK().
		Returning("*").
		Update(); err != nil {
		return errors.Wrap(err, "cannot update server")

	}

	return tx.Commit()
}
