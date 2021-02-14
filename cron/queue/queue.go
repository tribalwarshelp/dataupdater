package queue

import (
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/taskq/v3"
	"github.com/vmihailenco/taskq/v3/redisq"
)

type Config struct {
	DB          *pg.DB
	Redis       redis.UniversalClient
	WorkerLimit int
}

type queue struct {
	db                *pg.DB
	redis             redis.UniversalClient
	mainQueue         taskq.Queue
	ennoblementsQueue taskq.Queue
	factory           taskq.Factory
}

func New(cfg *Config) error {
	q := &queue{
		db:    cfg.DB,
		redis: cfg.Redis,
	}

	if err := q.init(cfg); err != nil {
		return err
	}

	return nil
}

func (q *queue) init(cfg *Config) error {
	q.factory = redisq.NewFactory()
	q.mainQueue = q.newQueue("main", cfg.WorkerLimit)
	q.ennoblementsQueue = q.newQueue("ennoblements", cfg.WorkerLimit*2)

	return nil
}

func (q *queue) newQueue(name string, limit int) taskq.Queue {
	return q.factory.RegisterQueue(&taskq.QueueOptions{
		Name:               name,
		ReservationTimeout: time.Minute * 2,
		Redis:              q.redis,
		MaxNumWorker:       int32(limit),
	})
}
