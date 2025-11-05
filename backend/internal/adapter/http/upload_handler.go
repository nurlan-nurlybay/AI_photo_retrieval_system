package http

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	httpdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/http/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
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
		maxFiles:  100,
	}
}

// @Summary      Upload multiple images
// @Description  Uploads between 1 and 100 images in a single multipart/form-data request.
// @Description  - Each file must be a JPEG or PNG.
// @Description  - The `dedup` parameter can be used to enable or disable duplicate image checking.
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Param        files  formData  file  true  "An array of image files (JPEG/PNG)"  collectionFormat:"multi"
// @Param        dedup  formData  bool  false "Enable duplicate check (default: true)"
// @Success      200    {object}  httpdto.UploadResponse
// @Failure      400    {object}  map[string]string "Invalid request or file format"
// @Failure      500    {object}  map[string]string "Internal server error"
// @Router       /api/upload/image [post]
func (h *ImageHandler) ImageUpload(c *gin.Context) {
	log := h.logger.With("handler", "ImageUpload")
	log.Info("received req")

	userID := c.GetInt64("userID")
	// TODO use real userID
	userID = 404

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

	// Map http to usecase inputs
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

	// Map usecase to http response
	resp := httpdto.UploadResponse{
		Results: make([]httpdto.UploadResult, 0, len(results)),
	}
	for i, r := range results {
		var ui *httpdto.MediaItem

		if r.Media != nil {
			item := r.Media
			ui = &httpdto.MediaItem{
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
			if item.Metadata.DateTimeOriginal != nil {
				ui.TakenAt = item.Metadata.DateTimeOriginal.UTC().Format(time.RFC3339)
			}
		}

		var errMsg string
		if r.Err != nil {
			errMsg = r.Err.Error()
			h.logger.Error("upload result contains error",
				"index", i,
				"filename", inputs[i].Filename,
				"status", r.Status,
				"error", errMsg,
			)
		} else if r.Status == ucdto.StatusFailed {
			h.logger.Error("upload result marked failed but error is nil",
				"index", i,
				"filename", inputs[i].Filename,
				"status", r.Status,
			)
		}

		resp.Results = append(resp.Results, httpdto.UploadResult{
			Filename: inputs[i].Filename,
			Status:   string(r.Status),
			Media:    ui,
			Error:    errMsg,
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

func (h *ImageHandler) DeleteUserImage(c *gin.Context) {
	log := h.logger.With("handler", "DeleteUserImage")
	log.Info("received req")

	var req httpdto.UserImageDeleteRequest
	if !BindAndValidate(c, h.validator, &req) {
		return
	}

	err := h.uploadUC.Delete(c.Request.Context(), req.UserID, req.ImageID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	resp := httpdto.UserImageDeleteResponse{
		Deleted: true,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ImageHandler) ListUserImages(c *gin.Context) {
	log := h.logger.With("handler", "ListUserImages")
	log.Info("received req")

	var req httpdto.UserImagesListRequest
	if !BindAndValidate(c, h.validator, &req) {
		return
	}

	filter := domain.MediaFilter{UserID: &req.UserID}
	page := domain.Page{Limit: req.Limit, Offset: req.Offset}
	sort := domain.Sort{}

	media, total, err := h.uploadUC.List(c.Request.Context(), filter, page, sort)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var resp httpdto.UserImagesListResponse
	resp.Results = make([]httpdto.MediaItem, 0, len(media))

	for _, m := range media {
		var createdAt string
		if !m.CreatedAt.IsZero() {
			createdAt = m.CreatedAt.UTC().Format(time.RFC3339)
		}
		var takenAt string
		if m.Metadata.DateTimeOriginal != nil {
			takenAt = m.Metadata.DateTimeOriginal.UTC().Format(time.RFC3339)
		}

		resp.Results = append(resp.Results, httpdto.MediaItem{
			ID:        m.ID,
			URL:       m.URL,
			ThumbURL:  m.ThumbURL,
			MimeType:  m.MimeType,
			SizeBytes: m.SizeBytes,
			Width:     m.Metadata.Width,
			Height:    m.Metadata.Height,
			Checksum:  m.Checksum,
			TakenAt:   takenAt,
			CreatedAt: createdAt,
		})
	}
	resp.Total = total

	c.JSON(http.StatusOK, resp)
}

func (h *ImageHandler) GetUserImage(c *gin.Context) {
	log := h.logger.With("handler", "GetUserImage")
	log.Info("received req")

	var req httpdto.UserImageGetRequest
	if !BindAndValidate(c, h.validator, &req) {
		return
	}

	m, err := h.uploadUC.GetByID(c.Request.Context(), req.UserID, req.ImageID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var createdAt string
	if !m.CreatedAt.IsZero() {
		createdAt = m.CreatedAt.UTC().Format(time.RFC3339)
	}
	var takenAt string
	if m.Metadata.DateTimeOriginal != nil {
		takenAt = m.Metadata.DateTimeOriginal.UTC().Format(time.RFC3339)
	}
	resp := httpdto.MediaItem{
		ID:        m.ID,
		URL:       m.URL,
		ThumbURL:  m.ThumbURL,
		MimeType:  m.MimeType,
		SizeBytes: m.SizeBytes,
		Width:     m.Metadata.Width,
		Height:    m.Metadata.Height,
		Checksum:  m.Checksum,
		TakenAt:   takenAt,
		CreatedAt: createdAt,
	}

	c.JSON(http.StatusOK, resp)
}
