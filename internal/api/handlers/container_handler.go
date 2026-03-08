package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mybox/internal/cgroup"
	"mybox/internal/runtime"

	"github.com/gin-gonic/gin"
)

// --- Request Payloads ---

type RunContainerRequest struct {
	Image   string   `json:"image" binding:"required"`
	Command []string `json:"command"`
	Ports   string   `json:"ports"` // e.g., "8080:80"
}

// Kaynak güncelleme istekleri için yapı
type UpdateResourcesRequest struct {
	Memory string `json:"memory"` // Örn: "512m", "1g"
	CPUs   string `json:"cpus"`   // Örn: "0.5", "2"
}

// --- Container Management Handlers ---

// ListContainersHandler, tüm konteyner durumlarını okuyarak JSON formatında döner

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

// Container çalıştırma işlemi: imageName'e göre image'ı bul, port çakışması kontrolü yap, runtime.Parent içinde çalıştır (arka planda)

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

// Container durdurma ve silme işlemi: normal kill dene, başarısız olursa sudo kill'e geri dön (fallback)

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

// StopContainerHandler, konteyneri güvenli bir şekilde (graceful) durdurmaya çalışır.
// Yanıt vermezse SIGKILL (9) ile zorla sonlandırır.
func StopContainerHandler(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Container ID is required"})
		return
	}

	stateFile := filepath.Join(ContainerDir, containerID+".json")
	data, err := os.ReadFile(stateFile)
	var state runtime.ContainerState
	if err == nil {
		json.Unmarshal(data, &state)
	}

	// 1. İşleme temiz kapanma sinyali (SIGTERM) gönder
	_ = exec.Command("sudo", "kill", "-15", containerID).Run()

	// 2. Kapanması için 'grace period' bekle
	time.Sleep(2 * time.Second)

	// 3. Hala yaşıyor mu kontrol et (kill -0)
	if err := exec.Command("sudo", "kill", "-0", containerID).Run(); err == nil {
		_ = exec.Command("sudo", "kill", "-9", containerID).Run()
	}

	// 4. Port Yönlendirmeyi Kaldır (Iptables temizliği) via CLI
	if state.HostPort != "" {
		_ = exec.Command("sudo", "mybox", "stop", containerID).Run()
	}

	_ = os.Remove(stateFile)

	c.JSON(http.StatusOK, gin.H{
		"message": "Container stopped and removed successfully",
		"id":      containerID,
	})
}

// UpdateContainerResourcesHandler, çalışan bir konteynerin Cgroups v2 limitlerini anlık günceller
func UpdateContainerResourcesHandler(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Konteyner ID (PID) gerekli"})
		return
	}

	// Mevcut mimaride ID, PID ile aynı değeri taşır. Integer dönüşümü yapıyoruz.
	pid, err := strconv.Atoi(containerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz Konteyner ID formatı"})
		return
	}

	var req UpdateResourcesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz payload formatı", "details": err.Error()})
		return
	}

	// Cgroups limitlerini sistem düzeyinde güncelle
	if err := cgroup.UpdateCgroups(pid, req.Memory, req.CPUs); err != nil {
		log.Printf("[Error] Failed to update cgroups for PID %d: %v\n", pid, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Kaynak limitleri güncellenemedi",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Konteyner kaynakları başarıyla güncellendi",
		"id":      containerID,
		"applied": req,
	})
}
