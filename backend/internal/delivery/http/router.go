package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(searchSvc usecase.SearchService, uploadSvc usecase.MediaService, l *logger.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	v := validator.New()

	searchHandler := NewSearchHandler(searchSvc, v, l)
	imageUploadHandler := NewImageHandler(uploadSvc, v, l)

	// Search routes
	r.GET("/api/v1/search/text", searchHandler.SearchByText)
	r.POST("/api/v1/search/image", searchHandler.SearchByImage)

	// Image Upload routes
	r.POST("/api/v1/upload/image", imageUploadHandler.ImageUpload)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	return r
}
