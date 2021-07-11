package queue

import (
	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

type Config struct {
	Redis       redis.UniversalClient
	WorkerLimit int
	DB          *pg.DB
}

func validateConfig(cfg *Config) error {
	if cfg == nil || cfg.Redis == nil {
		return errors.New("cfg.Redis is required")
	}
	return nil
}

type registerTasksConfig struct {
	DB    *pg.DB
	Queue *Queue
}

func validateRegisterTasksConfig(cfg *registerTasksConfig) error {
	if cfg == nil || cfg.DB == nil {
		return errors.New("cfg.DB is required")
	}
	if cfg.Queue == nil {
		return errors.New("cfg.Queue is required")
	}
	return nil
}
