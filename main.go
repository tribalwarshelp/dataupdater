package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/tribalwarshelp/shared/mode"

	_cron "github.com/tribalwarshelp/cron/cron"

	"github.com/go-pg/pg/extra/pgdebug"
	"github.com/go-pg/pg/v10"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func init() {
	os.Setenv("TZ", "UTC")

	if mode.Get() == mode.DevelopmentMode {
		godotenv.Load(".env.development")
	}

	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
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
		db.AddQueryHook(pgdebug.DebugHook{
			Verbose: true,
		})
	}
	logrus.WithFields(dbFields).Info("Connected to the database")

	c := cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))),
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
