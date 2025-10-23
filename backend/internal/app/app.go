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
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/worker"
	"golang.org/x/sync/errgroup"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/seaweedfs"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type App struct {
	Logger *logger.Logger

	MediaSvc  usecase.MediaService
	SearchSvc usecase.SearchService

	EmbedWorker *worker.EmbedWorker
	RetryWorker *worker.RetryWorker

	server *http.Server
	// Tracer trace.TracerProvider
}

func New(ctx context.Context, cfg *config.Config, log *logger.Logger) (*App, error) {
	// Conn to DB
	log.Info("loading Postgres", "DSN", cfg.Postgres.DSN())

	dbClient := InitDB(ctx, cfg.Postgres.DSN())
	mediaRepo := postgresadapter.NewMediaRepo(dbClient)
	embeddingsRepo := postgresadapter.NewEmbeddingsRepo(dbClient)
	log.Info("connected to Postgres", "DSN", cfg.Postgres.DSN())

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
	log.Info("connected to CLIP client", cfg.Clip.Host, cfg.Clip.Port)

	faissClient, err := faissadapter.NewClient(ctx, cfg.Faiss, httpClient)
	if err != nil {
		log.Fatal("failed to conn faiss:", err)
	}
	log.Info("connected to FAISS client", cfg.Faiss.Host, cfg.Faiss.Port)

	redisClient, err := redisadapter.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("failed to conn redis:", err)
	}
	log.Info("connected to Redis client", "addr", cfg.Redis.Addr)

	store, err := seaweedfs.NewSeaweedfs(ctx, cfg.Seaweedfs.BaseURL, httpClient)
	if err != nil {
		log.Fatal("failed to conn seaweedfs:", err)
	}
	log.Info("connected to Seaweedfs client")

	// Image processing libs
	// TODO: cfg for vips and exif
	imgProc := imageproc.NewVipsProcessor(512, 100)
	metaExt := metadata.NewExifExtractor()

	// Setup app services
	searchSvc := usecase.NewSearchService(mediaRepo, clipClient, faissClient, log)
	mediaSvc := usecase.NewMediaService(mediaRepo, store, redisClient, imgProc, metaExt, log)

	// Setup workers
	ew := &worker.EmbedWorker{
		Q:              redisClient,
		EmbeddingsRepo: embeddingsRepo,
		MediaRepo:      mediaRepo,
		Storage:        store,
		Clip:           clipClient,
		Faiss:          faissClient,
		ModelID:        "open_clip:ViT-L/14@336px",
		QueueKey:       "jobs:embed",
		IdleDelay:      2 * time.Second,
		Log:            log,
	}
	rw := &worker.RetryWorker{
		EmbeddingsRepo: embeddingsRepo,
		Faiss:          faissClient,
		Interval:       30 * time.Second, Batch: 500,
		AlreadyExistsSubstrings: []string{"already exists", "duplicate id"},
	}

	// Wire handlers
	router := httpadapter.SetupRoutes(searchSvc, mediaSvc, log)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler: router,
	}

	return &App{
		Logger:      log,
		SearchSvc:   searchSvc,
		MediaSvc:    mediaSvc,
		EmbedWorker: ew,
		RetryWorker: rw,
		server:      srv,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	// HTTP server
	g.Go(func() error {
		a.Logger.Info("starting HTTP server", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	})

	// EmbedWorker
	g.Go(func() error {
		a.Logger.Info("starting EmbedWorker")
		return a.EmbedWorker.Run(ctx)
	})

	// RetryWorker
	g.Go(func() error {
		a.Logger.Info("starting RetryWorker")
		return a.RetryWorker.Run(ctx)
	})

	// Wait for shutdown signal in separate goroutine
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownCh)

	go func() {
		<-shutdownCh
		cancel() // cancel all workers
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			a.Logger.Error("server shutdown error", "error", err)
		}
	}()

	// Wait for all goroutines or first error
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (a *App) Close() {
	a.Logger.Info("closing app resources...")
	// TODO close func
}
