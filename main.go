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
	"github.com/tribalwarshelp/cron/cron/queue"
	"github.com/tribalwarshelp/cron/cron/tasks"
	"github.com/tribalwarshelp/cron/db"
	envutils "github.com/tribalwarshelp/cron/utils/env"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func init() {
	os.Setenv("TZ", "UTC")

	if mode.Get() == mode.DevelopmentMode {
		godotenv.Load(".env.local")
	}

	setupLogger()
}

func main() {
	client, err := initializeRedis()
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't connect to Redis"))
	}
	defer client.Close()

	conn, err := db.New(&db.Config{LogQueries: envutils.GetenvBool("LOG_DB_QUERIES")})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't connect to the db"))
	}
	defer conn.Close()
	logrus.Info("Connection with the database has been established")

	queue, err := queue.New(&queue.Config{
		WorkerLimit: envutils.GetenvInt("WORKER_LIMIT"),
		Redis:       client,
	})
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "Couldn't create the task queue"))
	}
	tasks.RegisterTasks(&tasks.Config{
		DB:    conn,
		Queue: queue,
	})
	if err := queue.Start(context.Background()); err != nil {
		logrus.Fatal(err)
	}

	c := cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.PrintfLogger(logrus.WithField("package", "cron"))),
	))
	if err := twhelpcron.Attach(c, twhelpcron.Config{
		DB:           conn,
		RunOnStartup: envutils.GetenvBool("RUN_ON_STARTUP"),
		Queue:        queue,
	}); err != nil {
		logrus.Fatal(err)
	}
	c.Start()
	defer c.Stop()

	logrus.Info("Cron is running!")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	logrus.Info("shutting down")
	if err := queue.Close(); err != nil {
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
