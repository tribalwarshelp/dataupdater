package cron

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/shared/models"

	"github.com/tribalwarshelp/cron/cron/queue"
	"github.com/tribalwarshelp/cron/cron/tasks"
)

type handler struct {
	db                   *pg.DB
	maxConcurrentWorkers int
	pool                 *pool
	queue                queue.Queue
}

func (h *handler) init() error {
	if h.maxConcurrentWorkers <= 0 {
		h.maxConcurrentWorkers = runtime.NumCPU()
	}

	if h.pool == nil {
		h.pool = newPool(h.maxConcurrentWorkers)
	}

	return nil
}

func (h *handler) updateServerData() {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameLoadVersionsAndUpdateServerData).WithArgs(context.Background()))
}

func (h *handler) updateEnnoblements() {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskUpdateEnnoblements).WithArgs(context.Background()))
}

func (h *handler) updateHistory(location *time.Location) {
	servers := []*models.Server{}
	log := log.WithField("timezone", location.String())
	year, month, day := time.Now().In(location).Date()
	t := time.Date(year, month, day, 1, 30, 0, 0, location)
	err := h.db.
		Model(&servers).
		Where(
			"status = ? AND (history_updated_at IS NULL OR history_updated_at < ?) AND timezone = ?",
			models.ServerStatusOpen,
			t,
			location.String(),
		).
		Relation("Version").
		Select()
	if err != nil {
		log.Errorln(errors.Wrap(err, "updateHistory"))
		return
	}
	log.
		WithField("numberOfServers", len(servers)).
		Info("updateHistory: servers loaded")

	var wg sync.WaitGroup

	for _, server := range servers {
		h.pool.waitForWorker()
		wg.Add(1)
		worker := &updateServerHistoryWorker{
			db:       h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server:   server,
			location: location,
		}
		go func(server *models.Server, worker *updateServerHistoryWorker) {
			defer func() {
				h.pool.releaseWorker()
				wg.Done()
			}()
			log := log.WithField("serverKey", server.Key)
			log.Infof("updateHistory: %s: updating history", server.Key)
			if err := worker.update(); err != nil {
				log.Errorln("updateHistory:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("updateHistory: %s: history updated", server.Key)
		}(server, worker)
	}

	wg.Wait()
}

func (h *handler) updateStats(location *time.Location) {
	servers := []*models.Server{}
	log := log.WithField("timezone", location.String())
	year, month, day := time.Now().In(location).Date()
	t := time.Date(year, month, day, 1, 45, 0, 0, location)
	err := h.db.
		Model(&servers).
		Where(
			"status = ? AND (stats_updated_at IS NULL OR stats_updated_at < ?) AND timezone = ?",
			models.ServerStatusOpen,
			t,
			location.String(),
		).
		Relation("Version").
		Select()
	if err != nil {
		log.Errorf(errors.Wrap(err, "updateServerStats").Error())
		return
	}
	log.WithField("numberOfServers", len(servers)).Info("updateServerStats: servers loaded")

	var wg sync.WaitGroup

	for _, server := range servers {
		h.pool.waitForWorker()
		wg.Add(1)
		worker := &updateServerStatsWorker{
			db:       h.db.WithParam("SERVER", pg.Safe(server.Key)),
			server:   server,
			location: location,
		}
		go func(server *models.Server, worker *updateServerStatsWorker) {
			defer func() {
				h.pool.releaseWorker()
				wg.Done()
			}()
			log := log.WithField("serverKey", server.Key)
			log.Infof("updateServerStats: %s: updating stats", server.Key)
			if err := worker.update(); err != nil {
				log.Errorln("updateServerStats:", errors.Wrap(err, server.Key))
				return
			}
			log.Infof("updateServerStats: %s: stats updated", server.Key)
		}(server, worker)
	}

	wg.Wait()
}

func (h *handler) vacuumDatabase() {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameVacuum).WithArgs(context.Background()))
}
