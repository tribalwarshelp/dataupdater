package tasks

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
	"github.com/tribalwarshelp/shared/tw/dataloader"
)

type taskUpdateServerEnnoblements struct {
	*task
}

func (t *taskUpdateServerEnnoblements) execute(url string, server *models.Server) error {
	if err := t.validatePayload(server); err != nil {
		log.Debug(err)
		return nil
	}
	entry := log.WithField("key", server.Key)
	entry.Debugf("%s: update of the ennoblements has started...", server.Key)
	err := (&workerUpdateServerEnnoblements{
		db:         t.db.WithParam("SERVER", pg.Safe(server.Key)),
		dataloader: newDataloader(url),
	}).update()
	if err != nil {
		err = errors.Wrap(err, "taskUpdateServerEnnoblements.execute")
		entry.Error(err)
		return err
	}
	entry.Debugf("%s: ennoblements has been updated", server.Key)

	return nil
}

func (t *taskUpdateServerEnnoblements) validatePayload(server *models.Server) error {
	if server == nil {
		return errors.Errorf("taskUpdateServerEnnoblements.validatePayload: Expected *models.Server, got nil")
	}

	return nil
}

type workerUpdateServerEnnoblements struct {
	db         *pg.DB
	dataloader dataloader.DataLoader
}

func (w *workerUpdateServerEnnoblements) loadEnnoblements() ([]*models.Ennoblement, error) {
	lastEnnoblement := &models.Ennoblement{}
	if err := w.db.
		Model(lastEnnoblement).
		Limit(1).
		Order("ennobled_at DESC").
		Select(); err != nil && err != pg.ErrNoRows {
		return nil, errors.Wrapf(err, "couldn't load last ennoblement")
	}

	return w.dataloader.LoadEnnoblements(&dataloader.LoadEnnoblementsConfig{
		EnnobledAtGT: lastEnnoblement.EnnobledAt,
	})
}

func (w *workerUpdateServerEnnoblements) update() error {
	ennoblements, err := w.loadEnnoblements()
	if err != nil {
		return err
	}

	if len(ennoblements) > 0 {
		if _, err := w.db.Model(&ennoblements).Insert(); err != nil {
			return errors.Wrap(err, "couldn't insert ennoblements")
		}
	}

	return nil
}
