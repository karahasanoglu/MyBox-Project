package router

import (
	"mybox/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware, Masaüstü Web-görünümlerinin (Electron/Tauri) bu yerel API ile iletişim kurabilmesini sağlar.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Herhangi bir yerel kaynağa (origin) izin ver (yerel masaüstü uygulamaları için kritik öneme sahip)
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin")

		// Preflight OPTIONS isteklerini işle
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func StartAPI() {
	// Gin router'ını başlat
	router := gin.Default()

	// CORS middleware'ini ekle
	router.Use(CORSMiddleware())

	// API grubu oluştur
	api := router.Group("/api/v1")
	{
		// --- Container Management Routes ---
		containers := api.Group("/containers")
		{
			containers.GET("", handlers.ListContainersHandler)
			containers.POST("/run", handlers.RunContainerHandler)
			containers.GET("/:id", handlers.InspectContainerHandler)
			containers.DELETE("/:id", handlers.StopContainerHandler)
		}

		// --- Image Management Routes ---
		images := api.Group("/images")
		{
			images.GET("", handlers.ListImagesHandler)
			images.POST("/build", handlers.BuildImageHandler)
			images.DELETE("/:name", handlers.RemoveImageHandler)
		}
	}

	// API sunucusunu başlat
	router.Run(":18080")
}
