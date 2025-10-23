package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	// Conn to DB
	dbClient := InitDB(ctx, cfg.Postgres.DSN())
	pgRepo := postgresadapter.NewMediaRepo(dbClient)
	log.Info("connected to postgres")

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	// Prep dependencies
	clipClient, err := clipadapter.NewClient(ctx, cfg.Clip, httpClient)
	if err != nil {
		log.Fatal("failed to conn clip:", err)
	}

	faissClient, err := faissadapter.NewClient(ctx, cfg.Faiss, httpClient)
	if err != nil {
		log.Fatal("failed to conn faiss:", err)
	}

	redisClient, err := redisadapter.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("failed to conn redis:", err)
	}
	log.Info("connected to clip, faiss, redis client")

	store, err := seaweedfs.NewSeaweedfs(ctx, cfg.Seaweedfs.BaseURL, httpClient)
	if err != nil {
		log.Fatal("failed to conn seaweedfs:", err)
	}
	log.Info("connected to seaweedfs client")

	// Image processing libs
	// TODO: cfg for vips and exif
	imgProc := imageproc.NewVipsProcessor(512, 100)
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
	// TODO close func
}
