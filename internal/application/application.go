package application

import (
	"context"
	"errors"
	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	exitOk          = 0
	exitWatchDog    = 1
	timeoutShutdown = 10 * time.Second
)

type Application struct {
	Log    *logger.Logger
	Config *config.Config
}

func NewApplication() *Application {
	log := logger.NewLogger()
	log.Info("Starting application")

	cfg, err := config.NewDefaultConfig()
	if err != nil {
		log.Errorf("error opening config (%s): %s", err)
		os.Exit(-1)
	}
	log.SetLevel(cfg.GetLogLevel())

	return &Application{
		Log:    log,
		Config: cfg,
	}
}

func (a *Application) Run(f func(ctx context.Context) error) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error {
		return f(ctx)
	})

	go func() {
		<-ctx.Done()
		a.Log.Info("Waiting application shutdown...")
		time.Sleep(timeoutShutdown)
		os.Exit(exitWatchDog)
	}()

	if err := wg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			a.Log.Info("Graceful shutdown")
			return
		}
		a.Log.Fatalf("Graceful shutdown error: %s", err)
	}
	os.Exit(exitOk)
}
