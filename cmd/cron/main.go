package main

import (
	"github.com/Kichiyaki/goutil/envutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"

	"github.com/tribalwarshelp/dataupdater/cmd/internal"
	twhelpcron "github.com/tribalwarshelp/dataupdater/pkg/cron"
	"github.com/tribalwarshelp/dataupdater/pkg/postgres"
	"github.com/tribalwarshelp/dataupdater/pkg/queue"
)

func main() {
	redisClient, err := internal.NewRedisClient()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "couldn't connect to Redis"))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logrus.Warn(errors.Wrap(err, "couldn't close the Redis connection"))
		}
	}()

	dbConn, err := postgres.Connect(nil)
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "couldn't connect to the db"))
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			logrus.Warn(errors.Wrap(err, "couldn't close the db connection"))
		}
	}()

	q, err := queue.New(&queue.Config{
		DB:          dbConn,
		Redis:       redisClient,
		WorkerLimit: envutil.GetenvInt("WORKER_LIMIT"),
	})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "couldn't initialize a queue"))
	}

	c, err := twhelpcron.New(&twhelpcron.Config{
		DB:        dbConn,
		RunOnInit: envutil.GetenvBool("RUN_ON_INIT"),
		Queue:     q,
	})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "couldn't initialize a cron instance"))
	}
	if err := c.Start(); err != nil {
		logrus.Fatal(errors.Wrap(err, "couldn't start the cron"))
	}
	defer c.Stop()

	logrus.Info("Cron is up and running!")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	logrus.Info("shutting down")
}
