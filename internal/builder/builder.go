package builder

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func pullAndExtractImage(imageName, targetDir string) error {
	cacheRoot := "/var/lib/mybox/cache"
	imageSlug := strings.ReplaceAll(imageName, ":", "_")
	imageCacheDir := filepath.Join(cacheRoot, imageSlug)

	// Eğer cache yoksa çek (Sadeleşmiş sürüm)
	if _, err := os.Stat(imageCacheDir); err != nil {
		fmt.Printf("[*] %s imajı Docker Hub'dan çekiliyor (Skopeo)...\n", imageName)
		os.MkdirAll(imageCacheDir, 0755)
		cmd := exec.Command("skopeo", "copy", "docker://"+imageName, "dir:"+imageCacheDir)
		if err := cmd.Run(); err != nil {
			os.RemoveAll(imageCacheDir)
			return err
		}
	} else {
		fmt.Printf("[*] %s imajı yerel önbellekten (cache) kullanılıyor...\n", imageName)
	}

	// Katmanları aç
	files, _ := os.ReadDir(imageCacheDir)
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 10 {
			if strings.HasSuffix(file.Name(), ".json") || file.Name() == "version" {
				continue
			}
			layerPath := filepath.Join(imageCacheDir, file.Name())
			exec.Command("tar", "-xf", layerPath, "-C", targetDir).Run()
		}
	}
	return nil
}

func BuildImage(buildContext, imagePath string) {
	absContext, _ := filepath.Abs(buildContext)
	myBoxFilePath := filepath.Join(absContext, "MyBoxFile")
	
	// SALTED BUILD: Proje yoluna özel benzersiz build dizini
	// Bu sayede farklı projeler birbirinin build dosyalarını kirletemez.
	h := sha256.New()
	h.Write([]byte(absContext))
	buildHash := hex.EncodeToString(h.Sum(nil))[:8]
	workDir := filepath.Join("/tmp", "mybox_build_"+buildHash)
	
	file, err := os.Open(myBoxFilePath)
	if err != nil {
		fmt.Printf("Hata: %s bulunamadı\n", myBoxFilePath)
		return
	}
	defer file.Close()

	fmt.Println("---- MyBox Build Başladı ----")
	safeCleanup(workDir)
	os.MkdirAll(workDir, 0755)
	defer safeCleanup(workDir)

	currentWorkingDir := "/"
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		instruction := parts[0]
		argument := parts[1]

		switch instruction {
		case "FROM":
			fmt.Printf("[1] %s imajı hazırlanıyor...\n", argument)
			pullAndExtractImage(argument, workDir)

		case "WORKDIR":
			fmt.Printf("[*] WORKDIR ayarlanıyor: %s\n", argument)
			currentWorkingDir = argument
			os.MkdirAll(filepath.Join(workDir, currentWorkingDir), 0755)
			os.WriteFile(filepath.Join(workDir, ".mybox_workdir"), []byte(argument), 0644)

		case "COPY":
			fmt.Printf("[3] Dosya kopyalanıyor: %s\n", argument)
			copyParts := strings.Fields(argument)
			if len(copyParts) < 2 {
				continue
			}

			src := filepath.Join(absContext, copyParts[0])
			dstName := copyParts[1]
			dst := filepath.Join(workDir, currentWorkingDir, dstName)
			if filepath.IsAbs(dstName) {
				dst = filepath.Join(workDir, dstName)
			}

			os.MkdirAll(filepath.Dir(dst), 0755)

			// TEMIZ KOPYALAMA: Build kirliliğini ve gereksiz klasörleri engelle
			if copyParts[0] == "." {
				// rsync varsa rsync kullan (daha güvenli), yoksa cp
				if err := exec.Command("rsync", "-a", "--exclude", "mybox_rootfs*", src+"/", dst+"/").Run(); err != nil {
					exec.Command("sh", "-c", "cp -rp "+src+"/* "+dst+"/ 2>/dev/null || true").Run()
				}
			} else {
				exec.Command("cp", "-rp", src, dst).Run()
			}

		case "RUN":
			fmt.Printf("[2] Komut çalıştırılıyor: %s\n", argument)
			// DNS ve Proc mount (Basit sürüm)
			dns := filepath.Join(workDir, "etc/resolv.conf")
			os.MkdirAll(filepath.Dir(dns), 0755)
			exec.Command("cp", "/etc/resolv.conf", dns).Run()
			
			proc := filepath.Join(workDir, "proc")
			os.MkdirAll(proc, 0755)
			exec.Command("mount", "-t", "proc", "proc", proc).Run()
			
			cmd := exec.Command("chroot", workDir, "/bin/sh", "-c", "cd "+currentWorkingDir+" && "+argument)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
			
			exec.Command("umount", "-l", proc).Run()

		case "CMD":
			fmt.Printf("[*] Başlangıç komutu belirlendi: %s\n", argument)
			os.WriteFile(filepath.Join(workDir, ".mybox_entrypoint"), []byte(argument), 0644)
		}
	}

	fmt.Println("\n---- MyBox Build İşlemi Tamamlandı!! ----")
	cmd := exec.Command("tar", "-cf", imagePath, "-C", workDir, ".")
	cmd.Run()
	fmt.Printf("Imaj başarıyla oluşturuldu: %s\n", imagePath)
}

func safeCleanup(path string) {
	if path == "" || path == "/" { return }
	data, _ := os.ReadFile("/proc/mounts")
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.Fields(lines[i])
		if len(fields) > 1 && strings.HasPrefix(fields[1], path) {
			exec.Command("umount", "-l", fields[1]).Run()
		}
	}
	time.Sleep(100 * time.Millisecond)
	os.RemoveAll(path)
}
