package helpers

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func SetupLogger(log *slog.Logger) {
	logger = log
}

// ReportError is used by Widgets to report internal errors to the syslog
func ReportError(data string) {
	logger.Error(data)
}
