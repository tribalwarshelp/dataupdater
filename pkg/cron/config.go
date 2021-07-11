package cron

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"

	"github.com/tribalwarshelp/cron/pkg/queue"
)

type Config struct {
	DB        *pg.DB
	Queue     *queue.Queue
	RunOnInit bool
}

func validateConfig(cfg *Config) error {
	if cfg == nil || cfg.DB == nil {
		return errors.New("cfg.DB is required")
	}
	if cfg.Queue == nil {
		return errors.New("cfg.Queue is required")
	}
	return nil
}
