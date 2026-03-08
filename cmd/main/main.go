package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	// Modül adın 'mybox' olduğu için yollar bu şekilde olmalı
	"mybox/internal/api/router"
	"mybox/internal/builder"
	"mybox/internal/runtime"
)

const (
	ImageDir     = "/var/lib/mybox/images"
	ContainerDir = "/var/lib/mybox/containers"
	Version      = "1.0.7"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "install":
		handleInstall()
	case "images":
		handleImages()
	case "build":
		requireRoot()
		handleBuild(args)
	case "run":
		handleRun(args)
	case "ps":
		handlePS()
	case "stop":
		handleStop(args)
	case "rmi":
		requireRoot()
		handleRMI(args)
	case "child":
		runtime.Child()
	case "-version", "--version", "version":
		fmt.Printf("MyBox Version: %s\n", Version)
	case "serve":
		requireRoot()
		router.StartAPI()
	default:
		printHelp()
	}
}

// requireRoot: eğer kullanıcı root değilse komutu sudo ile yeniden çalıştırır.
// Bu sayede 'mybox run' ve 'mybox stop' sudo gerektirmeksizin kullanılabilir.
func requireRoot() {
	if syscall.Geteuid() == 0 {
		return // Zaten root
	}

	absPath := os.Args[0]
	if !filepath.IsAbs(absPath) {
		if lookPath, err := exec.LookPath(absPath); err == nil {
			absPath = lookPath
		}
	}

	fmt.Println("[*] Root yetkisi gerekiyor, sudo ile yeniden başlatılıyor...")
	cmd := exec.Command("sudo", append([]string{absPath}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("[-] sudo başarısız: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// --- OTOMATİK KURULUM ---
func handleInstall() {
	fmt.Println("[*] MyBox dizinleri kuruluyor...")

	// Gerekli dizinleri oluştur
	if err := exec.Command("sudo", "mkdir", "-p", ImageDir, ContainerDir).Run(); err != nil {
		fmt.Printf("[-] Dizin oluşturulamadı: %v\n", err)
		return
	}
	// Herkesin okuyup yazabilmesi için izin ver
	exec.Command("sudo", "chmod", "-R", "777", "/var/lib/mybox").Run()

	fmt.Println("[+] Dizinler hazır:")
	fmt.Printf("    İmajlar   : %s\n", ImageDir)
	fmt.Printf("    Konteynerler: %s\n", ContainerDir)
	fmt.Println("[+] MyBox kullanıma hazır!")
}

// --- KOMUTLAR ---
func handleBuild(args []string) {
	tag := "latest"
	context := "."
	for i, arg := range args {
		if arg == "-t" && i+1 < len(args) {
			// -t flag ile: mybox build -t isim .
			tag = args[i+1]
			if i+2 < len(args) && !strings.HasPrefix(args[i+2], "-") {
				context = args[i+2]
			}
		} else if !strings.HasPrefix(arg, "-") && i > 0 && args[i-1] != "-t" {
			// Pozisyonel arg: mybox build isim
			tag = arg
		} else if !strings.HasPrefix(arg, "-") && i == 0 {
			// İlk arg doğrudan isim: mybox build isim
			tag = arg
		}
	}
	dest := filepath.Join(ImageDir, tag+".tar")
	fmt.Printf("[*] İmaj oluşturuluyor: %s (tag: %s)\n", dest, tag)

	builder.BuildImage(context, dest)
}

func handleRun(args []string) {
	requireRoot()
	if len(args) == 0 {
		fmt.Println("Hata: İmaj adı gerekli.")
		return
	}

	imageName := args[len(args)-1]
	if !strings.HasSuffix(imageName, ".tar") {
		imageName += ".tar"
	}
	imagePath := filepath.Join(ImageDir, imageName)

	// start.sh yetkilerini otomatik düzelt (Permission Denied Çözümü)
	currentDir, _ := os.Getwd()
	scriptPath := filepath.Join(currentDir, "start.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		os.Chmod(scriptPath, 0755)
	}

	fmt.Printf("[*] Konteyner başlatılıyor: %s\n", imageName)

	// HATA ÇÖZÜLDÜ: imagePath artık Parent fonksiyonunda kullanılıyor
	runtime.Parent(args, imagePath)
}

func handlePS() {
	runtime.ListContainers()
}

func handleImages() {
	fmt.Printf("%-25s %-15s %-10s %s\n", "REPOSITORY", "TAG", "SIZE", "CREATED")
	fmt.Println(strings.Repeat("-", 75))
	files, _ := os.ReadDir(ImageDir)
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".tar" {
			continue
		}
		info, _ := f.Info()
		name := strings.TrimSuffix(f.Name(), ".tar")
		size := info.Size() / (1024 * 1024)
		if size == 0 {
			size = 1
		}
		created := info.ModTime().Format("2006-01-02 15:04")
		fmt.Printf("%-25s %-15s %-10s %s\n", name, name, fmt.Sprintf("%d MB", size), created)
	}
}

func handleRMI(args []string) {
	if len(args) == 0 {
		fmt.Println("Hata: İmaj adı gerekli. Kullanım: mybox rmi <isim>")
		return
	}
	for _, name := range args {
		// .tar uzantısı yoksa ekle
		if !strings.HasSuffix(name, ".tar") {
			name = name + ".tar"
		}
		imagePath := filepath.Join(ImageDir, name)
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			fmt.Printf("[-] Hata: '%s' imajı bulunamadı.\n", strings.TrimSuffix(name, ".tar"))
			continue
		}
		if err := os.Remove(imagePath); err != nil {
			fmt.Printf("[-] '%s' silinemedi: %v\n", strings.TrimSuffix(name, ".tar"), err)
		} else {
			fmt.Printf("[+] İmaj silindi: %s\n", strings.TrimSuffix(name, ".tar"))
		}
	}
}

func handleStop(args []string) {
	requireRoot()
	if len(args) == 0 {
		return
	}
	runtime.RemoveContainer(args[0])
}

func printHelp() {
	fmt.Println("\nMyBox CLI Kullanımı:")
	fmt.Println("  install          : Sistemi kurar")
	fmt.Println("  build [isim]     : İmaj oluşturur (veya -t isim)")
	fmt.Println("  run [seçenek]    : Konteyner başlatır")
	fmt.Println("  ps               : Çalışanları listeler")
	fmt.Println("  stop [PID]       : Konteyneri durdurur")
	fmt.Println("  images           : İmajları listeler")
	fmt.Println("  serve            : API sunucusunu başlatır")
	fmt.Println("  rmi [isim...]    : İmaj(lar)ı siler")
	fmt.Println("  --version        : Sürümü gösterir")
}
