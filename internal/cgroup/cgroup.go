package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const cgroupRoot = "/sys/fs/cgroup"

// parseMemoryBytes converts strings like "512m", "1g", "100k", "1000000" into bytes
func parseMemoryBytes(memStr string) (int64, error) {
	if len(memStr) == 0 {
		return 0, fmt.Errorf("empty memory string")
	}
	unit := strings.ToLower(memStr[len(memStr)-1:])
	var val int64
	var err error

	switch unit {
	case "k":
		v, e := strconv.ParseInt(memStr[:len(memStr)-1], 10, 64)
		val = v * 1024
		err = e
	case "m":
		v, e := strconv.ParseInt(memStr[:len(memStr)-1], 10, 64)
		val = v * 1024 * 1024
		err = e
	case "g":
		v, e := strconv.ParseInt(memStr[:len(memStr)-1], 10, 64)
		val = v * 1024 * 1024 * 1024
		err = e
	default:
		val, err = strconv.ParseInt(memStr, 10, 64)
	}
	return val, err
}

func SetupCgroups(pid int, memory string, cpus string) error {
	// Enable cpu and memory controllers in the root cgroup subtree so children can use them
	subtreePath := filepath.Join(cgroupRoot, "cgroup.subtree_control")
	os.WriteFile(subtreePath, []byte("+cpu +memory"), 0700)

	cgPath := filepath.Join(cgroupRoot, fmt.Sprintf("mybox_%d", pid))

	// Create cgroup directory
	if err := os.MkdirAll(cgPath, 0755); err != nil {
		return fmt.Errorf("cgroup dizini oluşturulamadı: %v", err)
	}

	// Set memory limit if provided
	if memory != "" {
		memBytes, err := parseMemoryBytes(memory)
		if err != nil {
			return fmt.Errorf("geçersiz bellek limiti '%s': %v", memory, err)
		}
		memMaxPath := filepath.Join(cgPath, "memory.max")
		if err := os.WriteFile(memMaxPath, []byte(fmt.Sprintf("%d", memBytes)), 0700); err != nil {
			return fmt.Errorf("memory.max yazılamadı: %v", err)
		}
		fmt.Printf("[*] Cgroup: Bellek limiti %s (%d bayt) olarak ayarlandı.\n", memory, memBytes)
	}

	// Set CPU limit if provided
	if cpus != "" {
		cpuFloat, err := strconv.ParseFloat(cpus, 64)
		if err != nil {
			return fmt.Errorf("geçersiz CPU limiti '%s': %v", cpus, err)
		}
		// period is usually 100000. quota = cpuFloat * 100000
		quota := int64(cpuFloat * 100000)
		period := int64(100000)
		cpuMaxPath := filepath.Join(cgPath, "cpu.max")
		if err := os.WriteFile(cpuMaxPath, []byte(fmt.Sprintf("%d %d", quota, period)), 0700); err != nil {
			return fmt.Errorf("cpu.max yazılamadı: %v", err)
		}
		fmt.Printf("[*] Cgroup: CPU limiti %s (quota: %d, period: %d) olarak ayarlandı.\n", cpus, quota, period)
	}

	// Add process to cgroup
	procsPath := filepath.Join(cgPath, "cgroup.procs")
	if err := os.WriteFile(procsPath, []byte(fmt.Sprintf("%d", pid)), 0700); err != nil {
		return fmt.Errorf("pid değeri cgroup.procs'a yazılamadı: %v", err)
	}

	return nil
}

func CleanupCgroups(pid int) {
	cgPath := filepath.Join(cgroupRoot, fmt.Sprintf("mybox_%d", pid))
	if _, err := os.Stat(cgPath); !os.IsNotExist(err) {
		fmt.Printf("[*] Cgroup: Temizleniyor %s\n", cgPath)
		if err := os.Remove(cgPath); err != nil {
			fmt.Printf("Uyarı: %s dizini silinemedi: %v\n", cgPath, err)
		}
	}
}

// UpdateCgroups çalışan bir konteynerin cgroup limitlerini dinamik olarak günceller.
func UpdateCgroups(pid int, memory string, cpus string) error {
	cgPath := filepath.Join(cgroupRoot, fmt.Sprintf("mybox_%d", pid))

	// Konteynerin cgroup dizini yoksa işlem yapılamaz
	if _, err := os.Stat(cgPath); os.IsNotExist(err) {
		return fmt.Errorf("cgroup dizini bulunamadı (konteyner durmuş olabilir): %s", cgPath)
	}

	// Bellek limitini güncelle
	if memory != "" {
		memBytes, err := parseMemoryBytes(memory)
		if err != nil {
			return fmt.Errorf("geçersiz bellek limiti '%s': %v", memory, err)
		}
		memMaxPath := filepath.Join(cgPath, "memory.max")
		if err := os.WriteFile(memMaxPath, []byte(fmt.Sprintf("%d", memBytes)), 0700); err != nil {
			return fmt.Errorf("memory.max dosyasına yazılamadı: %v", err)
		}
	}

	// CPU limitini güncelle
	if cpus != "" {
		cpuFloat, err := strconv.ParseFloat(cpus, 64)
		if err != nil {
			return fmt.Errorf("geçersiz CPU limiti '%s': %v", cpus, err)
		}
		quota := int64(cpuFloat * 100000)  //period 100000 hardcoded, bu ayarlanabilir yapılabilir
		period := int64(100000)
		cpuMaxPath := filepath.Join(cgPath, "cpu.max")
		if err := os.WriteFile(cpuMaxPath, []byte(fmt.Sprintf("%d %d", quota, period)), 0700); err != nil {
			return fmt.Errorf("cpu.max dosyasına yazılamadı: %v", err)
		}
	}

	return nil
}
