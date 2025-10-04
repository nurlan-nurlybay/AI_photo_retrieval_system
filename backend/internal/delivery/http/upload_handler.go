package http

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type ImageHandler struct {
	imageUC   *usecase.ImageUploadUsecase
	validator *validator.Validate
	logger    *logger.Logger
}

func NewImageHandler(image *usecase.ImageUploadUsecase, v *validator.Validate, l *logger.Logger) *ImageHandler {
	return &ImageHandler{imageUC: image, validator: v, logger: l}
}

func (h *ImageHandler) ImageUpload(c *gin.Context) {
	log := h.logger.With("handler", "UploadHandler")
	log.Info("received req")
}

type Metadata struct {
}

func extractExiMetadata(imageBytes []byte) (*Metadata, error) {
	// exif.RegisterParsers(mknote.All...)

	return nil, nil
}
