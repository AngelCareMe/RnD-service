package logger

import (
	"log"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

func Init(level string) *logrus.Logger {
	logger := logrus.New()

	lvl, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logger.Warnf("bad log level, set default 'info'")
		lvl = logrus.InfoLevel
	}
	logger.SetLevel(lvl)

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	log.SetOutput(os.Stdout)

	return logger
}
