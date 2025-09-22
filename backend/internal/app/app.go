package app

import (
	"context"
	"fmt"
	"net/http"

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
	SearchUC *usecase.SearchUsecase

	server   *http.Server
	// Tracer trace.TracerProvider
}

func New(ctx context.Context, cfg *config.Config, log *logger.Logger) (*App, error) {
	redisCache := redisadapter.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	clipClient := clipadapter.NewClient(cfg.Clip.BaseURL)

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

	return nil

}

func (a *App) Close() {

}
