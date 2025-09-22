package app

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/sdk/trace"
)

type App struct {
	grpcServer *service.GRPCServer
	redis      *redis.Client
	tracer     trace.TracerProvider
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {

	return &App{}, nil
}

func (a *App) Run() error {

	return nil

}

func (a *App) Close() {

}
