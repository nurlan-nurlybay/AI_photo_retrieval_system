package http

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	_ "github.com/nurlan-nurlybay/AI_photo_retrieval_system/docs"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
<<<<<<< HEAD
=======
	redisadapter "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/redis"
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

<<<<<<< HEAD
func SetupRoutes(searchSvc usecase.SearchService, uploadSvc usecase.MediaService, l *logger.Logger) *gin.Engine {
=======
func SetupRoutes(rdb *redisadapter.Client, searchSvc usecase.SearchService, uploadSvc usecase.MediaService, l *logger.Logger) *gin.Engine {
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: false,
	}))

	v := validator.New()

	searchHandler := NewSearchHandler(searchSvc, v, l)
	imageUploadHandler := NewImageHandler(uploadSvc, v, l)

	// Search routes
	r.POST("/api/search/text", searchHandler.SearchByText)
	r.POST("/api/search/image", searchHandler.SearchByImage)

	// Image Upload routes
	r.POST("/api/upload/image", imageUploadHandler.ImageUpload)

	// User image manage
	r.GET("/api/user/images/list", imageUploadHandler.ListUserImages)
	r.DELETE("/api/user/images/delete", imageUploadHandler.DeleteUserImage)
	r.GET("/api/user/images/get", imageUploadHandler.GetUserImage)

<<<<<<< HEAD
=======
	// Deletion cascade endpoints (3-tier)
	r.DELETE("/api/user/images/delete-batch", imageUploadHandler.DeleteBatchHandler)
	r.DELETE("/api/user/images/clear", imageUploadHandler.ClearGalleryHandler)
	r.DELETE("/api/user/account", imageUploadHandler.DeleteAccountHandler)

	// Streams
	r.GET("/api/status-stream", StatusStreamHandler(rdb))

>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	// Helper routes
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	return r
}
