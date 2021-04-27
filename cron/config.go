package cron

import (
	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

type Config struct {
	DB          *pg.DB
	Redis       redis.UniversalClient
	RunOnInit   bool
	Opts        []cron.Option
	WorkerLimit int
}

func validateConfig(cfg *Config) error {
	if cfg == nil || cfg.DB == nil {
		return errors.New("validateConfig: cfg.DB is required")
	}
	if cfg.Redis == nil {
		return errors.New("validateConfig: cfg.Redis is required")
	}
	return nil
}