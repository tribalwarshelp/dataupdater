package queue

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/taskq/v3"
	"github.com/vmihailenco/taskq/v3/redisq"
)

const (
	Main         = "main"
	Ennoblements = "ennoblements"
)

type Queue interface {
	Start(ctx context.Context) error
	Close() error
	Add(name string, msg *taskq.Message) error
}

type queue struct {
	redis             redis.UniversalClient
	mainQueue         taskq.Queue
	ennoblementsQueue taskq.Queue
	factory           taskq.Factory
}

func New(cfg *Config) (Queue, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	q := &queue{
		redis: cfg.Redis,
	}

	if err := q.init(cfg); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *queue) init(cfg *Config) error {
	q.factory = redisq.NewFactory()
	q.mainQueue = q.registerQueue(Main, cfg.WorkerLimit)
	q.ennoblementsQueue = q.registerQueue(Ennoblements, cfg.WorkerLimit)

	return nil
}

func (q *queue) registerQueue(name string, limit int) taskq.Queue {
	return q.factory.RegisterQueue(&taskq.QueueOptions{
		Name:               name,
		ReservationTimeout: time.Minute * 2,
		Redis:              q.redis,
		MinNumWorker:       int32(limit),
		MaxNumWorker:       int32(limit),
	})
}

func (q *queue) getQueueByName(name string) taskq.Queue {
	switch name {
	case Main:
		return q.mainQueue
	case Ennoblements:
		return q.ennoblementsQueue
	}
	return nil
}

func (q *queue) Start(ctx context.Context) error {
	if err := q.factory.StartConsumers(ctx); err != nil {
		return errors.Wrap(err, "couldn't start the queue")
	}
	return nil
}

func (q *queue) Close() error {
	if err := q.factory.Close(); err != nil {
		return errors.Wrap(err, "couldn't close the queue")
	}
	return nil
}

func (q *queue) Add(name string, msg *taskq.Message) error {
	queue := q.getQueueByName(name)
	if queue == nil {
		return errors.Errorf("couldn't add the message to the queue: unknown queue name '%s'", name)
	}
	if err := queue.Add(msg); err != nil {
		return errors.Wrap(err, "couldn't add the message to the queue")
	}
	return nil
}
