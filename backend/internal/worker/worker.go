package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type Queue interface {
	DequeueBlock(ctx context.Context, key string, timeoutSeconds int) (queueKey string, payload []byte, err error)
}

type Embedder interface {
	EmbedImage(ctx context.Context, data []byte, filename string) ([]float64, error)
}

type VectorIndex interface {
	Insert(ctx context.Context, id int64, vector []float64) error
}

type Repo interface {
	// SeaweedFS fetch
	LoadMediaBytes(ctx context.Context, mediaID int64) (bytes []byte, filename string, err error)

	// Embeddings table ops (status lives in table)
	UpsertEmbedding(ctx context.Context, mediaID int64, model string, vecBytes []byte) error
	MarkPending(ctx context.Context, mediaID int64) error
	MarkInIndex(ctx context.Context, mediaID int64) error
	MarkFailed(ctx context.Context, mediaID int64, msg string) error

	// Retry helpers
	ListUnindexed(ctx context.Context, limit int) ([]int64, error) // rows where status IN ('pending','failed')
	GetEmbeddingBytes(ctx context.Context, mediaID int64) ([]byte, error)
}

type EmbedJob struct {
	MediaID int64 `json:"media_id"`
}

// consume upload jobs, embed, store, index, set status
type EmbedWorker struct {
	Q         Queue
	Repo      Repo
	Clip      Embedder
	Faiss     VectorIndex
	ModelID   string        // e.g. "open_clip:ViT-L/14@336px"
	QueueKey  string        // e.g. "jobs:embed"
	IdleDelay time.Duration // sleep after BRPOP timeouts/errors
	Log       *logger.Logger
}

func (w *EmbedWorker) Run(ctx context.Context) error {
	if w.QueueKey == "" {
		w.QueueKey = "jobs:embed" // use default queue name
	}
	if w.IdleDelay <= 0 {
		w.IdleDelay = 300 * time.Millisecond
	}

	w.Log.InfoContext(ctx, "worker started", "queue", w.QueueKey)

	for {
		select {
		case <-ctx.Done():
			w.Log.InfoContext(ctx, "worker shutting down", "reason", "context cancelled")
			return ctx.Err()
		default:
		}

		w.Log.DebugContext(ctx, "waiting for job", "queue", w.QueueKey)
		key, payload, err := w.Q.DequeueBlock(ctx, w.QueueKey, 10)
		if err != nil {
			w.Log.ErrorContext(ctx, "dequeue failed", "error", err)
			time.Sleep(w.IdleDelay)
			continue
		}
		if len(payload) == 0 {
			w.Log.DebugContext(ctx, "no job received, sleeping", "queue", w.QueueKey)
			time.Sleep(w.IdleDelay)
			continue
		}

		w.Log.InfoContext(ctx, "job dequeued", "queue", key, "payload_size", len(payload))

		var job EmbedJob
		if err := json.Unmarshal(payload, &job); err != nil {
			w.Log.ErrorContext(ctx, "failed to unmarshal job", "error", err)
			continue
		}

		w.Log.InfoContext(ctx, "processing job", "media_id", job.MediaID)
		if err := w.processOne(ctx, job); err != nil {
			w.Log.ErrorContext(ctx, "job failed", "media_id", job.MediaID, "error", err)
		} else {
			w.Log.InfoContext(ctx, "job completed successfully", "media_id", job.MediaID)
		}
	}
}

func (w *EmbedWorker) processOne(ctx context.Context, job EmbedJob) error {
	w.Log.DebugContext(ctx, "fetching media bytes", "media_id", job.MediaID)
	bytes, filename, err := w.Repo.LoadMediaBytes(ctx, job.MediaID)
	if err != nil || len(bytes) == 0 {
		w.Log.ErrorContext(ctx, "failed to load media bytes", "media_id", job.MediaID, "error", err)
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}

	w.Log.DebugContext(ctx, "embedding image", "media_id", job.MediaID, "filename", filename)
	vec64, err := w.Clip.EmbedImage(ctx, bytes, filename)
	if err != nil {
		w.Log.ErrorContext(ctx, "embedding failed", "media_id", job.MediaID, "error", err)
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}

	w.Log.DebugContext(ctx, "packing embedding vector", "media_id", job.MediaID, "dims", len(vec64))
	vecBytes := f64ToLEf32(vec64)

	if err := w.Repo.UpsertEmbedding(ctx, job.MediaID, w.ModelID, vecBytes); err != nil {
		w.Log.ErrorContext(ctx, "failed to upsert embedding", "media_id", job.MediaID, "error", err)
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}
	_ = w.Repo.MarkPending(ctx, job.MediaID)

	w.Log.DebugContext(ctx, "inserting into FAISS", "media_id", job.MediaID)
	if err := w.Faiss.Insert(ctx, job.MediaID, vec64); err != nil {
		w.Log.ErrorContext(ctx, "failed to insert into FAISS",
			"media_id", job.MediaID,
			"dims", len(vec64),
			"error", fmt.Sprintf("%+v", err),
		)
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}

	w.Log.InfoContext(ctx, "embedding successfully indexed", "media_id", job.MediaID)
	_ = w.Repo.MarkInIndex(ctx, job.MediaID)
	return nil
}
