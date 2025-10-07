package worker

import (
	"context"
	"encoding/binary"
	"math"
	"time"
)

// sweep pending/failed, replay to FAISS, set status
type RetryWorker struct {
	Repo     Repo
	Faiss    VectorIndex
	Interval time.Duration // e.g. 30 * time.Second
	Batch    int           // e.g. 500

	// If your FAISS service returns a recognizable "already exists" error list substrings here
	AlreadyExistsSubstrings []string
}

func (w *RetryWorker) Run(ctx context.Context) error {
	if w.Interval <= 0 {
		w.Interval = 30 * time.Second
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
	ids, err := w.Repo.ListUnindexed(ctx, w.Batch)
	if err != nil || len(ids) == 0 {
		return
	}
	for _, id := range ids {
		vb, err := w.Repo.GetEmbeddingBytes(ctx, id)
		if err != nil || len(vb) == 0 {
			_ = w.Repo.MarkFailed(ctx, id, truncateErr(err))
			continue
		}
		vec := bytesToF64LE(vb)

		if err := w.Faiss.Insert(ctx, id, vec); err != nil {
			if isAlreadyExists(err, w.AlreadyExistsSubstrings) {
				_ = w.Repo.MarkInIndex(ctx, id)
				continue
			}
			_ = w.Repo.MarkFailed(ctx, id, truncateErr(err))
			continue
		}
		_ = w.Repo.MarkInIndex(ctx, id)
	}
}

func f64ToLEf32(v []float64) []byte {
	out := make([]byte, 4*len(v))
	for i, x := range v {
		f := float32(x)
		binary.LittleEndian.PutUint32(out[4*i:], math.Float32bits(f))
	}
	return out
}

func bytesToF64LE(b []byte) []float64 {
	n := len(b) / 4
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		u := binary.LittleEndian.Uint32(b[i*4:])
		out[i] = float64(math.Float32frombits(u))
	}
	return out
}

func truncateErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > 500 {
		s = s[:500]
	}
	return s
}

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
