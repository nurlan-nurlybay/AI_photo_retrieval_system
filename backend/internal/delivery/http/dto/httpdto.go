package httpdto

import "mime/multipart"

// Search
type SearchTextRequest struct {
	UserID int64  `form:"user_id" binding:"required"`
	Query  string `form:"q" binding:"required,min=2,max=200"`
}

type SearchImageRequest struct {
	UserID int64 `form:"user_id" binding:"required"`
}

type MediaResponse struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
}

// Upload
type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files[]" binding:"required,min=1,max=10"`
	// Optional flags
	Dedup bool `form:"dedup,default=true"`
}

type UploadResponse struct {
	Results []UploadResult `json:"results"`
	Summary struct {
		Total      int `json:"total"`
		Saved      int `json:"saved"`
		Duplicates int `json:"duplicates"`
		Failed     int `json:"failed"`
	} `json:"summary"`
}

type UploadResult struct {
	Filename string      `json:"filename"`
	Status   string      `json:"status"`          // "saved" | "duplicate" | "failed" | "skipped"
	Error    string      `json:"error,omitempty"` // set if Status == "failed"
	Media    *UploadItem `json:"media,omitempty"` // set if saved or duplicate (idempotent)
}

type UploadItem struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumbURL"`
	MimeType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`

	CreatedAt string `json:"createdAt"`         // RFC3339
	TakenAt   string `json:"takenAt,omitempty"` // RFC3339, empty if unknown

	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Checksum string `json:"checksum"`
}
