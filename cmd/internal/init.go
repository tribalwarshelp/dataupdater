package internal

import (
	"github.com/Kichiyaki/appmode"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
)

func init() {
	if err := setENVs(); err != nil {
		logrus.Fatal(err)
	}
	prepareLogger()
}

func setENVs() error {
	err := os.Setenv("TZ", "UTC")
	if err != nil {
		return errors.Wrap(err, "setENVs")
	}

	if appmode.Equals(appmode.DevelopmentMode) {
		err := godotenv.Load(".env.local")
		if err != nil {
			return errors.Wrap(err, "setENVs")
		}
	}

	return nil
}

func prepareLogger() {
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
