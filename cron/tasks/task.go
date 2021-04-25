package tasks

import (
	"github.com/go-pg/pg/v10"

	"github.com/tribalwarshelp/cron/cron/queue"
)

type task struct {
	db    *pg.DB
	queue queue.Queue
}
