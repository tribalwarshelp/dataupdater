package queue

import (
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

type Config struct {
	Redis       redis.UniversalClient
	WorkerLimit int
}

func validateConfig(cfg *Config) error {
	if cfg == nil || cfg.Redis == nil {
		return errors.New("validateConfig: cfg.Redis is required")
	}
	return nil
}
