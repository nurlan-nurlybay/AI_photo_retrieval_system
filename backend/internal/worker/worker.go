package worker

import (
	"context"
	"encoding/json"
	"time"
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
}

func (w *EmbedWorker) Run(ctx context.Context) error {
	if w.QueueKey == "" {
		w.QueueKey = "jobs:embed" // use default queue name
	}
	if w.IdleDelay <= 0 {
		w.IdleDelay = 300 * time.Millisecond
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, payload, err := w.Q.DequeueBlock(ctx, w.QueueKey, 10)
		if err != nil || len(payload) == 0 {
			time.Sleep(w.IdleDelay)
			continue
		}

		var job EmbedJob
		if err := json.Unmarshal(payload, &job); err != nil {
			continue
		}
		_ = w.processOne(ctx, job)
	}
}

func (w *EmbedWorker) processOne(ctx context.Context, job EmbedJob) error {
	// fetch original bytes
	bytes, filename, err := w.Repo.LoadMediaBytes(ctx, job.MediaID)
	if err != nil || len(bytes) == 0 {
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}

	// embed via CLIP
	vec64, err := w.Clip.EmbedImage(ctx, bytes, filename)
	if err != nil {
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}

	// pack float32 LE for DB
	vecBytes := f64ToLEf32(vec64)

	// upsert embedding and mark pending
	if err := w.Repo.UpsertEmbedding(ctx, job.MediaID, w.ModelID, vecBytes); err != nil {
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}
	_ = w.Repo.MarkPending(ctx, job.MediaID)

	// push to FAISS
	if err := w.Faiss.Insert(ctx, job.MediaID, vec64); err != nil {
		_ = w.Repo.MarkFailed(ctx, job.MediaID, truncateErr(err))
		return err
	}

	// success
	_ = w.Repo.MarkInIndex(ctx, job.MediaID)
	return nil
}
