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
	UserID   domain.UserID
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
