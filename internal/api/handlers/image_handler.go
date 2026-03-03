package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mybox/internal/builder"

	"github.com/gin-gonic/gin"
)

type BuildImageRequest struct {
	Tag     string `json:"tag" binding:"required"`
	Context string `json:"context" binding:"required"`
}

// --- Image Management Handlers ---

// ListImagesHandler, ImageDir içindeki .tar dosyalarını okuyarak JSON formatında döner

func ListImagesHandler(c *gin.Context) {
	files, err := os.ReadDir(ImageDir)
	if err != nil {
		log.Printf("[Error] Failed to read images directory: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images"})
		return
	}

	var images []map[string]interface{}
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".tar" {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(f.Name(), ".tar")
		sizeMB := info.Size() / (1024 * 1024)
		if sizeMB == 0 {
			sizeMB = 1
		}

		images = append(images, map[string]interface{}{
			"name":    name,
			"size":    fmt.Sprintf("%d MB", sizeMB),
			"created": info.ModTime().Format("2006-01-02 15:04"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Images retrieved successfully",
		"images":  images,
	})
}

// BuildImageHandler, verilen context yolunu kullanarak image build işlemini başlatır ve sonucu JSON formatında döner

func BuildImageHandler(c *gin.Context) {
	var req BuildImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	cleanContext := filepath.Clean(req.Context) // Windows uyumluluğu için
	dest := filepath.Join(ImageDir, req.Tag+".tar")

	// Frontend zaman aşımını (timeout) önlemek için asenkron yapılandırma (build)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Fatal] Build process panicked: %v\n", r)
			}
		}()

		builder.BuildImage(cleanContext, dest)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Image build triggered in the background",
		"tag":     req.Tag,
	})
}

// RemoveImageHandler, imageName parametresine göre image'ı siler ve sonucu JSON formatında döner

func RemoveImageHandler(c *gin.Context) {
	imageName := c.Param("name")
	if imageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image name is required"})
		return
	}

	if !strings.HasSuffix(imageName, ".tar") {
		imageName += ".tar"
	}

	imagePath := filepath.Join(ImageDir, imageName)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	if err := os.Remove(imagePath); err != nil {
		log.Printf("[Error] Failed to remove image %s: %v\n", imageName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image removed successfully",
		"image":   strings.TrimSuffix(imageName, ".tar"),
	})
}
