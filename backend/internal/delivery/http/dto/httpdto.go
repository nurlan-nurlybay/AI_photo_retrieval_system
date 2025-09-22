package httpdto

// Request 
type SearchTextRequest struct {
	DeviceID string `form:"device_id" binding:"required,uuid4"`
	Query    string `form:"q" binding:"required,min=2,max=200"`
}

type SearchImageRequest struct {
	DeviceID string `form:"device_id" binding:"required,uuid4"`
	// Gin will bind multipart files via c.FormFile, so no direct field here
}

// Response
type MediaResponse struct {
	ID       string `json:"id"`
	DeviceID string `json:"device_id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
}
