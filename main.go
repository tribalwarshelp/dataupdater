package main

import (
	"context"
	"github.com/Kichiyaki/appmode"
	"github.com/Kichiyaki/goutil/envutil"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	twhelpcron "github.com/tribalwarshelp/cron/pkg/cron"
	"github.com/tribalwarshelp/cron/pkg/postgres"

	"github.com/joho/godotenv"
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

	dbConn, err := postgres.Connect(&postgres.Config{LogQueries: envutil.GetenvBool("LOG_DB_QUERIES")})
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
		RunOnInit:   envutil.GetenvBool("RUN_ON_INIT"),
		Redis:       redisClient,
		WorkerLimit: envutil.GetenvInt("WORKER_LIMIT"),
	})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't initialize a cron instance"))
	}
	if err := c.Start(context.Background()); err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't start the cron"))
	}

	logrus.Info("Cron is up and running!")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	logrus.Info("shutting down")
	if err := c.Stop(); err != nil {
		logrus.Fatal(err)
	}
}

func setupENVs() error {
	err := os.Setenv("TZ", "UTC")
	if err != nil {
		return errors.Wrap(err, "setupENVs")
	}

	if appmode.Equals(appmode.DevelopmentMode) {
		err := godotenv.Load(".env.local")
		if err != nil {
			return errors.Wrap(err, "setupENVs")
		}
	}

	return nil
}

func setupLogger() {
	if appmode.Equals(appmode.DevelopmentMode) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	timestampFormat := "2006-01-02 15:04:05"
	if appmode.Equals(appmode.ProductionMode) {
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

func initializeRedis() (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     envutil.GetenvString("REDIS_ADDR"),
		Username: envutil.GetenvString("REDIS_USERNAME"),
		Password: envutil.GetenvString("REDIS_PASSWORD"),
		DB:       envutil.GetenvInt("REDIS_DB"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "initializeRedis")
	}
	return client, nil
}
