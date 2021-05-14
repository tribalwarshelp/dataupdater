package task

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/tw/twdataloader"
	"github.com/tribalwarshelp/shared/tw/twmodel"
)

type taskUpdateServerEnnoblements struct {
	*task
}

func (t *taskUpdateServerEnnoblements) execute(url string, server *twmodel.Server) error {
	if err := t.validatePayload(server); err != nil {
		log.Debug(errors.Wrap(err, "taskUpdateServerEnnoblements.execute"))
		return nil
	}
	entry := log.WithField("key", server.Key)
	entry.Debugf("%s: update of the ennoblements has started...", server.Key)
	err := (&workerUpdateServerEnnoblements{
		db:         t.db.WithParam("SERVER", pg.Safe(server.Key)),
		dataloader: newServerDataLoader(url),
	}).update()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateServerEnnoblements.execute")
		entry.Error(err)
		return err
	}
	entry.Debugf("%s: the ennoblements have been updated", server.Key)

	return nil
}

func (t *taskUpdateServerEnnoblements) validatePayload(server *twmodel.Server) error {
	if server == nil {
		return errors.Errorf("expected *twmodel.Server, got nil")
	}

	return nil
}

type workerUpdateServerEnnoblements struct {
	db         *pg.DB
	dataloader twdataloader.ServerDataLoader
}

func (w *workerUpdateServerEnnoblements) loadEnnoblements() ([]*twmodel.Ennoblement, error) {
	lastEnnoblement := &twmodel.Ennoblement{}
	if err := w.db.
		Model(lastEnnoblement).
		Limit(1).
		Order("ennobled_at DESC").
		Select(); err != nil && err != pg.ErrNoRows {
		return nil, errors.Wrapf(err, "couldn't load last ennoblement")
	}

	return w.dataloader.LoadEnnoblements(&twdataloader.LoadEnnoblementsConfig{
		EnnobledAtGT: lastEnnoblement.EnnobledAt,
	})
}

func (w *workerUpdateServerEnnoblements) update() error {
	ennoblements, err := w.loadEnnoblements()
	if err != nil {
		return err
	}

	if len(ennoblements) > 0 {
		if _, err := w.db.Model(&ennoblements).Returning("NULL").Insert(); err != nil {
			return errors.Wrap(err, "couldn't insert ennoblements")
		}
	}

	return nil
}
