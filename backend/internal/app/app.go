package app

import (
	"context"
	"log"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	faissadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
	postgresadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
)

type App struct {
	Redis  *redisadapter.Client
	Clip   *clipadapter.Client
	Faiss  *faissadapter.Client
	PGRepo *postgresadapter.MediaRepository

	SearchUC *usecase.SearchUsecase
	// Tracer trace.TracerProvider
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	redisClient := redisadapter.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	clipClient := clipadapter.NewClient(cfg.Clip.BaseURL)

	faissClient, err := faissadapter.NewClient(cfg.Faiss.Host, cfg.Faiss.Port)
	if err != nil {
		log.Fatalf("failed to connect to faiss: %v", err)
	}

	pgRepo, err := postgresadapter.NewMediaRepository(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	searchUC := usecase.NewSearchUsecase(clipClient, faissClient, pgRepo)

	return &App{Redis: redisClient,
		Clip:     clipClient,
		Faiss:    faissClient,
		PGRepo:   pgRepo,
		SearchUC: searchUC,
		// Tracer:   trace.NewTracerProvider(),
	}, nil
}

func (a *App) Run() error {

	return nil

}

func (a *App) Close() {

}
