package task

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twmodel"
	"time"
)

const (
	day = 24 * time.Hour
)

type taskVacuumServerDB struct {
	*task
}

func (t *taskVacuumServerDB) execute(server *twmodel.Server) error {
	if err := t.validatePayload(server); err != nil {
		log.Debug(err)
		return nil
	}
	entry := log.WithField("key", server.Key)
	entry.Infof("taskVacuumServerDB.execute: %s: vacumming the database...", server.Key)
	err := (&workerVacuumServerDB{
		db:     t.db.WithParam("SERVER", pg.Safe(server.Key)),
		server: server,
	}).vacuum()
	if err != nil {
		err = errors.Wrap(err, "taskVacuumServerDB.execute")
		entry.Error(err)
		return err
	}
	entry.Infof("taskVacuumServerDB.execute: %s: the database has been vacummed", server.Key)

	return nil
}

func (t *taskVacuumServerDB) validatePayload(server *twmodel.Server) error {
	if server == nil {
		return errors.New("taskVacuumServerDB.validatePayload: Expected *twmodel.Server, got nil")
	}

	return nil
}

type workerVacuumServerDB struct {
	db     *pg.DB
	server *twmodel.Server
}

func (w *workerVacuumServerDB) vacuum() error {
	tx, err := w.db.Begin()
	if err != nil {
		return err
	}
	defer func(s *twmodel.Server) {
		if err := tx.Close(); err != nil {
			log.Warn(errors.Wrapf(err, "%s: Couldn't rollback the transaction", s.Key))
		}
	}(w.server)

	withNonExistentPlayers := w.db.Model(&twmodel.Player{}).Column("id").Where("exists = false and NOW() - deleted_at > '14 days'")
	withNonExistentTribes := w.db.Model(&twmodel.Tribe{}).Column("id").Where("exists = false and NOW() - deleted_at > '1 days'")

	_, err = tx.Model(&twmodel.PlayerHistory{}).
		With("players", withNonExistentPlayers).
		Where("player_id IN (Select id FROM players) OR player_history.create_date < ?", time.Now().Add(-1*day*180)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "couldn't delete the old player history records")
	}

	_, err = tx.Model(&twmodel.TribeHistory{}).
		With("tribes", withNonExistentTribes).
		Where("tribe_id IN (Select id FROM tribes) OR tribe_history.create_date < ?", time.Now().Add(-1*day*180)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "couldn't delete the old tribe history records")
	}

	_, err = tx.Model(&twmodel.DailyPlayerStats{}).
		With("players", withNonExistentPlayers).
		Where("player_id IN (Select id FROM players) OR daily_player_stats.create_date < ?", time.Now().Add(-1*day*180)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "couldn't delete the old player stats records")
	}

	_, err = tx.Model(&twmodel.DailyTribeStats{}).
		With("tribes", withNonExistentTribes).
		Where("tribe_id IN (Select id FROM tribes) OR daily_tribe_stats.create_date < ?", time.Now().Add(-1*day*180)).
		Delete()
	if err != nil {
		return errors.Wrap(err, "couldn't delete the old tribe stats records")
	}

	return tx.Commit()
}
