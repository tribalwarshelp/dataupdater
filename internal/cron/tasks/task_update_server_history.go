package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"time"
)

type taskUpdateServerHistory struct {
	*task
}

func (t *taskUpdateServerHistory) execute(timezone string, server *twmodel.Server) error {
	if err := t.validatePayload(server); err != nil {
		log.Debug(err)
		return nil
	}
	location, err := t.loadLocation(timezone)
	if err != nil {
		err = errors.Wrap(err, "taskUpdateServerHistory.execute")
		log.Error(err)
		return err
	}
	entry := log.WithField("key", server.Key)
	entry.Infof("taskUpdateServerHistory.execute: %s: update of the history has started...", server.Key)
	err = (&workerUpdateServerHistory{
		db:       t.db.WithParam("SERVER", pg.Safe(server.Key)),
		server:   server,
		location: location,
	}).update()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateServerHistory.execute")
		entry.Error(err)
		return err
	}
	entry.Infof("taskUpdateServerHistory.execute: %s: history has been updated", server.Key)

	return nil
}

func (t *taskUpdateServerHistory) validatePayload(server *twmodel.Server) error {
	if server == nil {
		return errors.New("taskUpdateServerHistory.validatePayload: Expected *twmodel.Server, got nil")
	}

	return nil
}

type workerUpdateServerHistory struct {
	db       *pg.DB
	server   *twmodel.Server
	location *time.Location
}

func (w *workerUpdateServerHistory) update() error {
	var players []*twmodel.Player
	if err := w.db.Model(&players).Where("exists = true").Select(); err != nil {
		return errors.Wrap(err, "couldn't load players")
	}

	now := time.Now().In(w.location)
	createDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var ph []*twmodel.PlayerHistory
	for _, player := range players {
		ph = append(ph, &twmodel.PlayerHistory{
			OpponentsDefeated: player.OpponentsDefeated,
			PlayerID:          player.ID,
			TotalVillages:     player.TotalVillages,
			Points:            player.Points,
			Rank:              player.Rank,
			TribeID:           player.TribeID,
			CreateDate:        createDate,
		})
	}

	var tribes []*twmodel.Tribe
	if err := w.db.Model(&tribes).Where("exists = true").Select(); err != nil {
		return errors.Wrap(err, "couldn't load tribes")
	}
	var th []*twmodel.TribeHistory
	for _, tribe := range tribes {
		th = append(th, &twmodel.TribeHistory{
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
	defer func(s *twmodel.Server) {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrapf(err, "%s: Couldn't rollback the transaction", s.Key))
		}
	}(w.server)

	if len(ph) > 0 {
		if _, err := w.db.Model(&ph).Returning("NULL").Insert(); err != nil {
			return errors.Wrap(err, "couldn't insert players history")
		}
	}

	if len(th) > 0 {
		if _, err := w.db.Model(&th).Returning("NULL").Insert(); err != nil {
			return errors.Wrap(err, "couldn't insert tribes history")
		}
	}

	if _, err := tx.Model(w.server).
		Set("history_updated_at = ?", time.Now()).
		WherePK().
		Returning("*").
		Update(); err != nil {
		return errors.Wrap(err, "couldn't update server")

	}

	return tx.Commit()
}
