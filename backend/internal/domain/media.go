package domain

import (
	"context"
	"time"
)

type MediaID string
type UserID string

type Media struct {
	ID        MediaID
	UserID    UserID
	URL       string
	ThumbURL  string
	MimeType  string
	SizeBytes int64
	Checksum  string    // e.g., SHA256 for dedup
	CreatedAt time.Time // server insert time
	Metadata  Metadata
}

type Metadata struct {
	DateTimeOriginal *time.Time
	Orientation      int
	Width            int
	Height           int
	FileFormat       string // "jpeg","png","heic"
	CameraMake       string
	CameraModel      string
	Software         string
}

// Query helpers
type MediaFilter struct {
	UserID   *UserID
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
	Create(ctx context.Context, m *Media) error
	Delete(ctx context.Context, userID UserID, id MediaID) error
	Get(ctx context.Context, userID UserID, id MediaID) (*Media, error)
	List(ctx context.Context, f MediaFilter, p Page, s Sort) ([]*Media, int, error)

	// Optional dedup helpers
	GetByChecksum(ctx context.Context, userID UserID, checksum string) (*Media, error)
}

type Embedder interface {
	EmbedText(text string) ([]float32, error)
	EmbedImage(data []byte) ([]float32, error)
}

type VectorIndex interface {
	Search(deviceID string, embedding []float32, k int) ([]string, error)
}
