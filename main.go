package main

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/mode"

	twhelpcron "github.com/tribalwarshelp/cron/cron"
	"github.com/tribalwarshelp/cron/db"
	envutils "github.com/tribalwarshelp/cron/utils/env"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func init() {
	if err := setupENVs(); err != nil {
		logrus.Fatal(err)
	}
	setupLogger()
}

func main() {
	redisClient, err := initializeRedis()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't connect to Redis"))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logrus.Warn(errors.Wrap(err, "Couldn't close the Redis connection"))
		}
	}()

	dbConn, err := db.New(&db.Config{LogQueries: envutils.GetenvBool("LOG_DB_QUERIES")})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't connect to the db"))
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			logrus.Warn(errors.Wrap(err, "Couldn't close the db connection"))
		}
	}()

	c, err := twhelpcron.New(&twhelpcron.Config{
		DB:          dbConn,
		RunOnInit:   envutils.GetenvBool("RUN_ON_INIT"),
		Redis:       redisClient,
		WorkerLimit: envutils.GetenvInt("WORKER_LIMIT"),
		Opts: []cron.Option{
			cron.WithChain(
				cron.SkipIfStillRunning(
					cron.PrintfLogger(logrus.WithField("package", "cron")),
				),
			),
		},
	})
	if err != nil {
		logrus.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Cron is running!")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	logrus.Info("shutting down")
	if err := c.Stop(); err != nil {
		logrus.Fatal(err)
	}
}

func setupLogger() {
	if mode.Get() == mode.DevelopmentMode {
		logrus.SetLevel(logrus.DebugLevel)
	}

	timestampFormat := "2006-01-02 15:04:05"
	if mode.Get() == mode.ProductionMode {
		customFormatter := new(logrus.JSONFormatter)
		customFormatter.TimestampFormat = timestampFormat
		logrus.SetFormatter(customFormatter)
	} else {
		customFormatter := new(logrus.TextFormatter)
		customFormatter.TimestampFormat = timestampFormat
		customFormatter.FullTimestamp = true
		logrus.SetFormatter(customFormatter)
	}
}

func setupENVs() error {
	err := os.Setenv("TZ", "UTC")
	if err != nil {
		return errors.Wrap(err, "setupENVs")
	}

	if mode.Get() == mode.DevelopmentMode {
		err := godotenv.Load(".env.local")
		if err != nil {
			return errors.Wrap(err, "setupENVs")
		}
	}

	return nil
}

func initializeRedis() (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     envutils.GetenvString("REDIS_ADDR"),
		Username: envutils.GetenvString("REDIS_USERNAME"),
		Password: envutils.GetenvString("REDIS_PASSWORD"),
		DB:       envutils.GetenvInt("REDIS_DB"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "initializeRedis")
	}
	return client, nil
}
