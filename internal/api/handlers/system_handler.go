// internal/api/handlers/system_handler.go

package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// main.go'daki sabitlerle uyumlu varsayılan temel dizin
const BaseDir = "/var/lib/mybox"

// GetSystemStatsHandler, gopsutil kullanarak gerçek zamanlı donanım ve disk verisi döner.
func GetSystemStatsHandler(c *gin.Context) {
	// 1. Bellek istatistikleri
	v, errMem := mem.VirtualMemory()

	// 2. CPU istatistikleri (kısa aralıklı kullanım yüzdesi)
	cPercent, errCPU := cpu.Percent(0, false)
	var cpuUsage float64
	if errCPU == nil && len(cPercent) > 0 {
		cpuUsage = cPercent[0]
	}

	// 3. Konteyner imajlarının/durumlarının tutulduğu diskin doluluk durumu
	targetDir := os.Getenv("MYBOX_BASE_DIR")
	if targetDir == "" {
		targetDir = BaseDir
	}
	d, errDisk := disk.Usage(targetDir)

	// Donanım okuma hatalarını yoksayıp 0 dönebiliriz (API çökmemesi için)
	if errMem != nil {
		v = &mem.VirtualMemoryStat{}
	}
	if errDisk != nil {
		d = &disk.UsageStat{}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		//"version": os.Getenv("MYBOX_VERSION"), // Sabit 1.0.5 yerine ENV tavsiyesi
		"version": "1.0.5",
		"storage": targetDir,
		"metrics": gin.H{
			"cpu_percent":   cpuUsage,
			"mem_total_mb":  v.Total / (1024 * 1024),
			"mem_used_mb":   v.Used / (1024 * 1024),
			"mem_percent":   v.UsedPercent,
			"disk_total_mb": d.Total / (1024 * 1024),
			"disk_used_mb":  d.Used / (1024 * 1024),
			"disk_percent":  d.UsedPercent,
		},
	})
}
