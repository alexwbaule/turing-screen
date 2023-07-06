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
	"time"
)

type Application struct {
	Log    *logger.Logger
	Config *config.Config
	Theme  *theme.Theme
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
		Log:    log,
		Config: cfg,
		Theme:  themeConf,
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
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}()

	err := wg.Wait()

	if err == context.Canceled || err == nil {
		a.Log.Info("Graceful shutdown")
		return
	} else if err != nil {
		a.Log.Fatalf("Graceful shutdown error: %s", err)
	}
	os.Exit(0)
}
