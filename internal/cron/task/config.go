package task

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"github.com/tribalwarshelp/cron/internal/cron/queue"
)

type Config struct {
	DB    *pg.DB
	Queue queue.Queue
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
