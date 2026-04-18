package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/vector"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	clipdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

const defaultBatchSize = 50

type Queue interface {
	DequeueBlock(ctx context.Context, timeoutSeconds int, keys ...string) (queueKey string, payload []byte, err error)
	// TryDequeue is a non-blocking pop; returns ("", nil, nil) when all queues empty.
	TryDequeue(ctx context.Context, keys ...string) (queueKey string, payload []byte, err error)
	Enqueue(ctx context.Context, key string, payload []byte) error
	Publish(ctx context.Context, channel string, message interface{}) error
}

type Embedder interface {
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
	EmbedImageURL(ctx context.Context, url string) ([]float32, error)
	EmbedImageURLBatch(ctx context.Context, urls []string) ([][]float32, error)
	EmbedImageURLSlow(ctx context.Context, url string) (*clipdto.SlowEncodeResult, error)
	EmbedImageURLSlowBatch(ctx context.Context, urls []string) ([]clipdto.SlowEncodeResult, error)
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
	// All lookup and status-mutation methods take `model` so they target
	// exactly one row of the composite-PK (media_id, model). Without this,
	// updates would clobber every model's row for a given media.
	GetEmbedding(ctx context.Context, userID, mediaID int64, model string) (*domain.Embedding, error)
	DeleteEmbedding(ctx context.Context, userID, mediaID int64) error // intentional: delete all models when media is deleted
	MarkPending(ctx context.Context, userID, mediaID int64, model string) error
	MarkInIndex(ctx context.Context, userID, mediaID int64, model string) error
	MarkFailed(ctx context.Context, userID, mediaID int64, model, msg string) error
	ClaimForProcessing(ctx context.Context, userID, mediaID int64, model string) (bool, error)

	// Retry helpers — return (media_id, user_id, model) triples so the
	// retry loop can route image vs text ingest correctly and target the
	// right row on status updates.
	ListUnindexed(ctx context.Context, userID int64, limit int) ([]MediaRef, error)
	GetEmbeddingBytes(ctx context.Context, userID, mediaID int64, model string) ([]byte, error)
	ListUnembedded(ctx context.Context, limit int) ([]MediaRef, error)
}

// MediaRef carries a media_id + user_id + model triple for multi-user,
// multi-model retry routing. Model may be empty for ListUnembedded results
// (no embedding row exists yet, so nothing to target).
type MediaRef struct {
	MediaID int64
	UserID  int64
	Model   string
}

type Enqueuer interface {
	Enqueue(ctx context.Context, key string, payload []byte) error
}

// rawJob holds a dequeued job before it is processed.
type rawJob struct {
	key     string
	payload []byte
	job     ucdto.EmbedJob
}

// EmbedWorker consumes upload jobs, embeds, stores, indexes, and sets status.
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
	BatchSize      int           // max jobs per GPU call; defaults to defaultBatchSize
	IdleDelay      time.Duration // sleep after BRPOP timeouts/errors
	Log            *logger.Logger
}

func (w *EmbedWorker) Run(ctx context.Context) error {
	if w.IdleDelay <= 0 {
		w.IdleDelay = 300 * time.Millisecond
	}
	if w.BatchSize <= 0 {
		w.BatchSize = defaultBatchSize
	}

	var queues []string
	if w.SlowQueueKey != "" {
		queues = append(queues, w.SlowQueueKey)
	}
	if w.FastQueueKey != "" {
		queues = append(queues, w.FastQueueKey)
	}
	if len(queues) == 0 {
		queues = []string{"jobs:slow_queue", "jobs:fast_queue"}
	}

	w.Log.InfoContext(ctx, "worker started", "queues", queues, "batch_size", w.BatchSize)

	for {
		select {
		case <-ctx.Done():
			w.Log.InfoContext(ctx, "worker shutting down", "reason", "context cancelled")
			return ctx.Err()
		default:
		}

		// Block until at least one job is available.
		key, payload, err := w.Q.DequeueBlock(ctx, 10, queues...)
		if err != nil {
			w.Log.ErrorContext(ctx, "dequeue failed", "error", err)
			time.Sleep(w.IdleDelay)
			continue
		}
		if len(payload) == 0 {
			time.Sleep(w.IdleDelay)
			continue
		}

		var firstJob ucdto.EmbedJob
		if err := json.Unmarshal(payload, &firstJob); err != nil {
			w.Log.ErrorContext(ctx, "failed to unmarshal job (dropping)", "error", err)
			continue
		}

		batch := []rawJob{{key: key, payload: payload, job: firstJob}}

		// Drain more jobs non-blocking until we hit BatchSize.
		for len(batch) < w.BatchSize {
			k2, p2, err := w.Q.TryDequeue(ctx, queues...)
			if err != nil || len(p2) == 0 {
				break
			}
			var j ucdto.EmbedJob
			if err := json.Unmarshal(p2, &j); err != nil {
				w.Log.ErrorContext(ctx, "failed to unmarshal job (dropping)", "error", err)
				continue
			}
			batch = append(batch, rawJob{key: k2, payload: p2, job: j})
		}

		w.Log.DebugContext(ctx, "batch dequeued", "size", len(batch))

		// Split by model so each sub-batch hits the right ML endpoint.
		var fastBatch, slowBatch []rawJob
		for _, rj := range batch {
			if rj.key == w.FastQueueKey {
				fastBatch = append(fastBatch, rj)
			} else {
				slowBatch = append(slowBatch, rj)
			}
		}

		if len(fastBatch) > 0 {
			w.processFastBatch(ctx, fastBatch)
		}
		if len(slowBatch) > 0 {
			w.processSlowBatch(ctx, slowBatch)
		}
	}
}

// processFastBatch claims, embeds with SigLIP, and ingests a batch of fast-queue jobs.
func (w *EmbedWorker) processFastBatch(ctx context.Context, batch []rawJob) {
	const modelName = "siglip"

	// 1. Claim each job; collect the ones we own.
	type claimedJob struct {
		raw rawJob
		url string
	}
	var claimed []claimedJob

	for _, rj := range batch {
		ok, err := w.EmbeddingsRepo.ClaimForProcessing(ctx, rj.job.UserID, rj.job.MediaID, modelName)
		if err != nil {
			// Don't tight-loop on DB errors: sleep briefly before re-enqueueing
			// so a flapping Postgres can't pin this worker to 100% CPU.
			w.Log.ErrorContext(ctx, "claim failed, re-enqueuing", "media_id", rj.job.MediaID, "error", err)
			_ = w.Q.Enqueue(ctx, rj.key, rj.payload)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if !ok {
			w.Log.DebugContext(ctx, "skipping already-claimed media", "media_id", rj.job.MediaID)
			continue
		}

		media, err := w.MediaRepo.Get(ctx, rj.job.UserID, rj.job.MediaID)
		if err != nil || media == nil {
			w.Log.ErrorContext(ctx, "get media failed", "media_id", rj.job.MediaID, "error", err)
			w.markAndNotifyFailed(ctx, rj.job, modelName, utils.TruncateErr(err))
			continue
		}

		s3Key, err := utils.ExtractS3Key(media.URL)
		if err != nil {
			w.markAndNotifyFailed(ctx, rj.job, modelName, utils.TruncateErr(err))
			continue
		}
		url, err := w.Storage.GeneratePresignedURL(ctx, s3Key, 1*time.Hour)
		if err != nil || url == "" {
			w.Log.ErrorContext(ctx, "presign failed", "media_id", rj.job.MediaID, "error", err)
			w.markAndNotifyFailed(ctx, rj.job, modelName, utils.TruncateErr(err))
			continue
		}

		claimed = append(claimed, claimedJob{raw: rj, url: url})
	}

	if len(claimed) == 0 {
		return
	}

	// 2. Single GPU call for all URLs.
	urls := make([]string, len(claimed))
	for i, cj := range claimed {
		urls[i] = cj.url
	}

	vectors, err := w.Clip.EmbedImageURLBatch(ctx, urls)
	if err != nil {
		// ML call failed wholesale — just re-enqueue. Don't MarkFailed: the
		// claim-row will be reclaimed on retry (status moves processing -> processing
		// with fresh updated_at) and spurious 'failed' rows confuse the retry sweep.
		w.Log.ErrorContext(ctx, "fast batch embed failed, re-enqueueing", "error", err)
		for _, cj := range claimed {
			_ = w.Q.Enqueue(ctx, cj.raw.key, cj.raw.payload)
		}
		return
	}

	// 3. Build ingest items grouped by user for the vector service.
	// Group by userID since IngestImageBatch is per-user.
	type userGroup struct {
		items []vector.IngestItem
		jobs  []claimedJob
	}
	groups := map[int64]*userGroup{}
	for i, cj := range claimed {
		ug := groups[cj.raw.job.UserID]
		if ug == nil {
			ug = &userGroup{}
			groups[cj.raw.job.UserID] = ug
		}
		ug.items = append(ug.items, vector.IngestItem{
			ImageID: cj.raw.job.MediaID,
			Vector:  vectors[i],
		})
		ug.jobs = append(ug.jobs, cj)
	}

	for userID, ug := range groups {
		if err := w.Vector.IngestImageBatch(ctx, userID, ug.items); err != nil {
			w.Log.ErrorContext(ctx, "vector ingest failed, re-enqueueing", "user_id", userID, "error", err)
			for _, cj := range ug.jobs {
				_ = w.Q.Enqueue(ctx, cj.raw.key, cj.raw.payload)
			}
			continue
		}

		for i, cj := range ug.jobs {
			w.finalise(ctx, cj.raw.job, ug.items[i].Vector, modelName, "fast_encoded")
		}
	}

	w.Log.InfoContext(ctx, "fast batch complete", "count", len(claimed))
}

// processSlowBatch claims, embeds with Qwen, and ingests a batch of slow-queue jobs.
func (w *EmbedWorker) processSlowBatch(ctx context.Context, batch []rawJob) {
	const modelName = "qwen"

	type claimedJob struct {
		raw rawJob
		url string
	}
	var claimed []claimedJob

	for _, rj := range batch {
		ok, err := w.EmbeddingsRepo.ClaimForProcessing(ctx, rj.job.UserID, rj.job.MediaID, modelName)
		if err != nil {
			w.Log.ErrorContext(ctx, "claim failed, re-enqueuing", "media_id", rj.job.MediaID, "error", err)
			_ = w.Q.Enqueue(ctx, rj.key, rj.payload)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if !ok {
			w.Log.DebugContext(ctx, "skipping already-claimed media", "media_id", rj.job.MediaID)
			continue
		}

		media, err := w.MediaRepo.Get(ctx, rj.job.UserID, rj.job.MediaID)
		if err != nil || media == nil {
			w.Log.ErrorContext(ctx, "get media failed", "media_id", rj.job.MediaID, "error", err)
			w.markAndNotifyFailed(ctx, rj.job, modelName, utils.TruncateErr(err))
			continue
		}

		s3Key, err := utils.ExtractS3Key(media.URL)
		if err != nil {
			w.markAndNotifyFailed(ctx, rj.job, modelName, utils.TruncateErr(err))
			continue
		}
		url, err := w.Storage.GeneratePresignedURL(ctx, s3Key, 1*time.Hour)
		if err != nil || url == "" {
			w.Log.ErrorContext(ctx, "presign failed", "media_id", rj.job.MediaID, "error", err)
			w.markAndNotifyFailed(ctx, rj.job, modelName, utils.TruncateErr(err))
			continue
		}

		claimed = append(claimed, claimedJob{raw: rj, url: url})
	}

	if len(claimed) == 0 {
		return
	}

	// 2. Single GPU call for all URLs.
	urls := make([]string, len(claimed))
	for i, cj := range claimed {
		urls[i] = cj.url
	}

	results, err := w.Clip.EmbedImageURLSlowBatch(ctx, urls)
	if err != nil {
		w.Log.ErrorContext(ctx, "slow batch embed failed, re-enqueueing", "error", err)
		for _, cj := range claimed {
			_ = w.Q.Enqueue(ctx, cj.raw.key, cj.raw.payload)
		}
		return
	}

	// 3. Build ingest items grouped by user.
	type userGroup struct {
		items []vector.IngestItem
		jobs  []claimedJob
		vecs  [][]float32
	}
	groups := map[int64]*userGroup{}
	for i, cj := range claimed {
		ug := groups[cj.raw.job.UserID]
		if ug == nil {
			ug = &userGroup{}
			groups[cj.raw.job.UserID] = ug
		}
		ug.items = append(ug.items, vector.IngestItem{
			ImageID: cj.raw.job.MediaID,
			Vector:  results[i].TextVector,
			Tags:    results[i].Tags,
		})
		ug.jobs = append(ug.jobs, cj)
		ug.vecs = append(ug.vecs, results[i].TextVector)
	}

	for userID, ug := range groups {
		if err := w.Vector.IngestTextBatch(ctx, userID, ug.items); err != nil {
			w.Log.ErrorContext(ctx, "vector text ingest failed, re-enqueueing", "user_id", userID, "error", err)
			for _, cj := range ug.jobs {
				_ = w.Q.Enqueue(ctx, cj.raw.key, cj.raw.payload)
			}
			continue
		}

		for i, cj := range ug.jobs {
			w.finalise(ctx, cj.raw.job, ug.vecs[i], modelName, "slow_encoded")
		}
	}

	w.Log.InfoContext(ctx, "slow batch complete", "count", len(claimed))
}

// markAndNotifyFailed marks the embedding row as failed, updates media.status,
// and fires an SSE event so Flutter gets a real-time "failed" notification.
func (w *EmbedWorker) markAndNotifyFailed(ctx context.Context, job ucdto.EmbedJob, model, reason string) {
	_ = w.EmbeddingsRepo.MarkFailed(ctx, job.UserID, job.MediaID, model, reason)
	_ = w.MediaRepo.UpdateStatus(ctx, job.UserID, job.MediaID, "failed")
	pubMsg := map[string]interface{}{
		"media_id": job.MediaID,
		"user_id":  job.UserID,
		"status":   "failed",
	}
	pubBytes, _ := json.Marshal(pubMsg)
	_ = w.Q.Publish(ctx, "status_updates", pubBytes)
}

// finalise records the embedding result to Postgres and fires an SSE event.
func (w *EmbedWorker) finalise(ctx context.Context, job ucdto.EmbedJob, vec []float32, modelName, status string) {
	if err := w.MediaRepo.UpdateStatus(ctx, job.UserID, job.MediaID, status); err != nil {
		w.Log.ErrorContext(ctx, "failed to update media status", "media_id", job.MediaID, "error", err)
	}

	pubMsg := map[string]interface{}{
		"media_id": job.MediaID,
		"user_id":  job.UserID,
		"status":   status,
	}
	pubBytes, _ := json.Marshal(pubMsg)
	_ = w.Q.Publish(ctx, "status_updates", pubBytes)

	now := time.Now().UTC()
	emb := &domain.Embedding{
		MediaID:   job.MediaID,
		UserID:    job.UserID,
		Model:     modelName,
		VecBytes:  utils.Float32ToBytes(vec),
		Status:    "in_index",
		LastError: "",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = w.EmbeddingsRepo.UpsertEmbedding(ctx, emb)

	w.Log.InfoContext(ctx, "job finalised", "media_id", job.MediaID, "status", status)
}
