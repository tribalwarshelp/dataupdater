package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/tribalwarshelp/shared/mode"

	_cron "github.com/tribalwarshelp/cron/cron"

	"github.com/go-pg/pg/v10"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func init() {
	os.Setenv("TZ", "UTC")

	if mode.Get() == mode.DevelopmentMode {
		godotenv.Load(".env.development")
	}
}

func main() {
	db := pg.Connect(&pg.Options{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
		Addr:     os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		PoolSize: runtime.NumCPU() * 5,
	})
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	c := cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))),
	))
	if err := _cron.Attach(c, _cron.Config{
		DB:                   db,
		MaxConcurrentWorkers: mustParseEnvToInt("MAX_CONCURRENT_WORKERS"),
	}); err != nil {
		log.Fatal(err)
	}
	c.Start()
	defer c.Stop()

	log.Print("Cron is running!")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	log.Print("shutting down")
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
