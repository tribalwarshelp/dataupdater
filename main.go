package main

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/mode"

	_cron "github.com/tribalwarshelp/cron/cron"

	gopglogrusquerylogger "github.com/Kichiyaki/go-pg-logrus-query-logger/v10"
	"github.com/go-pg/pg/v10"
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
	dbOptions := &pg.Options{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
		Addr:     os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		PoolSize: mustParseEnvToInt("DB_POOL_SIZE"),
	}
	dbFields := logrus.Fields{
		"user":     dbOptions.User,
		"database": dbOptions.Database,
		"addr":     dbOptions.Addr,
	}
	db := pg.Connect(dbOptions)
	defer func() {
		if err := db.Close(); err != nil {
			logrus.WithFields(dbFields).Fatalln(err)
		}
	}()
	if strings.ToUpper(os.Getenv("LOG_DB_QUERIES")) == "TRUE" {
		db.AddQueryHook(gopglogrusquerylogger.QueryLogger{
			Entry: logrus.NewEntry(logrus.StandardLogger()),
		})
	}
	logrus.WithFields(dbFields).Info("Connection with the database has been established")

	c := cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.PrintfLogger(logrus.WithField("package", "cron"))),
	))
	if err := _cron.Attach(c, _cron.Config{
		DB:                   db,
		MaxConcurrentWorkers: mustParseEnvToInt("MAX_CONCURRENT_WORKERS"),
		RunOnStartup:         os.Getenv("RUN_ON_STARTUP") == "true",
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
}

func mustParseEnvToInt(key string) int {
	str := os.Getenv(key)
	if str == "" {
		return 0
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return i
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
