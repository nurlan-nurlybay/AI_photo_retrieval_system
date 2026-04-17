package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/vector"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	clipdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type Queue interface {
	DequeueBlock(ctx context.Context, timeoutSeconds int, keys ...string) (queueKey string, payload []byte, err error)
	Enqueue(ctx context.Context, key string, payload []byte) error
	Publish(ctx context.Context, channel string, message interface{}) error
}

type Embedder interface {
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
	EmbedImageURL(ctx context.Context, url string) ([]float32, error)
	EmbedImageURLSlow(ctx context.Context, url string) (*clipdto.SlowEncodeResult, error)
	// TODO: Add EmbedImageCaption/Qwen method once backend supports calling it
}

type VectorClient interface {
	IngestImageBatch(ctx context.Context, userID int64, items []vector.IngestItem) error
	IngestTextBatch(ctx context.Context, userID int64, items []vector.IngestItem) error
}

type ObjectStorage interface {
	GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
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
	ClaimForProcessing(ctx context.Context, userID, mediaID int64, model string) (bool, error)

	// Retry helpers — return (media_id, user_id) pairs for correct namespace routing
	ListUnindexed(ctx context.Context, userID int64, limit int) ([]MediaRef, error)
	GetEmbeddingBytes(ctx context.Context, userID, mediaID int64) ([]byte, error)
	ListUnembedded(ctx context.Context, limit int) ([]MediaRef, error)
}

// MediaRef carries a media_id + user_id pair for multi-user retry routing.
type MediaRef struct {
	MediaID int64
	UserID  int64
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
	if w.IdleDelay <= 0 {
		w.IdleDelay = 300 * time.Millisecond
	}

	var queues []string
	if w.SlowQueueKey != "" {
		queues = append(queues, w.SlowQueueKey)
	}
	if w.FastQueueKey != "" {
		queues = append(queues, w.FastQueueKey)
	}
	if len(queues) == 0 {
		queues = []string{"jobs:slow_queue", "jobs:fast_queue"} // Fallback
	}

	w.Log.InfoContext(ctx, "worker started", "queues", queues)
	for {
		select {
		case <-ctx.Done():
			w.Log.InfoContext(ctx, "worker shutting down", "reason", "context cancelled")
			return ctx.Err()
		default:
		}

		key, payload, err := w.Q.DequeueBlock(ctx, 10, queues...)
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
			w.Log.ErrorContext(ctx, "failed to unmarshal job (dropping)", "error", err)
			continue
		}

		w.Log.DebugContext(ctx, "processing job", "user_id", job.UserID, "media_id", job.MediaID)
		if err := w.processOne(ctx, job, key); err != nil {
			w.Log.WarnContext(ctx, "job failed, re-enqueuing",
				"user_id", job.UserID, "media_id", job.MediaID,
				"queue", key, "error", err)

			// Re-enqueue the original payload back into the same queue.
			// No retry limit — the worker will keep cycling until the
			// ML service is ready (e.g. Qwen model download can take 30+ min).
			if enqErr := w.Q.Enqueue(ctx, key, payload); enqErr != nil {
				w.Log.ErrorContext(ctx, "CRITICAL: failed to re-enqueue job",
					"media_id", job.MediaID, "queue", key, "error", enqErr)
			}

			// Brief backoff to avoid tight-looping on the same failing job
			time.Sleep(5 * time.Second)
		} else {
			w.Log.DebugContext(ctx, "job completed successfully", "media_id", job.MediaID)
		}
	}
}

func (w *EmbedWorker) processOne(ctx context.Context, job ucdto.EmbedJob, sourceQueue string) error {
	// Determine model before doing any work so we can claim the row early.
	modelName := "siglip"
	if sourceQueue != w.FastQueueKey {
		modelName = "qwen"
	}

	// Atomically claim this media so the RetryWorker won't re-enqueue it
	// while we're still processing. Skips if another worker already owns it
	// or if it's already fully indexed.
	claimed, err := w.EmbeddingsRepo.ClaimForProcessing(ctx, job.UserID, job.MediaID, modelName)
	if err != nil {
		return fmt.Errorf("claim for processing: %w", err)
	}
	if !claimed {
		w.Log.DebugContext(ctx, "skipping already-claimed media", "media_id", job.MediaID)
		return nil
	}

	w.Log.InfoContext(ctx, "fetching media bytes", "media_id", job.MediaID)

	// Fetch media
	media, err := w.MediaRepo.Get(ctx, job.UserID, job.MediaID)
	if err != nil {
		return fmt.Errorf("get media: %w", err)
	}
	if media == nil {
		return fmt.Errorf("media not found: user=%d, media=%d", job.UserID, job.MediaID)
	}

	// Generate presigned URL for processing step
	key, err := utils.ExtractS3Key(media.URL)
	if err != nil {
		return err
	}
	url, err := w.Storage.GeneratePresignedURL(ctx, key, 1*time.Hour)
	if err != nil || url == "" {
		w.Log.ErrorContext(ctx, "failed to generate presigned URL", "media_id", job.MediaID, "error", err)
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

	var vec32 []float32
	var newStatus string
	var ingestItem vector.IngestItem

	if sourceQueue == w.FastQueueKey {
		vec, err := w.Clip.EmbedImageURL(ctx, url)
		if err != nil {
			w.Log.ErrorContext(ctx, "embedding failed", "media_id", job.MediaID, "error", err)
			_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
			return err
		}
		vec32 = vec

		ingestItem = vector.IngestItem{
			ImageID: job.MediaID,
			Vector:  vec32,
		}

		if err := w.Vector.IngestImageBatch(ctx, job.UserID, []vector.IngestItem{ingestItem}); err != nil {
			w.Log.ErrorContext(ctx, "failed to ingest image batch", "media_id", job.MediaID, "error", err)
			_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
			return err
		}
		newStatus = "fast_encoded"
	} else {
		slowRes, err := w.Clip.EmbedImageURLSlow(ctx, url)
		if err != nil {
			w.Log.ErrorContext(ctx, "slow embedding failed", "media_id", job.MediaID, "error", err)
			_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
			return err
		}
		vec32 = slowRes.TextVector

		ingestItem = vector.IngestItem{
			ImageID: job.MediaID,
			Vector:  vec32,
			Tags:    slowRes.Tags,
		}

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
	now := time.Now().UTC()
	emb := &domain.Embedding{
		MediaID:   job.MediaID,
		UserID:    job.UserID,
		Model:     modelName,
		VecBytes:  bytesVec,
		Status:    "in_index",
		LastError: "",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = w.EmbeddingsRepo.UpsertEmbedding(ctx, emb)

	w.Log.InfoContext(ctx, "embedding successfully ingested via vector-service", "media_id", job.MediaID, "status", newStatus)
	return nil
}
