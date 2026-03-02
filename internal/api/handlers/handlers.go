package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mybox/internal/builder"
	"mybox/internal/runtime"

	"github.com/gin-gonic/gin"
)

const (
	ImageDir     = "/var/lib/mybox/images"
	ContainerDir = "/var/lib/mybox/containers"
)

// --- Request Payloads ---

type RunContainerRequest struct {
	Image   string   `json:"image" binding:"required"`
	Command []string `json:"command"`
	Ports   string   `json:"ports"` // e.g., "8080:80"
}

type BuildImageRequest struct {
	Tag     string `json:"tag" binding:"required"`
	Context string `json:"context" binding:"required"`
}

// --- Container Management Handlers ---

func ListContainersHandler(c *gin.Context) {
	files, err := os.ReadDir(ContainerDir)
	if err != nil {
		log.Printf("[Error] Failed to read containers directory: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list containers"})
		return
	}

	var containers []runtime.ContainerState
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(ContainerDir, file.Name()))
		if err != nil {
			log.Printf("[Warning] Failed to read state file %s: %v\n", file.Name(), err)
			continue
		}

		var state runtime.ContainerState
		if err := json.Unmarshal(data, &state); err == nil {
			containers = append(containers, state)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Containers retrieved successfully",
		"containers": containers,
	})
}

func RunContainerHandler(c *gin.Context) {
	var req RunContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	imageName := req.Image
	if !strings.HasSuffix(imageName, ".tar") {
		imageName += ".tar"
	}
	imagePath := filepath.Join(ImageDir, imageName)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Image '%s' not found", req.Image)})
		return
	}

	var args []string
	if req.Ports != "" {
		// Port çakışması kontrolü: aynı host port zaten kullanımdaysa reddet
		parts := strings.SplitN(req.Ports, ":", 2)
		hostPort := parts[0]
		if inUse, occupiedBy := runtime.IsHostPortInUse(hostPort); inUse {
			c.JSON(http.StatusConflict, gin.H{
				"error":        fmt.Sprintf("Host port %s is already in use", hostPort),
				"container_id": occupiedBy,
				"hint":         fmt.Sprintf("Stop container %s first or choose a different host port", occupiedBy),
			})
			return
		}
		args = append(args, "-p", req.Ports)
	}

	// runtime.Parent konteyner durana kadar bloklandığı (engellendiği) için bir goroutine içinde çalıştırılıyor
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Fatal] Recovered from container panic: %v\n", r)
			}
		}()
		runtime.Parent(args, imagePath)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Container is starting in the background",
		"image":   req.Image,
	})
}

func InspectContainerHandler(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Container ID is required"})
		return
	}

	stateFile := filepath.Join(ContainerDir, containerID+".json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		} else {
			log.Printf("[Error] Failed to read state for container %s: %v\n", containerID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to inspect container"})
		}
		return
	}

	var state runtime.ContainerState
	json.Unmarshal(data, &state)

	c.JSON(http.StatusOK, gin.H{
		"message": "Container details retrieved",
		"details": state,
	})
}

func StopContainerHandler(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Container ID is required"})
		return
	}

	// Normal sonlandırmayı (kill) dene, eğer kök (root) işlem ise sudo kill'e geri dön (fallback)
	err := exec.Command("kill", "-9", containerID).Run()
	if err != nil {
		err = exec.Command("sudo", "kill", "-9", containerID).Run()
	}

	if err != nil {
		log.Printf("[Error] Failed to kill container %s: %v\n", containerID, err)
	}

	_ = os.Remove(filepath.Join(ContainerDir, containerID+".json"))

	c.JSON(http.StatusOK, gin.H{
		"message": "Container stopped and removed successfully",
		"id":      containerID,
	})
}

// --- Image Management Handlers ---

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
