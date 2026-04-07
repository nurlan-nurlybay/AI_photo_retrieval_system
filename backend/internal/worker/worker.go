package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

<<<<<<< HEAD
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
=======
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/vector"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	clipdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip/dto"
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

type Queue interface {
<<<<<<< HEAD
	DequeueBlock(ctx context.Context, key string, timeoutSeconds int) (queueKey string, payload []byte, err error)
=======
	DequeueBlock(ctx context.Context, timeoutSeconds int, keys ...string) (queueKey string, payload []byte, err error)
	Enqueue(ctx context.Context, key string, payload []byte) error
	Publish(ctx context.Context, channel string, message interface{}) error
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}

type Embedder interface {
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
<<<<<<< HEAD
}

type VectorIndex interface {
	Insert(ctx context.Context, userID, mediaID int64, vector []float32) error
}

type ObjectStorage interface {
	Get(ctx context.Context, key string) ([]byte, error)
=======
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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}

type MediaRepo interface {
	Get(ctx context.Context, userID, mediaID int64) (*domain.Media, error)
<<<<<<< HEAD
=======
	UpdateStatus(ctx context.Context, userID, mediaID int64, status string) error
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}

type EmbeddingsRepo interface {
	UpsertEmbedding(ctx context.Context, emb *domain.Embedding) error
	GetEmbedding(ctx context.Context, userID, mediaID int64) (*domain.Embedding, error)
	DeleteEmbedding(ctx context.Context, userID, mediaID int64) error
	MarkPending(ctx context.Context, userID, mediaID int64) error
	MarkInIndex(ctx context.Context, userID, mediaID int64) error
	MarkFailed(ctx context.Context, userID, mediaID int64, msg string) error

<<<<<<< HEAD
	// Retry helpers
	ListUnindexed(ctx context.Context, userID int64, limit int) ([]int64, error) // rows where status IN ('pending','failed')
	GetEmbeddingBytes(ctx context.Context, userID, mediaID int64) ([]byte, error)
	ListUnembedded(ctx context.Context, limit int) ([]int64, error) // media IDs with no embeddings row
=======
	// Retry helpers — return (media_id, user_id) pairs for correct namespace routing
	ListUnindexed(ctx context.Context, userID int64, limit int) ([]MediaRef, error)
	GetEmbeddingBytes(ctx context.Context, userID, mediaID int64) ([]byte, error)
	ListUnembedded(ctx context.Context, limit int) ([]MediaRef, error)
}

// MediaRef carries a media_id + user_id pair for multi-user retry routing.
type MediaRef struct {
	MediaID int64
	UserID  int64
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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
<<<<<<< HEAD
	Faiss          VectorIndex
	ModelID        string        // e.g. "open_clip:ViT-L/14@336px"
	QueueKey       string        // e.g. "jobs:embed"
=======
	Vector         VectorClient
	ModelID        string        // e.g. "open_clip:ViT-L/14@336px"
	FastQueueKey   string
	SlowQueueKey   string
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	IdleDelay      time.Duration // sleep after BRPOP timeouts/errors
	Log            *logger.Logger
}

func (w *EmbedWorker) Run(ctx context.Context) error {
<<<<<<< HEAD
	if w.QueueKey == "" {
		w.QueueKey = "jobs:embed" // use default queue name
	}
=======
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	if w.IdleDelay <= 0 {
		w.IdleDelay = 300 * time.Millisecond
	}

<<<<<<< HEAD
	w.Log.InfoContext(ctx, "worker started", "queue", w.QueueKey)
=======
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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	for {
		select {
		case <-ctx.Done():
			w.Log.InfoContext(ctx, "worker shutting down", "reason", "context cancelled")
			return ctx.Err()
		default:
		}

<<<<<<< HEAD
		// w.Log.DebugContext(ctx, "waiting for job", "queue", w.QueueKey)
		key, payload, err := w.Q.DequeueBlock(ctx, w.QueueKey, 10)
=======
		key, payload, err := w.Q.DequeueBlock(ctx, 10, queues...)
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		if err != nil {
			w.Log.ErrorContext(ctx, "dequeue failed", "error", err)
			time.Sleep(w.IdleDelay)
			continue
		}
		if len(payload) == 0 {
<<<<<<< HEAD
			w.Log.DebugContext(ctx, "no job received, sleeping", "queue", w.QueueKey)
=======
			w.Log.DebugContext(ctx, "no job received, sleeping")
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
			time.Sleep(w.IdleDelay)
			continue
		}

		w.Log.DebugContext(ctx, "job dequeued", "queue", key, "payload_size", len(payload))

		var job ucdto.EmbedJob
		if err := json.Unmarshal(payload, &job); err != nil {
<<<<<<< HEAD
			w.Log.ErrorContext(ctx, "failed to unmarshal job", "error", err)
=======
			w.Log.ErrorContext(ctx, "failed to unmarshal job (dropping)", "error", err)
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
			continue
		}

		w.Log.DebugContext(ctx, "processing job", "user_id", job.UserID, "media_id", job.MediaID)
<<<<<<< HEAD
		if err := w.processOne(ctx, job); err != nil {
			w.Log.ErrorContext(ctx, "job failed", "user_id", job.UserID, "media_id", job.MediaID, "error", err)
=======
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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		} else {
			w.Log.DebugContext(ctx, "job completed successfully", "media_id", job.MediaID)
		}
	}
}

<<<<<<< HEAD
func (w *EmbedWorker) processOne(ctx context.Context, job ucdto.EmbedJob) error {
=======
func (w *EmbedWorker) processOne(ctx context.Context, job ucdto.EmbedJob, sourceQueue string) error {
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	w.Log.InfoContext(ctx, "fetching media bytes", "media_id", job.MediaID)

	// Fetch media
	media, err := w.MediaRepo.Get(ctx, job.UserID, job.MediaID)
	if err != nil {
		return fmt.Errorf("get media: %w", err)
	}
	if media == nil {
		return fmt.Errorf("media not found: user=%d, media=%d", job.UserID, job.MediaID)
	}

<<<<<<< HEAD
	// Load bytes from storage
=======
	// Generate presigned URL for processing step
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	key, err := utils.ExtractS3Key(media.URL)
	if err != nil {
		return err
	}
<<<<<<< HEAD
	bytes, err := w.Storage.Get(ctx, key)
	if err != nil || len(bytes) == 0 {
		w.Log.ErrorContext(ctx, "failed to load media bytes", "media_id", job.MediaID, "error", err)
=======
	url, err := w.Storage.GeneratePresignedURL(ctx, key, 1*time.Hour)
	if err != nil || url == "" {
		w.Log.ErrorContext(ctx, "failed to generate presigned URL", "media_id", job.MediaID, "error", err)
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
		_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
		return err
	}

<<<<<<< HEAD
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
=======
	var vec32 []float32
	var newStatus string
	var modelName string
	var ingestItem vector.IngestItem

	if sourceQueue == w.FastQueueKey {
		vec, err := w.Clip.EmbedImageURL(ctx, url)
		if err != nil {
			w.Log.ErrorContext(ctx, "embedding failed", "media_id", job.MediaID, "error", err)
			_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, utils.TruncateErr(err))
			return err
		}
		vec32 = vec
		modelName = "siglip"

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
		modelName = "qwen"

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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	return nil
}
