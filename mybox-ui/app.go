package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mybox/internal/network"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App yapısı, uygulama durumunu ve HTTP istemcisini tutar
type App struct {
	ctx        context.Context
	httpClient *http.Client
	baseURL    string
}

// POST/PATCH işlemleri için JSON veri modelleri
type RunContainerReq struct {
	Image   string   `json:"image"`
	Command []string `json:"command"`
	Ports   string   `json:"ports"`
	Memory  string   `json:"memory"` // Yeni
	CPUs    string   `json:"cpus"`   // Yeni
}

type UpdateResourceReq struct {
	Memory string `json:"memory"`
	Cpus   string `json:"cpus"`
}

// Yeni Build istek modeli
type BuildImageReq struct {
	Tag     string `json:"tag"`
	Context string `json:"context"`
}

// RunContainer modelini limitleri de içerecek şekilde genişletiyoruz (Gerekirse)

// NewApp yeni bir uygulama instance'ı döndürür
func NewApp() *App {
	return &App{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    "http://localhost:18080/api/v1",
	}
}

// startup, arayüz hazır olduğunda tetiklenir
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// requestHelper, tüm HTTP çağrılarını yöneten merkezi fonksiyondur
func (a *App) requestHelper(method, endpoint string, body interface{}) (interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("payload serileştirilemedi: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(a.ctx, method, a.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("istek oluşturulamadı: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API'ye ulaşılamadı: %w", err)
	}
	defer resp.Body.Close()

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil && err != io.EOF {
		return nil, fmt.Errorf("yanıt okunamadı: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API hatası (%d): %v", resp.StatusCode, result)
	}

	return result, nil
}

// --- API Uç Noktaları ---

func (a *App) GetSystemStats() (interface{}, error) {
	return a.requestHelper(http.MethodGet, "/system/stats", nil)
}

func (a *App) ListContainers() (interface{}, error) {
	return a.requestHelper(http.MethodGet, "/containers", nil)
}

func (a *App) RunContainer(req RunContainerReq) (interface{}, error) {
	return a.requestHelper(http.MethodPost, "/containers/run", req)
}

func (a *App) DeleteContainer(id string) (interface{}, error) {
	return a.requestHelper(http.MethodDelete, fmt.Sprintf("/containers/%s", id), nil)
}

// YENİ: Sistemden klasör seçme diyalogu açar
// YENİ: Sistemden klasör seçme diyalogu açar (Çökme Korumalı)
func (a *App) SelectDirectory() (string, error) {
	// 1. Context kontrolü
	if a.ctx == nil {
		return "", fmt.Errorf("sistem context'i henüz hazır değil")
	}

	// 2. Çökme (panic) yakalayıcı
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Diyalog açılırken panic oluştu: %v\n", r)
		}
	}()

	// 3. Diyaloğu açmayı dene
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "MyBoxFile'ın Bulunduğu Klasörü Seçin",
	})

	if err != nil {
		return "", err
	}
	return dir, nil
}

// YENİ: API'ye build isteği gönderir
func (a *App) BuildImage(req BuildImageReq) (interface{}, error) {
	return a.requestHelper(http.MethodPost, "/images/build", req)
}

// İmaj listesini getirir
func (a *App) ListImages() (interface{}, error) {
	return a.requestHelper(http.MethodGet, "/images", nil)
}

// Belirtilen imajı siler
func (a *App) RemoveImage(name string) (interface{}, error) {
	return a.requestHelper(http.MethodDelete, fmt.Sprintf("/images/%s", name), nil)
}

// UpdateContainerResources, çalışan bir konteynerin limitlerini günceller
func (a *App) UpdateContainerResources(id string, req UpdateResourceReq) (interface{}, error) {
	return a.requestHelper(http.MethodPatch, fmt.Sprintf("/containers/%s/resources", id), req)
}

// OpenURL, sistem tarayıcısında belirtilen URL'yi açar
func (a *App) OpenURL(url string) {
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}

// GetHostIP, makinenin yerel ağdaki IP adresini döner
func (a *App) GetHostIP() string {
	return network.GetLocalIP()
}

// CheckMyBoxFile, belirtilen dizinde MyBoxFile olup olmadığını kontrol eder
func (a *App) CheckMyBoxFile(path string) bool {
	myboxFilePath := filepath.Join(path, "MyBoxFile")
	info, err := os.Stat(myboxFilePath)
	return err == nil && !info.IsDir()
}
