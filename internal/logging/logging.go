package logging

import (
	"fmt"
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	var file *os.File
	logDir := "/opt/when-bus/when-bus.log"

	if os.Getenv("ENV") == "prod" {
		file, _ = os.OpenFile(logDir, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	} else {
		file = os.Stdout
	}

	logger = slog.New(slog.NewJSONHandler(file, nil))

}

func Debug(msg string, a ...interface{}) {
	logger.Debug(fmt.Sprintf(msg, a...))
}

func Info(msg string, a ...interface{}) {
	logger.Info(fmt.Sprintf(msg, a...))
}

func Warn(msg string, a ...interface{}) {
	logger.Warn(fmt.Sprintf(msg, a...))
}

func Error(msg string, a ...interface{}) {
	logger.Error(fmt.Sprintf(msg, a...))
}

func Fatal(msg string, a ...interface{}) {
	logger.Error(msg, a...)
	os.Exit(1)
}
