package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type Queue interface {
	DequeueBlock(ctx context.Context, key string, timeoutSeconds int) (queueKey string, payload []byte, err error)
}

type Embedder interface {
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
}

type VectorIndex interface {
	Insert(ctx context.Context, userID, mediaID int64, vector []float32) error
}

type ObjectStorage interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

type MediaRepo interface {
	Get(ctx context.Context, userID, mediaID int64) (*domain.Media, error)
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
}

// consume upload jobs, embed, store, index, set status
type EmbedWorker struct {
	Q              Queue
	EmbeddingsRepo EmbeddingsRepo
	MediaRepo      MediaRepo
	Storage        ObjectStorage
	Clip           Embedder
	Faiss          VectorIndex
	ModelID        string        // e.g. "open_clip:ViT-L/14@336px"
	QueueKey       string        // e.g. "jobs:embed"
	IdleDelay      time.Duration // sleep after BRPOP timeouts/errors
	Log            *logger.Logger
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

		// w.Log.DebugContext(ctx, "waiting for job", "queue", w.QueueKey)
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

		w.Log.DebugContext(ctx, "job dequeued", "queue", key, "payload_size", len(payload))

		var job ucdto.EmbedJob
		if err := json.Unmarshal(payload, &job); err != nil {
			w.Log.ErrorContext(ctx, "failed to unmarshal job", "error", err)
			continue
		}

		w.Log.DebugContext(ctx, "processing job", "user_id", job.UserID, "media_id", job.MediaID)
		if err := w.processOne(ctx, job); err != nil {
			w.Log.ErrorContext(ctx, "job failed", "user_id", job.UserID, "media_id", job.MediaID, "error", err)
		} else {
			w.Log.DebugContext(ctx, "job completed successfully", "media_id", job.MediaID)
		}
	}
}

func (w *EmbedWorker) processOne(ctx context.Context, job ucdto.EmbedJob) error {
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

	// Generate embedding vector
	vec32, err := w.Clip.EmbedImage(ctx, bytes)
	if err != nil {
		w.Log.ErrorContext(ctx, "embedding failed", "media_id", job.MediaID, "error", err)
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

	// Serialize vector to bytes
	bytesVec := utils.Float32ToBytes(vec32)
	emb := &domain.Embedding{
		MediaID:   job.MediaID,
		UserID:    job.UserID,
		Model:     job.Modality,
		VecBytes:  bytesVec,
		Status:    "pending",
		LastError: "",
	}

	// Upsert embedding
	if err := w.EmbeddingsRepo.UpsertEmbedding(ctx, emb); err != nil {
		w.Log.ErrorContext(ctx, "failed to upsert embedding", "media_id", job.MediaID, "error", err)
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

	// Insert into FAISS/Milvus
	if err := w.Faiss.Insert(ctx, job.UserID, job.MediaID, vec32); err != nil {
		w.Log.ErrorContext(ctx, "failed to insert into FAISS",
			"media_id", job.MediaID, "dims", len(vec32), "error", err,
		)
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

	// Mark as successfully indexed
	_ = w.EmbeddingsRepo.MarkInIndex(ctx, job.UserID, job.MediaID)
	w.Log.InfoContext(ctx, "embedding successfully indexed", "media_id", job.MediaID)
	return nil
}
