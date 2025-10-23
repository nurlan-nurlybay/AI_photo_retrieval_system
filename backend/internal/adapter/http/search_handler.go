package http

import (
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

	mediaWithScore, err := h.searchUC.SearchByText(c.Request.Context(), req.UserID, req.Query, 10)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": mediaWithScore})
}

func (h *SearchHandler) SearchByImage(c *gin.Context) {
	log := h.logger.With("handler", "SearchByImage")
	log.Info("received request")

	var req httpdto.SearchImageRequest
	if !BindAndValidate(c, h.validator, &req) {
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing image"})
		return
	}
	defer file.Close()

	imgBytes := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			imgBytes = append(imgBytes, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	mediaWithScore, err := h.searchUC.SearchByImage(c.Request.Context(), req.UserID, imgBytes, 10)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, mediaWithScore)
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
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return false
	}
	if err := validate.Struct(obj); err != nil {
		// pick the first field error
		ve := err.(validator.ValidationErrors)[0]
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ve.Field() + " failed " + ve.Tag(),
		})
		return false
	}
	return true
}
