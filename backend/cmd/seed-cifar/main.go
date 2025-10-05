package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	clipadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	faissadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
	postgresadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/postgres"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

const (
	seaweedMaster = "http://localhost:9333"
	clipURL       = "http://localhost:8003/v1/encode/image"
	faissURL      = "http://localhost:8002/v1/vectors/add"
	datasetPath   = "./cmd/seed-cifar/assets"
)

func main() {
	cfg, err := config.Load("../../config/dev.yaml")
	if err != nil {
		log.Fatal("cfg load fatal")
	}
	log := logger.New(logger.Config(cfg.Log))

	clipClient := clipadapter.NewClient(cfg.Clip)

	faissClient := faissadapter.NewClient(cfg.Faiss)

	ctx := context.Background()
	pgxCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		log.Fatal("parse pgx config: %v", err)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), pgxCfg)
	if err != nil {
		log.Fatal("connect postgres: %v", err)
	}
	defer dbpool.Close()

	log.Info("connected to postgres")

	pgRepo := postgresadapter.NewMediaPG(dbpool)

	files := []string{}
	err = filepath.Walk(datasetPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".png") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal("walk:", err)
	}
	log.Info("Found %d images to seed\n", len(files))

	for _, path := range files {
		label := filepath.Base(filepath.Dir(path))

		url, err := uploadToSeaweed(path)
		if err != nil {
			log.Error("upload failed for %s: %v\n", path, err)
			continue
		}

		id, err := pgRepo.InsertMediaMetadata(ctx, url, label)
		if err != nil {
			log.Error("db insert failed for %s: %v\n", path, err)
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			log.Error("file open err", err)
			continue
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			log.Error("failed to read file", err)
			continue
		}

		vec, err := clipClient.EmbedImage(ctx, data, label)
		if err != nil {
			log.Error("encode failed for %s: %v", path, err)
			continue
		}

		err = faissClient.Insert(ctx, id, vec)
		if err != nil {
			log.Error("faiss insert failed for %s: %v", path, err)
			continue
		}

		log.Info("Seeded %s (label=%s, id=%s)", path, label, id)
	}

	log.Info("Done seeding dataset")
}

func uploadToSeaweed(path string) (string, error) {
	assignResp, err := http.Get(seaweedMaster + "/dir/assign")
	if err != nil {
		return "", err
	}
	defer assignResp.Body.Close()

	var assign struct {
		Fid       string `json:"fid"`
		PublicURL string `json:"publicUrl"`
	}
	if err := json.NewDecoder(assignResp.Body).Decode(&assign); err != nil {
		return "", err
	}

	fileData, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fileData.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", filepath.Base(path))
	_, _ = ioutil.ReadAll(fileData)
	fileBytes, _ := ioutil.ReadFile(path)
	part.Write(fileBytes)
	writer.Close()

	target := fmt.Sprintf("http://%s/%s", assign.PublicURL, assign.Fid)
	req, _ := http.NewRequest("POST", target, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return "", fmt.Errorf("seaweed upload failed: %s", resp.Status)
	}

	return target, nil
}

func insertMedia(db *sql.DB, url, label string) (string, error) {
	var id string
	err := db.QueryRowContext(context.Background(),
		"INSERT INTO media (url, label) VALUES ($1, $2) RETURNING id", url, label).Scan(&id)
	return id, err
}
