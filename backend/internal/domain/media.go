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
	LocalPath string
<<<<<<< HEAD
=======
	Status    string
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
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

// Return response
type MediaWithScore struct {
<<<<<<< HEAD
	Media *Media
	Score float32
=======
	Media    *Media
	Score    float32
	UsedQwen bool
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}

// Query helpers
type MediaFilter struct {
	UserID   *int64
	After    *time.Time
	Before   *time.Time
	MimeLike []string
}

type Page struct {
	Limit  int
	Offset int
}

type Sort struct {
	Field string
	Desc  bool
}

// Repo layer contract
type MediaRepository interface {
	Create(ctx context.Context, m *Media) (int64, error)
	Delete(ctx context.Context, uID, mID int64) error
	Get(ctx context.Context, uID, mID int64) (*Media, error)
	List(ctx context.Context, f MediaFilter, p Page, s Sort) ([]*Media, int, error)

	// dedup helper
	GetByChecksum(ctx context.Context, uID int64, checksum string) (*Media, error)
<<<<<<< HEAD
=======

	// bulk deletion helpers
	ListAllByUser(ctx context.Context, userID int64) ([]*Media, error)
	DeleteAllByUser(ctx context.Context, userID int64) error
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}
