package http

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	httpdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/delivery/http/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	ucdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type ImageHandler struct {
	uploadUC  usecase.MediaService
	validator *validator.Validate
	logger    *logger.Logger
	maxFiles  int
}

func NewImageHandler(image usecase.MediaService, v *validator.Validate, l *logger.Logger) *ImageHandler {
	return &ImageHandler{
		uploadUC:  image,
		validator: v,
		logger:    l,
		maxFiles:  10,
	}
}

func (h *ImageHandler) ImageUpload(c *gin.Context) {
	log := h.logger.With("handler", "ImageUpload")
	log.Info("received req")

	userID := c.GetInt64("userID")
	if userID == 0 {
		userID = 404
	}

	var req httpdto.UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form: " + err.Error()})
		return
	}

	if len(req.Files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"err": "no files[] provided"})
		return
	}
	if len(req.Files) > h.maxFiles {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many files (max 10)"})
		return
	}

	// Map DTO -> usecase inputs
	inputs := make([]ucdto.UploadInput, 0, len(req.Files))
	for _, fh := range req.Files {
		// read into memory
		// TODO: switch to streaming/tempfile
		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open file " + fh.Filename})
			return
		}
		b, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read file " + fh.Filename})
			return
		}

		mt := fh.Header.Get("Content-Type")
		inputs = append(inputs, ucdto.UploadInput{
			UserID:   userID,
			Filename: fh.Filename,
			MimeType: mt,
			Size:     int64(len(b)),
			Body:     bytes.NewReader(b), // io.ReadSeeker
		})
	}

	results, err := h.uploadUC.UploadBatch(c.Request.Context(), inputs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Map usecase -> DTO response
	resp := httpdto.UploadResponse{
		Results: make([]httpdto.UploadResult, 0, len(results)),
	}
	for i, r := range results {
		var ui *httpdto.UploadItem

		if r.Media != nil {
			item := r.Media
			ui = &httpdto.UploadItem{
				ID:        item.ID,
				URL:       item.URL,
				ThumbURL:  item.ThumbURL,
				MimeType:  item.MimeType,
				SizeBytes: item.SizeBytes,
				Width:     item.Metadata.Width,
				Height:    item.Metadata.Height,
				Checksum:  item.Checksum,
			}
			if !item.CreatedAt.IsZero() {
				ui.CreatedAt = item.CreatedAt.UTC().Format(time.RFC3339)
			}
		}

		resp.Results = append(resp.Results, httpdto.UploadResult{
			Filename: inputs[i].Filename,
			Status:   string(r.Status),
			Media:    ui,
			Error:    r.Err.Error(),
		})
	}

	resp.Summary.Total = len(resp.Results)
	for _, r := range resp.Results {
		switch r.Status {
		case "saved":
			resp.Summary.Saved++
		case "duplicate":
			resp.Summary.Duplicates++
		case "failed":
			resp.Summary.Failed++
		}
	}

	c.JSON(http.StatusOK, resp)
}
