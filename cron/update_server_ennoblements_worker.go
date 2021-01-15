package cron

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"
	"github.com/tribalwarshelp/shared/tw/dataloader"
)

type updateServerEnnoblementsWorker struct {
	db         *pg.DB
	dataloader dataloader.DataLoader
	server     *models.Server
}

func (w *updateServerEnnoblementsWorker) loadEnnoblements() ([]*models.Ennoblement, error) {
	lastEnnoblement := &models.Ennoblement{}
	if err := w.db.
		Model(lastEnnoblement).
		Limit(1).
		Order("ennobled_at DESC").
		Select(); err != nil && err != pg.ErrNoRows {
		return nil, errors.Wrapf(err, "cannot load last ennoblement")
	}

	return w.dataloader.LoadEnnoblements(&dataloader.LoadEnnoblementsConfig{
		EnnobledAtGT: lastEnnoblement.EnnobledAt,
	})
}

func (w *updateServerEnnoblementsWorker) update() error {
	ennoblements, err := w.loadEnnoblements()
	if err != nil {
		return err
	}

	if len(ennoblements) > 0 {
		if _, err := w.db.Model(&ennoblements).Insert(); err != nil {
			return errors.Wrap(err, "cannot insert ennoblements")
		}
	}

	return nil
}
