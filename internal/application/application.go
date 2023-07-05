package application

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

type Application struct {
	Log     *logger.Logger
	Context context.Context
	Config  *config.Config
	Theme   *theme.Theme
}

func NewApplication() *Application {
	log := logger.NewLogger()

	log.Info("Starting application")

	cfg, err := config.NewDefaultConfig()
	if err != nil {
		log.Errorf("error opening config (%s): %s", err)
		os.Exit(-1)
	}
	themeName := cfg.GetDeviceTheme()

	themeConf, err := theme.LoadTheme(themeName)
	if err != nil {
		log.Errorf("error opening theme (%s): %s", themeName, err)
		os.Exit(-1)
	}
	return &Application{
		Log:     log,
		Context: context.Background(),
		Config:  cfg,
		Theme:   themeConf,
	}
}

func (a *Application) Run(f func(ctx context.Context) error) {
	ctx, cancel := signal.NotifyContext(a.Context, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error {
		return f(ctx)
	})

	go func() {
		<-ctx.Done()
		a.Log.Info("Waiting application stop...")
		os.Exit(1)
	}()

	if err := wg.Wait(); err != nil {
		if err == context.Canceled {
			a.Log.Info("Graceful stopped")
		}
		a.Log.Fatalf("Graceful shutdown error: %s", err)
	}
	os.Exit(0)
}
