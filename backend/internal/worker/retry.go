package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/vector"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/utils"
)

// sweep pending/failed, replay to FAISS, set status
type RetryWorker struct {
	EmbeddingsRepo EmbeddingsRepo
	VectorClient   VectorClient
	Interval       time.Duration // e.g. 1 * time.Second
	Batch          int           // e.g. 500

	// Re-enqueue media that have no embeddings row at all
	Queue    Enqueuer
	QueueKey string // e.g. "jobs:embed"

	// If your FAISS service returns a recognizable "already exists" error list substrings here
	AlreadyExistsSubstrings []string
}

func (w *RetryWorker) Run(ctx context.Context) error {
	if w.Interval <= 0 {
		w.Interval = 1 * time.Second
	}
	if w.Batch <= 0 {
		w.Batch = 500
	}

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.step(ctx)
		}
	}
}

func (w *RetryWorker) step(ctx context.Context) {
	// 1. Replay existing pending/failed embeddings into vector service
	refs, err := w.EmbeddingsRepo.ListUnindexed(ctx, 0, w.Batch)
	if err == nil && len(refs) > 0 {
		for _, ref := range refs {
			vb, err := w.EmbeddingsRepo.GetEmbeddingBytes(ctx, ref.UserID, ref.MediaID)
			if err != nil || len(vb) == 0 {
				_ = w.EmbeddingsRepo.MarkFailed(ctx, ref.UserID, ref.MediaID, utils.TruncateErr(err))
				continue
			}
			vec := utils.BytesToFloat32(vb)

			item := vector.IngestItem{
				ImageID: ref.MediaID,
				Vector:  vec,
			}
			if err := w.VectorClient.IngestImageBatch(ctx, ref.UserID, []vector.IngestItem{item}); err != nil {
				if isAlreadyExists(err, w.AlreadyExistsSubstrings) {
					_ = w.EmbeddingsRepo.MarkInIndex(ctx, ref.UserID, ref.MediaID)
					continue
				}
				_ = w.EmbeddingsRepo.MarkFailed(ctx, ref.UserID, ref.MediaID, utils.TruncateErr(err))
				continue
			}
			_ = w.EmbeddingsRepo.MarkInIndex(ctx, ref.UserID, ref.MediaID)
		}
	}

	// 2. Re-enqueue media that have no embeddings row at all (uploaded while ML was down)
	if w.Queue == nil {
		return
	}
	unembedded, err := w.EmbeddingsRepo.ListUnembedded(ctx, w.Batch)
	if err != nil || len(unembedded) == 0 {
		return
	}
	queueKey := w.QueueKey
	if queueKey == "" {
		queueKey = "jobs:embed"
	}
	for _, ref := range unembedded {
		job := ucdto.EmbedJob{
			UserID:   ref.UserID,
			MediaID:  ref.MediaID,
			Modality: "image",
		}
		payload, err := json.Marshal(job)
		if err != nil {
			continue
		}
		_ = w.Queue.Enqueue(ctx, queueKey, payload)
	}
}

// ===== Helpers =====
func isAlreadyExists(err error, needles []string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, n := range needles {
		if n == "" {
			continue
		}
		if contains(msg, n) {
			return true
		}
	}
	return false
}

func contains(haystack, needle string) bool {
	// small inline contains without extra import
	if len(needle) == 0 || len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
