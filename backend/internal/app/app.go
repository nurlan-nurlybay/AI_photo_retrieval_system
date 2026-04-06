package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	httpadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/http"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/imageproc"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/metadata"
	postgresadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/vector"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/worker"
	"golang.org/x/sync/errgroup"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/storage"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
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
	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		postgresDSN = cfg.Postgres.DSN()
	}
	log.Info("loading Postgres", "DSN", postgresDSN)

	dbClient := InitDB(ctx, postgresDSN)
	mediaRepo := postgresadapter.NewMediaRepo(dbClient)
	embeddingsRepo := postgresadapter.NewEmbeddingsRepo(dbClient)

	httpClient := &http.Client{
		Timeout: 1000 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       1000,
			IdleConnTimeout:    900 * time.Second,
			DisableCompression: false,
		},
	}
	fmt.Println("httpClient", httpClient.Timeout)

	// Prep dependencies
	var clipClient usecase.Embedder
	if mlURL := os.Getenv("ML_SERVICE_URL"); mlURL != "" {
		clipClient = clipadapter.NewClientFromURL(mlURL, httpClient)
		log.Info("connected to CLIP client via env", "url", mlURL)
	} else {
		var err error
		clipClient, err = clipadapter.NewClient(ctx, cfg.Clip, httpClient)
		if err != nil {
			log.Fatal("failed to conn clip:", err)
		}
		log.Info("connected to CLIP client via config", "host", cfg.Clip.Host, "port", cfg.Clip.Port)
	}

	vectorSvcURL := os.Getenv("VECTOR_SERVICE_URL")
	if vectorSvcURL == "" {
		vectorSvcURL = fmt.Sprintf("http://%s:%d", cfg.Faiss.Host, cfg.Faiss.Port)
	}
	vectorClient := vector.NewClient(vector.Config{URL: vectorSvcURL}, httpClient)
	log.Info("connected to Vector Service HTTP client", "url", vectorSvcURL)

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.Redis.Addr = redisURL
	}
	redisClient, err := redisadapter.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("failed to conn redis:", err)
	}
	log.Info("connected to Redis client", "addr", cfg.Redis.Addr)

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("failed to load aws config:", err)
	}
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		bucketName = "media"
	}
	store := storage.NewS3Client(awsCfg, bucketName)
	log.Info("connected to S3 client", "bucket", bucketName)

	// Image processing libs
	// TODO: cfg for vips and exif
	imgProc := imageproc.NewVipsProcessor(512, 100)
	metaExt := metadata.NewExifExtractor()

	// Setup app services
	searchSvc := usecase.NewSearchService(mediaRepo, clipClient, vectorClient, store, log)
	mediaSvc := usecase.NewMediaService(vectorClient, mediaRepo, store, redisClient, imgProc, metaExt, log)

	// Setup workers
	ew := &worker.EmbedWorker{
		Q:              redisClient,
		EmbeddingsRepo: embeddingsRepo,
		MediaRepo:      mediaRepo,
		Storage:        store,
		Clip:           clipClient,
		Vector:         vectorClient,
		ModelID:        "open_clip:ViT-L/14@336px",
		FastQueueKey:   "jobs:fast_queue",
		SlowQueueKey:   "jobs:slow_queue",
		IdleDelay:      2 * time.Second,
		Log:            log,
	}
	rw := &worker.RetryWorker{
		EmbeddingsRepo:          embeddingsRepo,
		VectorClient:            vectorClient,
		Queue:                   redisClient,
		QueueKey:                "jobs:fast_queue",
		Interval:                30 * time.Second,
		Batch:                   500,
		AlreadyExistsSubstrings: []string{"already exists", "duplicate id"},
	}

	// Wire handlers
	router := httpadapter.SetupRoutes(redisClient, searchSvc, mediaSvc, log)

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

	// EmbedWorker Pool
	fastWorkers := 10
	if val := os.Getenv("EMBED_FAST_WORKERS"); val != "" {
		if c, err := strconv.Atoi(val); err == nil {
			fastWorkers = c
		}
	}
	slowWorkers := 5
	if val := os.Getenv("EMBED_SLOW_WORKERS"); val != "" {
		if c, err := strconv.Atoi(val); err == nil {
			slowWorkers = c
		}
	}

	a.Logger.Info("starting EmbedWorker pools", "fastWorkers", fastWorkers, "slowWorkers", slowWorkers)
	for i := 0; i < fastWorkers; i++ {
		g.Go(func() error {
			ew := *a.EmbedWorker
			ew.FastQueueKey = "jobs:fast_queue"
			ew.SlowQueueKey = ""
			return ew.Run(ctx)
		})
	}
	for i := 0; i < slowWorkers; i++ {
		g.Go(func() error {
			ew := *a.EmbedWorker
			ew.FastQueueKey = ""
			ew.SlowQueueKey = "jobs:slow_queue"
			return ew.Run(ctx)
		})
	}

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
