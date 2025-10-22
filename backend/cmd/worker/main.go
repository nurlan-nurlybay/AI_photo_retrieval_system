package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	faissadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
	postgresadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/seaweedfs"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/worker"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

func main() {
	cfg, err := config.Load("./config/dev.yaml")
	if err != nil {
		panic(err)
	}

	log := logger.New(logger.Config(cfg.Log))
	log.WithGroup("WORKER")
	log.Info("config loaded", "version", cfg.Version)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	// Create connection pool
	pgxpool, err := postgresadapter.InitDB(ctx, cfg)
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
	}
	log.Info("connected to postgres")

	// Prep dependencies
	store, err := seaweedfs.NewLocalFS("./var/uploads", "http://localhost:8080/uploads")
	if err != nil {
		log.Fatal("failed to connect to seaweedfs:", err)
	}
	workerRepo := postgresadapter.NewWorkRepo(pgxpool, store)
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
	log.Info("connected to redis client")

	ew := &worker.EmbedWorker{
		Q: redisClient, Repo: workerRepo, Clip: clipClient, Faiss: faissClient,
		ModelID: "open_clip:ViT-L/14@336px", QueueKey: "jobs:embed", IdleDelay: 2 * time.Second,
		Log: log,
	}
	rw := &worker.RetryWorker{
		Repo: workerRepo, Faiss: faissClient,
		Interval: 30 * time.Second, Batch: 500,
		AlreadyExistsSubstrings: []string{"already exists", "duplicate id"},
	}

	go func() { _ = ew.Run(ctx) }()
	go func() { _ = rw.Run(ctx) }()
	<-ctx.Done()
}
