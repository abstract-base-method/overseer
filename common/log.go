package common

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
)

func GetLogger(prefix string) *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          prefix,
		// TODO: make these configurable
		Formatter: log.TextFormatter,
		Level:     log.DebugLevel,
	})
}
