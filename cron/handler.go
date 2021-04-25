package cron

import (
	"context"

	"github.com/tribalwarshelp/cron/cron/queue"
	"github.com/tribalwarshelp/cron/cron/tasks"
)

type handler struct {
	queue queue.Queue
}

func (h *handler) updateServerData() {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameLoadVersionsAndUpdateServerData).WithArgs(context.Background()))
}

func (h *handler) updateEnnoblements() {
	h.queue.Add(queue.EnnoblementsQueue, tasks.Get(tasks.TaskUpdateEnnoblements).WithArgs(context.Background()))
}

func (h *handler) updateHistory(timezone string) {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskUpdateHistory).WithArgs(context.Background(), timezone))
}

func (h *handler) updateStats(timezone string) {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskUpdateStats).WithArgs(context.Background(), timezone))
}

func (h *handler) vacuumDatabase() {
	h.queue.Add(queue.MainQueue, tasks.Get(tasks.TaskNameVacuum).WithArgs(context.Background()))
}
