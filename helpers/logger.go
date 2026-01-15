package helpers

import (
	"log/slog"
)

var logger slog.Logger

func SetupLogger(log slog.Logger) {
	logger = log
}

// ReportError is used by Widgets to report internal errors to the syslog
func ReportError(data string) {
	logger.Error(data)
}
