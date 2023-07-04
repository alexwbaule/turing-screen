package logger

import (
	"fmt"
	"golang.org/x/exp/slog"
	"os"
	"strings"
)

var (
	Version = "development"
	Build   = "0"
)
var loglevel *slog.LevelVar

type Logger struct {
	*slog.Logger
}

func NewLogger() *Logger {

	loglevel = new(slog.LevelVar)
	opts := slog.HandlerOptions{
		AddSource: false,
		Level:     loglevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, &opts)

	Slog := slog.New(handler)
	slog.With("version", Version, "build", Build)
	slog.SetDefault(Slog)

	return &Logger{Slog}
}

func (l *Logger) Errorf(format string, v ...any) {
	l.Error(fmt.Sprintf(format, v...))
}

func (l *Logger) Infof(format string, v ...any) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *Logger) SetLevel(level string) {

	if level == "" {
		return
	}

	switch strings.ToLower(level) {
	case "debug":
		loglevel.Set(slog.LevelDebug)

	case "info":
		loglevel.Set(slog.LevelInfo)

	case "warn":
		loglevel.Set(slog.LevelWarn)

	case "error":
		loglevel.Set(slog.LevelError)

	default:
		l.Warnf("Invalid log level %s", level)
		return
	}

	l.Infof("Log level changed to %s", strings.ToUpper(level))
}

func (l *Logger) Fatal(v ...any) {
	l.Error(fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...any) {
	l.Fatal(fmt.Sprintf(format, v...))
}

func (l *Logger) String(k, v string) slog.Attr {
	return slog.String(k, v)
}

func (l *Logger) Warnf(format string, v ...any) {
	l.Warn(fmt.Sprintf(format, v...))
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.Logger.With(args...)}
}
