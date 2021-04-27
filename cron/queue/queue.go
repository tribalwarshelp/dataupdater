package queue

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/taskq/v3"
	"github.com/vmihailenco/taskq/v3/redisq"
)

type Name string

const (
	MainQueue         Name = "main"
	EnnoblementsQueue Name = "ennoblements"
)

type Queue interface {
	Start(ctx context.Context) error
	Close() error
	Add(name Name, msg *taskq.Message) error
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
	q.mainQueue = q.registerQueue(MainQueue, cfg.WorkerLimit)
	q.ennoblementsQueue = q.registerQueue(EnnoblementsQueue, cfg.WorkerLimit*2)

	return nil
}

func (q *queue) registerQueue(name Name, limit int) taskq.Queue {
	return q.factory.RegisterQueue(&taskq.QueueOptions{
		Name:               string(name),
		ReservationTimeout: time.Minute * 2,
		Redis:              q.redis,
		MinNumWorker:       int32(limit),
		MaxNumWorker:       int32(limit),
	})
}

func (q *queue) getQueueByName(name Name) taskq.Queue {
	switch name {
	case MainQueue:
		return q.mainQueue
	case EnnoblementsQueue:
		return q.ennoblementsQueue
	}
	return nil
}

func (q *queue) Start(ctx context.Context) error {
	if err := q.factory.StartConsumers(ctx); err != nil {
		return errors.Wrap(err, "Couldn't start the queue")
	}
	return nil
}

func (q *queue) Close() error {
	if err := q.factory.Close(); err != nil {
		return errors.Wrap(err, "Couldn't close the queue")
	}
	return nil
}

func (q *queue) Add(name Name, msg *taskq.Message) error {
	queue := q.getQueueByName(name)
	if queue == nil {
		return errors.Errorf("Couldn't add the message to the queue: unknown queue name '%s'", name)
	}
	if err := queue.Add(msg); err != nil {
		return errors.Wrap(err, "Couldn't add the message to the queue")
	}
	return nil
}
