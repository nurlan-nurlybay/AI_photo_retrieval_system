// @title        AI Photo Retrieval System API
// @version      1.0
// @description  REST API for searching and uploading images using AI-based embeddings
// @host         localhost:8080
// @BasePath     /api

package main

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/app"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/storage"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

func main() {
	// Dynamically read the environment variables from .env
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/dev.yaml" // default fallback
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		panic(err)
	}

	log := logger.New(logger.Config(cfg.Log))
	log.Info("config loaded", "version", cfg.Version)

	ctx := context.Background()

	// Initialize S3 Service
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("eu-north-1"))
	if err != nil {
		log.Fatal("unable to load AWS SDK config", "error", err)
	}
	s3Client := storage.NewS3Client(awsCfg, "nurlan-photo-retrieval-storage-136455701428-eu-north-1-an")
	_ = s3Client // initialized as requested

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("failed to create app", "error", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatal("failed to run app", "error", err)
	}
}
