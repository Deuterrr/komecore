package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	applogger "komecore/pkg/logger"

	"github.com/go-chi/chi/v5"
)

const komecoreBanner = `
▄▄ ▄▄  ▄▄▄   ▄▄ ▄▄▄  ▄▄▄▄  ▄▄▄▄  ▄▄▄  ▄▄▄▄   ▄▄▄▄
██▄▀▀ ▄█▀██ ▄█▀█▀██ ▄█▀▀▀ ██▀██ ▄█▀██ ██▀██ ▄█▀▀▀
██▀█▄ ▓▓ ▓▓ ▓▓   ▓▓ ▓▓▀▀  ██ ▀▀ ▓▓ ▓▓ ▓▓▀█▄ ▓▓▀▀
▀▀ ▀▀  ▀▀▀  ▀▀   ▀▀ ▀▀▀▀▀ ▀▀▀▀▀  ▀▀▀  ▀▀ ▀▀ ▀▀▀▀▀
`

type App struct {
	cfg          Config
	dependencies *Dependency
	container    *Container
	scheduler    *Scheduler
	router       *chi.Mux
	logger       applogger.Logger
	cancelJobs   context.CancelFunc
}

func New(cfg Config) (*App, error) {
	dependencies, err := NewDependency(cfg)
	if err != nil {
		return nil, err
	}

	c := NewContainer(cfg, dependencies)
	r := NewRouter(c)
	scheduler := NewScheduler(cfg, c, c.Logger)

	initializer := NewInitializer(c, cfg.App, cfg.DB)
	if err := initializer.Run(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to run startup initializer: %w", err)
	}

	return &App{
		cfg:          cfg,
		dependencies: dependencies,
		container:    c,
		scheduler:    scheduler,
		router:       r,
		logger:       c.Logger,
	}, nil
}

func (a *App) Run() error {
	ctx := context.Background()
	serverTimeout, shutdownTimeout := resolveTimeouts(a.cfg)

	fmt.Print(komecoreBanner)
	firstWords := fmt.Sprintf("rise and shine, %s!", a.cfg.App.Name)
	a.logger.Info(ctx, firstWords,
		applogger.Field{Key: "host", Value: a.cfg.App.Host},
		applogger.Field{Key: "port", Value: a.cfg.App.Port},
		applogger.Field{Key: "server_timeout", Value: serverTimeout.String()},
		applogger.Field{Key: "shutdown_timeout", Value: shutdownTimeout.String()},
	)

	// Start background jobs after emitting server startup log.
	jobCtx, cancel := context.WithCancel(context.Background())
	a.cancelJobs = cancel

	if a.scheduler != nil {
		a.scheduler.Start(jobCtx)
	}

	addr := a.cfg.App.Host + ":" + a.cfg.App.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
		IdleTimeout:  serverTimeout,
	}
	serverErrors := make(chan error, 1)

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("http server error: %w", err)

	case sig := <-shutdown:
		return a.gracefulShutdown(ctx, server, shutdownTimeout, sig)
	}
}

func (a *App) Close() {
	if a == nil {
		return
	}

	// Stop background jobs first so they don't race
	// against the DB connection being closed.
	if a.cancelJobs != nil {
		a.cancelJobs()
		a.cancelJobs = nil
	}

	if a.scheduler != nil {
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = a.scheduler.Wait(drainCtx)
		drainCancel()
	}

	if a.dependencies != nil {
		a.dependencies.Close()
		a.dependencies = nil
	}
}

func (a *App) gracefulShutdown(ctx context.Context, server *http.Server, shutdownTimeout time.Duration, sig os.Signal) error {
	a.logger.Info(ctx, "initiate shutdown...",
		applogger.Field{Key: "signal", Value: sig.String()},
		applogger.Field{Key: "timeout", Value: shutdownTimeout.String()},
	)

	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, shutdownTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error(ctx, "forced http server shutdown due to error",
			applogger.Field{Key: "error", Value: err.Error()},
		)
		_ = server.Close()
	}

	a.Close()
	lastWords := fmt.Sprintf("Now, farewell, and remember all %s logs!", a.cfg.App.Codename)
	a.logger.Info(ctx, lastWords)

	return nil
}

func resolveTimeouts(cfg Config) (serverTimeout, shutdownTimeout time.Duration) {
	serverSec := cfg.App.ServerTimeout
	if serverSec <= 0 {
		serverSec = 15
	}
	serverTimeout = time.Duration(serverSec) * time.Second

	shutdownSec := cfg.App.ShutdownTimeout
	if shutdownSec <= 0 {
		shutdownSec = 15
	}
	shutdownTimeout = time.Duration(shutdownSec) * time.Second

	return serverTimeout, shutdownTimeout
}
