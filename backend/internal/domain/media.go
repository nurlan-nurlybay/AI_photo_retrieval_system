package domain

import (
	"context"
	"time"
)

type Media struct {
	ID        int64
	UserID    int64
	URL       string
	ThumbURL  string
	MimeType  string // e.g. "image/jpeg", "image/png", "image/heic".
	SizeBytes int64
	Checksum  string // e.g. SHA256 for dedup
	CreatedAt time.Time
	Metadata  Metadata
}

type Metadata struct {
	DateTimeOriginal *time.Time
	Orientation      int
	Width            int
	Height           int
	CameraMake       string
	CameraModel      string
	Software         string
}

// Query helpers
type MediaFilter struct {
	UserID   *string
	After    *time.Time
	Before   *time.Time
	MimeLike []string
}

type Page struct{ Limit, Offset int }
type Sort struct {
	Field string
	Desc  bool
}

type MediaRepository interface {
	Create(ctx context.Context, m *Media) (int64, error)
	Delete(ctx context.Context, uID, mID int64) error
	Get(ctx context.Context, uID, mID int64) (*Media, error)
	List(ctx context.Context, f MediaFilter, p Page, s Sort) ([]*Media, int, error)

	// dedup helper
	GetByChecksum(ctx context.Context, uID int64, checksum string) (*Media, error)
}

type Embedder interface {
	EmbedText(text string) ([]float32, error)
	EmbedImage(data []byte) ([]float32, error)
}

type VectorIndex interface {
	Search(deviceID string, embedding []float32, k int) ([]string, error)
}

type MediaWithScore struct {
	Media *Media
	Score float32
}
