package main

import (
	"context"
	"github.com/Kichiyaki/goutil/envutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"

	"github.com/tribalwarshelp/dataupdater/cmd/internal"
	"github.com/tribalwarshelp/dataupdater/postgres"
	"github.com/tribalwarshelp/dataupdater/queue"
)

func main() {
	redisClient, err := internal.NewRedisClient()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't connect to Redis"))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logrus.Warn(errors.Wrap(err, "Couldn't close the Redis connection"))
		}
	}()

	dbConn, err := postgres.Connect(&postgres.Config{SkipDBInitialization: true})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't connect to the db"))
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			logrus.Warn(errors.Wrap(err, "Couldn't close the db connection"))
		}
	}()

	q, err := queue.New(&queue.Config{
		DB:          dbConn,
		Redis:       redisClient,
		WorkerLimit: envutil.GetenvInt("WORKER_LIMIT"),
	})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't initialize a queue"))
	}
	if err := q.Start(context.Background()); err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't start the queue"))
	}

	logrus.Info("Data updater is up and running!")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	logrus.Info("shutting down")
	if err := q.Close(); err != nil {
		logrus.Fatal(err)
	}
}
