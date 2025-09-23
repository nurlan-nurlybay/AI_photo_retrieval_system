package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-playground/validator/v10"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	faissadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
	postgresadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"
	httpdelivery "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/delivery/http"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type App struct {
	Logger *logger.Logger

	MediaRepo usecase.MediaRepo
	Cache     usecase.Cache
	VectorDB  usecase.VectorIndex
	Embedder  usecase.Embedder
	SearchUC  *usecase.SearchUsecase

	server *http.Server
	// Tracer trace.TracerProvider
}

func New(ctx context.Context, cfg *config.Config, log *logger.Logger) (*App, error) {
	redisCache := redisadapter.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	clipClient := clipadapter.NewClient(cfg.Clip)

	faissClient := faissadapter.NewClient(cfg.Faiss)

	pgRepo, err := postgresadapter.NewMediaRepository(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Fatal("failed to connect to postgres: %v", err)
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	searchUC := usecase.NewSearchUsecase(clipClient, faissClient, pgRepo)

	v := validator.New()
	searchHandler := httpdelivery.NewSearchHandler(searchUC, v, log)
	router := httpdelivery.SetupRoutes(searchHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler: router,
	}

	return &App{
		Logger:    log,
		MediaRepo: pgRepo,
		Cache:     redisCache,
		VectorDB:  faissClient,
		Embedder:  clipClient,
		SearchUC:  searchUC,
		server:    srv,
	}, nil
}

func (a *App) Run() error {
	errCh := make(chan error, 1)
	go func() {
		a.Logger.Info("starting HTTP server", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-shutdownCh:
		a.Logger.Info("shutting down server", "signal", sig)
		if err := a.server.Close(); err != nil {
			a.Logger.Error("server shutdown error", "error", err)
			return fmt.Errorf("server shutdown: %w", err)
		}
	}

	return nil
}

func (a *App) Close() {
	a.Logger.Info("closing app resources...")
	// TODO
}
