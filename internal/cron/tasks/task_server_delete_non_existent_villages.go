package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twdataloader"
	"github.com/tribalwarshelp/shared/tw/twmodel"
)

type taskServerDeleteNonExistentVillages struct {
	*task
}

func (t *taskServerDeleteNonExistentVillages) execute(url string, server *twmodel.Server) error {
	if err := t.validatePayload(server); err != nil {
		log.Debug(err)
		return nil
	}
	entry := log.WithField("key", server.Key)
	entry.Infof("taskServerDeleteNonExistentVillages.execute: %s: Deleting non-existent villages...", server.Key)
	err := (&workerDeleteNonExistentVillages{
		db:         t.db.WithParam("SERVER", pg.Safe(server.Key)),
		dataloader: newServerDataLoader(url),
		server:     server,
	}).delete()
	if err != nil {
		err = errors.Wrap(err, "taskServerDeleteNonExistentVillages.execute")
		entry.Error(err)
		return err
	}
	entry.Infof("taskServerDeleteNonExistentVillages.execute: %s: Non-existent villages have been deleted", server.Key)
	return nil
}

func (t *taskServerDeleteNonExistentVillages) validatePayload(server *twmodel.Server) error {
	if server == nil {
		return errors.New("taskUpdateServerData.validatePayload: Expected *twmodel.Server, got nil")
	}

	return nil
}

type workerDeleteNonExistentVillages struct {
	db         *pg.DB
	dataloader twdataloader.ServerDataLoader
	server     *twmodel.Server
}

func (w *workerDeleteNonExistentVillages) delete() error {
	villages, err := w.dataloader.LoadVillages()
	if err != nil {
		return errors.Wrap(err, "workerDeleteNonExistentVillages.delete")
	}
	var idsToDelete []int
	searchableByVillageID := &villagesSearchableByID{villages}
	if err := w.db.
		Model(&twmodel.Village{}).
		Column("id").
		ForEach(func(village *twmodel.Village) error {
			index := searchByID(searchableByVillageID, village.ID)
			if index < 0 {
				idsToDelete = append(idsToDelete, village.ID)
			}
			return nil
		}); err != nil {
		return errors.Wrap(err, "workerDeleteNonExistentVillages.delete")
	}

	totalDeleted := 0
	if len(idsToDelete) > 0 {
		result, err := w.db.Model(&twmodel.Village{}).Where("id = ANY(?)", pg.Array(idsToDelete)).Delete()
		if err != nil {
			return errors.Wrap(err, "workerDeleteNonExistentVillages.delete")
		}
		totalDeleted = result.RowsAffected()
	}
	log.Debugf("%s: deleted %d villages", w.server.Key, totalDeleted)
	return nil
}
