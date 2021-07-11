package internal

import (
	"context"
	"github.com/Kichiyaki/goutil/envutil"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"time"
)

func NewRedisClient() (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     envutil.GetenvString("REDIS_ADDR"),
		Username: envutil.GetenvString("REDIS_USERNAME"),
		Password: envutil.GetenvString("REDIS_PASSWORD"),
		DB:       envutil.GetenvInt("REDIS_DB"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "NewRedisClient")
	}
	return client, nil
}
