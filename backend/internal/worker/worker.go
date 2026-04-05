package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/vector"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type Queue interface {
	DequeueBlock(ctx context.Context, timeoutSeconds int, keys ...string) (queueKey string, payload []byte, err error)
	Publish(ctx context.Context, channel string, message interface{}) error
}

type Embedder interface {
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
	// TODO: Add EmbedImageCaption/Qwen method once backend supports calling it
}

type VectorClient interface {
	IngestImageBatch(ctx context.Context, userID int64, items []vector.IngestItem) error
	IngestTextBatch(ctx context.Context, userID int64, items []vector.IngestItem) error
}

type ObjectStorage interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

type MediaRepo interface {
	Get(ctx context.Context, userID, mediaID int64) (*domain.Media, error)
	UpdateStatus(ctx context.Context, userID, mediaID int64, status string) error
}

type EmbeddingsRepo interface {
	UpsertEmbedding(ctx context.Context, emb *domain.Embedding) error
	GetEmbedding(ctx context.Context, userID, mediaID int64) (*domain.Embedding, error)
	DeleteEmbedding(ctx context.Context, userID, mediaID int64) error
	MarkPending(ctx context.Context, userID, mediaID int64) error
	MarkInIndex(ctx context.Context, userID, mediaID int64) error
	MarkFailed(ctx context.Context, userID, mediaID int64, msg string) error

	// Retry helpers
	ListUnindexed(ctx context.Context, userID int64, limit int) ([]int64, error) // rows where status IN ('pending','failed')
	GetEmbeddingBytes(ctx context.Context, userID, mediaID int64) ([]byte, error)
	ListUnembedded(ctx context.Context, limit int) ([]int64, error) // media IDs with no embeddings row
}

type Enqueuer interface {
	Enqueue(ctx context.Context, key string, payload []byte) error
}

// consume upload jobs, embed, store, index, set status
type EmbedWorker struct {
	Q              Queue
	EmbeddingsRepo EmbeddingsRepo
	MediaRepo      MediaRepo
	Storage        ObjectStorage
	Clip           Embedder
	Vector         VectorClient
	ModelID        string        // e.g. "open_clip:ViT-L/14@336px"
	FastQueueKey   string
	SlowQueueKey   string
	IdleDelay      time.Duration // sleep after BRPOP timeouts/errors
	Log            *logger.Logger
}

func (w *EmbedWorker) Run(ctx context.Context) error {
	if w.FastQueueKey == "" {
		w.FastQueueKey = "jobs:fast_queue"
	}
	if w.SlowQueueKey == "" {
		w.SlowQueueKey = "jobs:slow_queue"
	}
	if w.IdleDelay <= 0 {
		w.IdleDelay = 300 * time.Millisecond
	}

	w.Log.InfoContext(ctx, "worker started", "fast_queue", w.FastQueueKey, "slow_queue", w.SlowQueueKey)
	for {
		select {
		case <-ctx.Done():
			w.Log.InfoContext(ctx, "worker shutting down", "reason", "context cancelled")
			return ctx.Err()
		default:
		}

		key, payload, err := w.Q.DequeueBlock(ctx, 10, w.SlowQueueKey, w.FastQueueKey) // Priotize slow queue natively by ordering
		if err != nil {
			w.Log.ErrorContext(ctx, "dequeue failed", "error", err)
			time.Sleep(w.IdleDelay)
			continue
		}
		if len(payload) == 0 {
			w.Log.DebugContext(ctx, "no job received, sleeping")
			time.Sleep(w.IdleDelay)
			continue
		}

		w.Log.DebugContext(ctx, "job dequeued", "queue", key, "payload_size", len(payload))

		var job ucdto.EmbedJob
		if err := json.Unmarshal(payload, &job); err != nil {
			w.Log.ErrorContext(ctx, "failed to unmarshal job", "error", err)
			continue
		}

		w.Log.DebugContext(ctx, "processing job", "user_id", job.UserID, "media_id", job.MediaID)
		if err := w.processOne(ctx, job, key); err != nil {
			w.Log.ErrorContext(ctx, "job failed", "user_id", job.UserID, "media_id", job.MediaID, "error", err)
		} else {
			w.Log.DebugContext(ctx, "job completed successfully", "media_id", job.MediaID)
		}
	}
}

func (w *EmbedWorker) processOne(ctx context.Context, job ucdto.EmbedJob, sourceQueue string) error {
	w.Log.InfoContext(ctx, "fetching media bytes", "media_id", job.MediaID)

	// Fetch media
	media, err := w.MediaRepo.Get(ctx, job.UserID, job.MediaID)
	if err != nil {
		return fmt.Errorf("get media: %w", err)
	}
	if media == nil {
		return fmt.Errorf("media not found: user=%d, media=%d", job.UserID, job.MediaID)
	}

	// Load bytes from storage
	key, err := utils.ExtractS3Key(media.URL)
	if err != nil {
		return err
	}
	bytes, err := w.Storage.Get(ctx, key)
	if err != nil || len(bytes) == 0 {
		w.Log.ErrorContext(ctx, "failed to load media bytes", "media_id", job.MediaID, "error", err)
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

	// Generate embedding vector (TODO: support Qwen text embedding generation dynamically)
	vec32, err := w.Clip.EmbedImage(ctx, bytes)
	if err != nil {
		w.Log.ErrorContext(ctx, "embedding failed", "media_id", job.MediaID, "error", err)
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

	// Insert into Vector Service
	ingestItem := vector.IngestItem{
		ImageID: job.MediaID,
		Vector:  vec32,
	}

	var newStatus string
	if sourceQueue == w.FastQueueKey {
		if err := w.Vector.IngestImageBatch(ctx, job.UserID, []vector.IngestItem{ingestItem}); err != nil {
			w.Log.ErrorContext(ctx, "failed to ingest image batch", "media_id", job.MediaID, "error", err)
			_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
			return err
		}
		newStatus = "fast_encoded"
	} else {
		// Even if "qwen" logic is missing above, we simulate sending it as TextBatch to vector service.
		if err := w.Vector.IngestTextBatch(ctx, job.UserID, []vector.IngestItem{ingestItem}); err != nil {
			w.Log.ErrorContext(ctx, "failed to ingest text batch", "media_id", job.MediaID, "error", err)
			_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
			return err
		}
		newStatus = "slow_encoded"
	}

	// Record to Postgres Media Repo
	if err := w.MediaRepo.UpdateStatus(ctx, job.UserID, job.MediaID, newStatus); err != nil {
		w.Log.ErrorContext(ctx, "failed to update media status", "error", err)
		return err
	}

	// Publish to Redis Pub/Sub for SSE event mapping
	pubMsg := map[string]interface{}{
		"media_id": job.MediaID,
		"user_id":  job.UserID,
		"status":   newStatus,
	}
	pubBytes, _ := json.Marshal(pubMsg)
	_ = w.Q.Publish(ctx, "status_updates", pubBytes)

	// Keep legacy embeddings tracking functional for RetryWorker
	bytesVec := utils.Float32ToBytes(vec32)
	emb := &domain.Embedding{
		MediaID:   job.MediaID,
		UserID:    job.UserID,
		Model:     job.Modality,
		VecBytes:  bytesVec,
		Status:    "in_index",
		LastError: "",
	}
	_ = w.EmbeddingsRepo.UpsertEmbedding(ctx, emb)

	w.Log.InfoContext(ctx, "embedding successfully ingested via vector-service", "media_id", job.MediaID, "status", newStatus)
	return nil
}
