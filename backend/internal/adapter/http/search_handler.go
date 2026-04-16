package http

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	httpdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/http/dto"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type SearchHandler struct {
	searchUC  usecase.SearchService
	validator *validator.Validate
	logger    *logger.Logger
}

func NewSearchHandler(search usecase.SearchService, validator *validator.Validate, logger *logger.Logger) *SearchHandler {
	return &SearchHandler{searchUC: search, validator: validator, logger: logger}
}

func (h *SearchHandler) SearchByText(c *gin.Context) {
	log := h.logger.With("handler", "SearchByText")
	log.Info("received request")

	var req httpdto.SearchTextRequest
	if !BindAndValidate(c, h.validator, &req) {
		return
	}

	k := req.Limit
	if k <= 0 {
		k = 10
	} else if k > 200 {
		k = 200
	}

	mediaWithScore, usedQwen, err := h.searchUC.SearchByText(c.Request.Context(), req.UserID, req.Query, k)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var resp httpdto.SearchResponse
	resp.UsedQwen = usedQwen
	resp.Results = make([]httpdto.MediaResponse, 0, len(mediaWithScore))

	for _, r := range mediaWithScore {
		if r.Media == nil {
			continue
		}
		if req.Threshold > 0 && r.Score < req.Threshold {
			continue
		}
		resp.Results = append(resp.Results, httpdto.MediaResponse{
			ID:        r.Media.ID,
			UserID:    r.Media.UserID,
			Score:     r.Score,
			LocalPath: r.Media.LocalPath,
		})
	}
	resp.Total = len(resp.Results)

	c.JSON(http.StatusOK, resp)
}

func (h *SearchHandler) SearchByImage(c *gin.Context) {
	log := h.logger.With("handler", "SearchByImage")
	log.Info("received request")

	var req httpdto.SearchImageRequest
	if !BindAndValidate(c, h.validator, &req) {
		return
	}

	// Open uploaded file
	file, err := req.File.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open file"})
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read image"})
		return
	}

	mediaWithScore, usedQwen, err := h.searchUC.SearchByImage(c.Request.Context(), req.UserID, imgBytes, 10)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var resp httpdto.SearchResponse
	resp.UsedQwen = usedQwen
	resp.Results = make([]httpdto.MediaResponse, 0, len(mediaWithScore))

	for _, r := range mediaWithScore {
		if r.Media == nil {
			continue
		}
		resp.Results = append(resp.Results, httpdto.MediaResponse{
			ID:        r.Media.ID,
			UserID:    r.Media.UserID,
			Score:     r.Score,
			LocalPath: r.Media.LocalPath,
		})
	}
	resp.Total = len(resp.Results)

	c.JSON(http.StatusOK, resp)
}

func (h *SearchHandler) handleError(c *gin.Context, err error) {
	switch err {
	case domain.ErrMediaNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// On error writes the HTTP 400 and returns false
func BindAndValidate(c *gin.Context, validate *validator.Validate, obj interface{}) bool {
	ct := c.ContentType()
	var err error

	if ct == "application/json" {
		err = c.ShouldBindJSON(obj)
	} else {
		err = c.ShouldBind(obj) // handles form, query, multipart, etc.
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}

	if err := validate.Struct(obj); err != nil {
		ve := err.(validator.ValidationErrors)[0]
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ve.Field() + " failed " + ve.Tag(),
		})
		return false
	}

	return true
}
