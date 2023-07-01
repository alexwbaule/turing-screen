package logger

import (
	"golang.org/x/exp/slog"
	"os"
)

var (
	Version = "development"
	Build   = "0"
)

type Logger struct {
	*slog.Logger
}

func NewLogger() *slog.Logger {
	opts := slog.HandlerOptions{
		AddSource: false,
	}
	handler := slog.NewJSONHandler(os.Stdout, &opts)
	return slog.New(handler).With("version", Version, "build", Build)
}
