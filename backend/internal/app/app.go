package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	faissadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
	httpadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/http"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/imageproc"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/metadata"
	postgresadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/seaweedfs"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type App struct {
	Logger *logger.Logger

	MediaSvc  usecase.MediaService
	SearchSvc usecase.SearchService

	server *http.Server
	// Tracer trace.TracerProvider
}

func New(ctx context.Context, cfg *config.Config, log *logger.Logger) (*App, error) {
	// Create connection pool and repo
	pgxpool, err := postgresadapter.InitDB(ctx, cfg)
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
	}
	log.Info("connected to postgres")

	// Prep dependencies
	pgRepo := postgresadapter.NewMediaPG(pgxpool)
	clipClient := clipadapter.NewClient(cfg.Clip)
	faissClient, err := faissadapter.NewClient(ctx, cfg.Faiss)
	if err != nil {
		log.Fatal("failed to conn faiss:", err)
	}
	redisClient, err := redisadapter.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("failed to conn redis:", err)
	}
	log.Info("connected to redis client")

	// TODO: actuall seaweedfs implemnetation
	// currently store in ./var/uploads/media/1/abc/
	store, err := seaweedfs.NewLocalFS("./var/uploads", "http://localhost:8080/uploads")
	if err != nil {
		log.Fatal("failed to connect to seaweedfs:", err)
	}

	// Image processing libs
	imgProc := imageproc.NewVipsProcessor(512, 85)
	metaExt := metadata.NewExifExtractor()

	// Setup app services
	searchSvc := usecase.NewSearchService(pgRepo, clipClient, faissClient)
	mediaSvc := usecase.NewMediaService(pgRepo, store, redisClient, imgProc, metaExt, log)

	// Wire handlers
	router := httpadapter.SetupRoutes(searchSvc, mediaSvc, log)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler: router,
	}

	return &App{
		Logger:    log,
		SearchSvc: searchSvc,
		MediaSvc:  mediaSvc,
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
