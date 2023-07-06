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
var loglevel = new(slog.LevelVar)

type Logger struct {
	*slog.Logger
}

func NewLogger() *Logger {
	Slog := slog.New(
		slog.NewJSONHandler(os.Stdout,
			&slog.HandlerOptions{
				AddSource: false,
				Level:     loglevel,
			},
		),
	).With("version", Version, "build", Build)
	slog.SetDefault(Slog)
	return &Logger{
		Slog,
	}
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

func (l *Logger) Errorf(format string, v ...any) {
	l.Error(fmt.Sprintf(format, v...))
}

func (l *Logger) Infof(format string, v ...any) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Debugf(format string, v ...any) {
	l.Debug(fmt.Sprintf(format, v...))
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
