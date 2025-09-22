package main

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/app"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

func main() {
	cfg, err := config.Load("config.yaml")
	// cfg, err := config.Load("./config/dev.yaml")
	if err != nil {
		panic(err)
	}

	log := logger.New(logger.Config(cfg.Log))
	log.Info("config loaded", "version", cfg.Version)

	ctx := context.Background()

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("failed to create app", "error", err)
	}

	if err := application.Run(); err != nil {
		log.Fatal("failed to run app", "error", err)
	}
}
