package ucdto

import (
	"io"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type UploadStatus string

const (
	StatusSaved     UploadStatus = "saved"
	StatusDuplicate UploadStatus = "duplicate"
	StatusFailed    UploadStatus = "failed"
)

type UploadInput struct {
	UserID   int64
	Filename string
	MimeType string
	Size     int64
	Body     io.ReadSeeker
}

type UploadResult struct {
	Status UploadStatus
	Media  *domain.Media // nil if StatusFailed
	Err    error         // non-nil if StatusFailed
}

type EmbedJob struct {
	MediaID  int64  `json:"media_id"`
	Modality string `json:"modality"` // "image" now; "text" later if you add captions
}
