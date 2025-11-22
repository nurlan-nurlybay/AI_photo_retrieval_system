package httpdto

import "mime/multipart"

// ===== Search Image =====
type SearchTextRequest struct {
	UserID int64  `json:"user_id" binding:"required"`
	Query  string `json:"q" binding:"required,min=2,max=200"`
}

type SearchImageRequest struct {
	UserID int64                 `form:"user_id" binding:"required"`
	File   *multipart.FileHeader `form:"file"    binding:"required"`
}

type SearchResponse struct {
	Results []MediaResponse `json:"results"`
	Total   int             `json:"total"`
}

type MediaResponse struct {
	ID       int64   `json:"id"`
	UserID   int64   `json:"user_id"`
	URL      string  `json:"url"`
	ThumbURL string  `json:"thumb_url"`
	Score    float32 `json:"score,omitempty"`
}

// ===== Upload Image =====
type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files[]" binding:"required,min=1,max=100"`
	Dedup bool                    `form:"dedup,default=true"`
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
	Filename string     `json:"filename"`
	Status   string     `json:"status"`          // "saved" | "duplicate" | "failed" | "skipped"
	Error    string     `json:"error,omitempty"` // present if Status == "failed"
	Media    *MediaItem `json:"media,omitempty"` // present if saved or duplicate
}

// ===== User Images =====
type UserImagesListRequest struct {
	UserID int64 `form:"user_id" binding:"required"`
	Limit  int   `form:"limit,omitempty"`
	Offset int   `form:"offset,omitempty"`
}

type UserImagesListResponse struct {
	Results []MediaItem `json:"results"`
	Total   int         `json:"total"`
}

type UserImageGetRequest struct {
	UserID  int64 `json:"user_id" binding:"required"`
	ImageID int64 `json:"image_id" binding:"required"`
}

type UserImageDeleteRequest struct {
	UserID  int64 `json:"user_id" binding:"required"`
	ImageID int64 `json:"image_id" binding:"required"`
}

type UserImageDeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// GENERIC RESPONSE 
type MediaItem struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumb_url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Checksum  string `json:"checksum"`
	TakenAt   string `json:"taken_at,omitempty"` // RFC3339, empty if unknown
	CreatedAt string `json:"created_at"`         // RFC3339
}
