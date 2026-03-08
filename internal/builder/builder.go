package builder

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func pullAndExtractImage(imageName, targetDir string) error {
	fmt.Printf("[*] %s imajı Docker Hub'dan çekiliyor (Skopeo)...\n", imageName)

	tempDir := "./temp_oci"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Bu komut manifest ve layer'ları parçalanmış halde getirir
	cmd := exec.Command("skopeo", "copy", "docker://"+imageName, "dir:"+tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("imaj çekilemedi: %v", err)
	}

	// İmajın içindeki tüm .tar.gz (layer) dosyalarını bul ve sırayla targetDir'e aç
	files, _ := os.ReadDir(tempDir)
	for _, file := range files {

		if !file.IsDir() && len(file.Name()) > 10 {
			fmt.Printf("   > Katman açılıyor: %s\n", file.Name()[:12])
			layerPath := filepath.Join(tempDir, file.Name())

			exec.Command("tar", "-xf", layerPath, "-C", targetDir).Run()
		}
	}
	return nil
}

func downloadImage(url, dest string) error {
	fmt.Printf("Imaj merkezi depoda bulunamadı, indiriliyor: %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("indirme hatası: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTar(tarFile, targetDir string) error {
	cmd := exec.Command("tar", "-xf", tarFile, "-C", targetDir)
	return cmd.Run()
}

func saveAsImage(sourceDir, imagePath string) error {
	fmt.Printf("Imaj paketleniyor: %s\n", imagePath)
	// --no-same-owner: çıkarırken root sahipliği zorunlu kılmaz
	cmd := exec.Command("sudo", "tar", "-cf", imagePath,
		"--no-same-owner",
		"-C", sourceDir, ".")
	return cmd.Run()
}

func BuildImage(buildContext, imagePath string) {
	myBoxFilePath := filepath.Join(buildContext, "MyBoxFile")

	fmt.Printf("[*] Build İşlemi Başladı\n")
	fmt.Printf("[*] Context: %s\n", buildContext)
	fmt.Printf("[*] MyBoxFile: %s\n", myBoxFilePath)

	// MyBoxFile kontrolü
	if _, err := os.Stat(myBoxFilePath); os.IsNotExist(err) {
		fmt.Printf("[!] HATA: %s bulunamadı. Lütfen seçili klasörde bir 'MyBoxFile' olduğundan emin olun.\n", myBoxFilePath)
		return
	}

	workDir, err := os.MkdirTemp("", "mybox_builder_rootfs_*")
	if err != nil {
		fmt.Printf("Temp dizin oluşturulamadı: %v\n", err)
		return
	}
	defer os.RemoveAll(workDir)

	// MERKEZİ DEPO AYARI
	homedir, _ := os.UserHomeDir()
	imageStore := filepath.Join(homedir, ".mybox", "images")
	os.MkdirAll(imageStore, 0755)

	file, err := os.Open(myBoxFilePath)
	if err != nil {
		fmt.Printf("Hata: %s bulunamadı\n", myBoxFilePath)
		return
	}
	defer file.Close()

	fmt.Println("---- MyBox Build Başladı ----")

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
			os.RemoveAll(workDir)
			os.MkdirAll(workDir, 0755)

			// imaj dosyaları docker hub'tan çekiliyor sabit bir sözlük yok
			err := pullAndExtractImage(argument, workDir)
			if err != nil {
				fmt.Printf("Hata: %v\n", err)
				return
			}

		case "WORKDIR":
			fmt.Printf("[*] WORKDIR ayarlanıyor: %s\n", argument)
			currentWorkingDir = argument
			absWorkDir := filepath.Join(workDir, currentWorkingDir)
			os.MkdirAll(absWorkDir, 0755)
			// Runtime'ın doğru dizini bilmesi için kaydet
			os.WriteFile(filepath.Join(workDir, ".mybox_workdir"), []byte(argument), 0644)

		case "RUN":
			fmt.Printf("[2] Komut çalıştırılıyor: %s\n", argument)
			// İnternet erişimi için DNS ayarlarını kopyala
			dnsPath := filepath.Join(workDir, "etc/resolv.conf")
			os.MkdirAll(filepath.Dir(dnsPath), 0755)
			exec.Command("cp", "/etc/resolv.conf", dnsPath).Run()

			cmd := exec.Command("sudo", "chroot", workDir, "/bin/sh", "-c", "cd "+currentWorkingDir+" && "+argument)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("[!] HATA: RUN komutu başarısız oldu: %v\n", err)
				fmt.Printf("[!] Komut: %s\n", argument)
				fmt.Println("[!] Build iptal ediliyor.")
				return
			}

		case "COPY":
			fmt.Printf("[3] Dosya kopyalanıyor: %s\n", argument)
			copyParts := strings.Fields(argument)
			if len(copyParts) < 2 {
				continue
			}

			srcPath := filepath.Join(buildContext, copyParts[0])
			dstPath := copyParts[1]

			// Eğer hedef yol '/' ile başlıyorsa mutlak yoldur (root'a göredir)
			// değilse WORKDIR'e göredir.
			var finalDstInRootfs string
			if filepath.IsAbs(dstPath) {
				finalDstInRootfs = filepath.Join(workDir, dstPath)
			} else {
				finalDstInRootfs = filepath.Join(workDir, currentWorkingDir, dstPath)
			}

			// Hedefin üst dizinini oluştur
			os.MkdirAll(filepath.Dir(finalDstInRootfs), 0755)

			var copyErr error
			if info, err := os.Stat(srcPath); err == nil && info.IsDir() {
				// Dizin kopyalama: mybox_rootfs ve .tar dosyalarını hariç tut
				os.MkdirAll(finalDstInRootfs, 0755)
				// rsync varsa kullan, yoksa find+cp kullan
				rsyncCmd := exec.Command("rsync", "-a",
					"--exclude=mybox_rootfs",
					"--exclude=*.tar",
					"--exclude=.git",
					srcPath+"/", finalDstInRootfs+"/")
				if copyErr = rsyncCmd.Run(); copyErr != nil {
					// rsync yoksa cp ile dene (yine de mybox_rootfs hariç)
					fmt.Printf("   [*] rsync bulunamadı, cp kullanılıyor\n")
					entries, _ := os.ReadDir(srcPath)
					for _, entry := range entries {
						if entry.Name() == "mybox_rootfs" || entry.Name() == "temp_oci" {
							continue
						}
						if strings.HasSuffix(entry.Name(), ".tar") {
							continue
						}
						src := filepath.Join(srcPath, entry.Name())
						cpCmd := exec.Command("cp", "-r", src, finalDstInRootfs)
						if err := cpCmd.Run(); err != nil {
							fmt.Printf("   Uyarı: %s kopyalanamadı: %v\n", entry.Name(), err)
						}
					}
					copyErr = nil // tek tek kopyaladık
				}
			} else {
				// Tek dosya kopyalama
				copyErr = exec.Command("cp", "-r", srcPath, finalDstInRootfs).Run()
			}

			if copyErr != nil {
				fmt.Printf("Kopyalama hatası: %v\n", copyErr)
			} else {
				// Kopyalanan dosyaların erişilebilir/çalıştırılabilir olmasını garantile
				exec.Command("chmod", "-R", "755", finalDstInRootfs).Run()
			}

		case "CMD":
			fmt.Printf("[*] Başlangıç komutu belirlendi: %s\n", argument)

			os.WriteFile(filepath.Join(workDir, ".mybox_entrypoint"), []byte(argument), 0644)

		case "EXPOSE":
			fmt.Printf("[*] Port bilgilendirmesi: %s\n", argument)
			// Runtime'ın portu bilmesi için kaydet
			os.WriteFile(filepath.Join(workDir, ".mybox_expose"), []byte(argument), 0644)
		}
	}

	fmt.Println("\n---- MyBox Build İşlemi Tamamlandı!! ----")

	if err := saveAsImage(workDir, imagePath); err != nil {
		fmt.Printf("Hata: %v\n", err)
	} else {
		fmt.Printf("Imaj başarıyla oluşturuldu: %s\n", imagePath)
	}
}
