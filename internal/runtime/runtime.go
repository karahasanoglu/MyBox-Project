package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mybox/internal/cgroup"
	"mybox/internal/network"
)

const stateDir = "/var/lib/mybox/containers"

type ContainerState struct {
	PID       int       `json:"pid"`
	ID        string    `json:"id"`
	Image     string    `json:"image"`
	IP        string    `json:"ip"`
	HostPort  string    `json:"host_port"`
	ContPort  string    `json:"cont_port"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
	Command   string    `json:"command"`
}

func saveState(state ContainerState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	filename := filepath.Join(stateDir, fmt.Sprintf("%d.json", state.PID))
	return ioutil.WriteFile(filename, data, 0644)
}

func removeState(pid int) {
	filename := filepath.Join(stateDir, fmt.Sprintf("%d.json", pid))
	os.Remove(filename)
}

// IsHostPortInUse, verilen host portun çalışan herhangi bir konteyner tarafından
// kullanılıp kullanılmadığını kontrol eder.
// Sadece gerçekten canlı (sinyale yanıt veren) işlemleri dikkate alır.
func IsHostPortInUse(hostPort string) (bool, string) {
	files, err := ioutil.ReadDir(stateDir)
	if err != nil {
		return false, ""
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		data, err := ioutil.ReadFile(filepath.Join(stateDir, file.Name()))
		if err != nil {
			continue
		}
		var s ContainerState
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if s.HostPort != hostPort {
			continue
		}
		// Sürecin hâlâ canlı olup olmadığını kontrol et
		process, findErr := os.FindProcess(s.PID)
		if findErr != nil {
			continue
		}
		sigErr := process.Signal(syscall.Signal(0))
		isAlive := sigErr == nil || sigErr == syscall.EPERM
		if isAlive {
			return true, s.ID
		}
	}
	return false, ""
}

func Parent(args []string, imagePath string) {
	// 1. Host side network setup
	bridgeName := "mybox0"
	bridgeIP := "10.0.0.1/24"

	must(network.SetupBridge(bridgeName, bridgeIP))
	must(network.SetupNAT(bridgeName))

	// --- Port çakışması kontrolü (cmd.Start'tan ÖNCE) ---
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			parts := strings.SplitN(args[i+1], ":", 2)
			if len(parts) == 2 {
				requestedHostPort := parts[0]
				if inUse, occupiedBy := IsHostPortInUse(requestedHostPort); inUse {
					fmt.Printf("\n[!] HATA: Host port %s zaten kullanımda (konteyner ID: %s)\n", requestedHostPort, occupiedBy)
					fmt.Printf("    İpucu: Önce o konteyneri durdurun → mybox stop %s\n", occupiedBy)
					fmt.Printf("    Ya da farklı bir host portu seçin.\n\n")
					return
				}
			}
		}
	}

	// Veth adı ve container IP'sini cmd.Start()'tan ÖNCE üret
	// Sebebi: CLONE_NEWPID ile başlatılan child içinde os.Getpid()=1 döner (host PID değil).
	// Bu değerleri env variable ile child'a iletiyoruz.
	contID := fmt.Sprintf("%04x", rand.Intn(0x10000))
	hostVeth := "vh-" + contID // örn: vh-3f2a
	contVeth := "vc-" + contID // örn: vc-3f2a

	// Benzersiz container IP: 10.0.X.Y
	ipX := rand.Intn(253) + 1 // 1-253
	ipY := rand.Intn(253) + 2 // 2-254
	containerIP := fmt.Sprintf("10.0.%d.%d", ipX, ipY)
	containerIPCIDR := containerIP + "/24"

	fmt.Printf("[*] Konteyner Ağ: veth=%s ip=%s\n", hostVeth, containerIP)

	// 2. Clone with NEWNET — Env'i Start'tan ÖNCE set et
	cmd := exec.Command("/proc/self/exe", append([]string{"child", imagePath}, args...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Child bu env'leri okuyarak doğru veth ve IP'yi bulur
	cmd.Env = append(os.Environ(),
		"MYBOX_CONT_VETH="+contVeth,
		"MYBOX_CONT_IP="+containerIPCIDR,
		"MYBOX_CONT_ID="+contID,
	)

	if err := cmd.Start(); err != nil {
		fmt.Printf("Hata: %v\n", err)
		os.Exit(1)
	}

	pid := cmd.Process.Pid
	fmt.Printf("[*] Child PID: %d\n", pid)

	memoryLimit := ""
	cpuLimit := ""
	for i, arg := range args {
		if arg == "--memory" && i+1 < len(args) {
			memoryLimit = args[i+1]
		}
		if (arg == "--cpus" || arg == "--cpu") && i+1 < len(args) {
			cpuLimit = args[i+1]
		}
	}

	must(cgroup.SetupCgroups(pid, memoryLimit, cpuLimit))
	defer cgroup.CleanupCgroups(pid)
	defer cleanupRootfs(contID)

	if err := network.CreateVethPair(hostVeth, contVeth); err != nil {
		fmt.Printf("Uyarı: Veth çifti oluşturulamadı: %v\n", err)
		cmd.Wait()
		return
	}
	if err := network.AttachVethToBridge(hostVeth, bridgeName); err != nil {
		fmt.Printf("Uyarı: Veth köprüye bağlanamadı: %v\n", err)
		cmd.Wait()
		return
	}
	// Child'in hala yaşayıp yaşamadığını kontrol et
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		fmt.Println("[-] Child süreç erken çıktı, iptal ediliyor.")
		return
	}
	if err := network.MoveVethToNamespace(contVeth, pid); err != nil {
		fmt.Printf("Uyarı: Veth namespace'e taşınamadı: %v\n", err)
		cmd.Wait()
		return
	}

	// State bilgileri için değişkenler
	// Host'a bu container IP'ye nasıl gidileceğini söyle:
	// ip route add 10.0.82.67/32 dev mybox0
	// Bu olmadan DNAT sonrası paketi bridge'e iletemiyor!
	if err := exec.Command("ip", "route", "add", containerIP+"/32", "dev", bridgeName).Run(); err != nil {
		fmt.Printf("Uyarı: Container route eklenemedi: %v\n", err)
	}
	defer exec.Command("ip", "route", "del", containerIP+"/32", "dev", bridgeName).Run()

	var hostPort, contPort string

	imageName := filepath.Base(imagePath)

	// İmajın EXPOSE portunu oku (builder tarafından kaydedilmiş)
	// Bu bilgiyi port uyumsuzluklarını tespit etmek için kullanıyoruz
	imageStorePath := strings.TrimSuffix(imagePath, filepath.Ext(imagePath))
	_ = imageStorePath // sadece expose kontrolü için kullanılır

	// 4. Port Yönlendirme Yapılandırması (-p 8080:80)
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			ports := strings.Split(args[i+1], ":")
			if len(ports) == 2 {
				hostPort = ports[0]
				contPort = ports[1]
				must(network.SetupPortForwarding(hostPort, containerIP, contPort))

				localIP := network.GetLocalIP()
				fmt.Printf("\n[!] Konteynere şuradan erişebilirsiniz: http://%s:%s\n\n", localIP, hostPort)
			}
		}
	}

	// State Kaydet
	state := ContainerState{
		PID:       pid,
		ID:        fmt.Sprintf("%d", pid),
		Image:     imageName,
		IP:        containerIP,
		HostPort:  hostPort,
		ContPort:  contPort,
		StartTime: time.Now(),
		Status:    "Running",
		Command:   strings.Join(args, " "),
	}
	if err := saveState(state); err != nil {
		fmt.Printf("Uyarı: State kaydedilemedi: %v\n", err)
	}

	// Container bitince state'i temizle
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Hata: %v\n", err)
	}
	removeState(pid)
}

func cleanupRootfs(contID string) {
	tempRootfs := fmt.Sprintf("/tmp/mybox_rootfs_%s", contID)
	if _, err := os.Stat(tempRootfs); !os.IsNotExist(err) {
		fmt.Printf("[*] Rootfs temizleniyor: %s\n", tempRootfs)
		// Unmount /proc if it's still mounted
		procPath := filepath.Join(tempRootfs, "proc")
		syscall.Unmount(procPath, 0)

		// Attempt to unmount the root itself just in case there are binds
		syscall.Unmount(tempRootfs, 0)

		if err := os.RemoveAll(tempRootfs); err != nil {
			fmt.Printf("Uyarı: %s rootfs dizini silinemedi: %v\n", tempRootfs, err)
		}
	}
}

func ListContainers() {
	// Klasörün var olduğundan emin ol
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		fmt.Println("Konteyner dizini oluşturulamadı:", err)
		return
	}

	files, err := ioutil.ReadDir(stateDir)
	if err != nil {
		fmt.Println("Konteyner listesi alınamadı:", err)
		return
	}

	fmt.Printf("%-10s %-22s %-15s %-18s %-10s %s\n", "PID", "IMAGE", "IP", "PORTS", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 100))

	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		data, err := ioutil.ReadFile(filepath.Join(stateDir, file.Name()))
		if err != nil {
			continue
		}
		var s ContainerState
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}

		// PID canlılık kontrolü:
		// kill(pid, 0) çıktıları:
		//   nil   -> süreç var ve iznimiz var
		//   EPERM -> süreç var ama iznimiz yok (root process, biz normal user)
		//   ESRCH -> süreç yok
		process, findErr := os.FindProcess(s.PID)
		var isAlive bool
		if findErr == nil {
			sigErr := process.Signal(syscall.Signal(0))
			// nil veya EPERM => process yaşıyor
			isAlive = sigErr == nil || sigErr == syscall.EPERM
		}

		if isAlive {
			ports := "-"
			if s.HostPort != "" && s.ContPort != "" {
				ports = fmt.Sprintf("%s->%s", s.HostPort, s.ContPort)
			}
			created := s.StartTime.Format("2006-01-02 15:04")
			fmt.Printf("%-10d %-22s %-15s %-18s %-10s %s\n", s.PID, s.Image, s.IP, ports, s.Status, created)
			found = true
		} else {
			// Süreç ölmüşse state dosyasını sil
			removeState(s.PID)
		}
	}

	if !found {
		fmt.Println("Çalışan konteyner yok.")
	}
}
func RemoveContainer(pidStr string) {
	fmt.Printf("[*] Konteyner durduruluyor: PID=%s\n", pidStr)

	// 1. Önce normal kill dene
	killErr := exec.Command("kill", "-9", pidStr).Run()
	if killErr != nil {
		// Normal kill başarısız (muhtemelen root process) → sudo ile dene
		fmt.Printf("   [*] Sudo ile durduruluyor...\n")
		killErr = exec.Command("sudo", "kill", "-9", pidStr).Run()
	}

	// 2. State dosyasını her durumda sil
	pid, parseErr := strconv.Atoi(pidStr)
	if parseErr == nil {
		removeState(pid)
	}

	// 3. Sonuç raporu
	if killErr != nil {
		fmt.Printf("[-] Konteyner durdurulamadı (PID=%s): %v\n", pidStr, killErr)
		fmt.Println("    İpcu: 'sudo mybox stop <PID>' deneyin.")
	} else {
		fmt.Println("[+] Konteyner başarıyla durduruldu.")
	}
}

func InspectContainer(pidStr string) {
	data, err := ioutil.ReadFile(filepath.Join(stateDir, pidStr+".json"))
	if err != nil {
		fmt.Printf("Hata: %s PID'li konteyner bulunamadı.\n", pidStr)
		return
	}
	fmt.Println(string(data))
}
func Child() {
	fmt.Println("\n---- [MyBox] Konteyner Başlatılıyor ----")
	imagePath := os.Args[2]

	contID := os.Getenv("MYBOX_CONT_ID")
	if contID == "" {
		contID = fmt.Sprintf("fallback_pid_%d", os.Getpid())
	}
	tempRootfs := "/tmp/mybox_rootfs_" + contID
	// Önceki başarısız çalışmadan kalan artıkları temizle ve sıfırla
	os.RemoveAll(tempRootfs)
	os.MkdirAll(tempRootfs, 0755)

	// Note: Cleanup of this directory and its mounts is now handled by the parent process
	// because defering unmount/remove here fails after chrooting.

	fmt.Printf("[*] İmaj dosyası açılıyor: %s\n", imagePath)
	// --no-same-owner: root sahipli dosyalar root izni olmadan da çıkarılabilir
	// --overwrite: var/lock gibi sembolik link çakışmalarını aş
	tarCmd := exec.Command("tar", "-xf", imagePath, "--no-same-owner", "--overwrite", "-C", tempRootfs)
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		fmt.Printf("Hata: İmaj açılamadı: %v\n", err)
		return
	}

	// 4. Container side network setup
	// Veth adı ve IP'yi parent'tan env variable ile al
	// (PID namespace içinde os.Getpid()=1 döner, host PID'si değil)
	contVeth := os.Getenv("MYBOX_CONT_VETH")
	containerCIDR := os.Getenv("MYBOX_CONT_IP")
	gateway := "10.0.0.1"

	// Fallback: env set edilmediyse varsayılan değerleri kullan
	if contVeth == "" {
		contVeth = "vc-0000"
	}
	if containerCIDR == "" {
		containerCIDR = "10.0.1.2/24"
	}

	fmt.Printf("[*] Ağ: veth=%s ip=%s\n", contVeth, containerCIDR)

	fmt.Println("[*] Ağ yapılandırması bekleniyor...")
	time.Sleep(1 * time.Second)
	must(network.SetupContainerNetwork(contVeth, containerCIDR, gateway))

	must(syscall.Sethostname([]byte("mybox-container")))
	// 5. Volume Mounting (-v /host:/cont)
	for i, arg := range os.Args {
		if arg == "-v" && i+1 < len(os.Args) {
			volumes := strings.Split(os.Args[i+1], ":")
			if len(volumes) == 2 {
				hostPath := volumes[0]
				contPath := volumes[1]
				// Konteyner içindeki hedef dizini oluştur (chroot öncesi tempRootfs içinde)
				targetInRootfs := filepath.Join(tempRootfs, contPath)
				os.MkdirAll(targetInRootfs, 0755)

				fmt.Printf("[*] Volume bağlanıyor: %s -> %s\n", hostPath, contPath)
				// mount --bind <hostPath> <targetInRootfs>
				if err := exec.Command("mount", "--bind", hostPath, targetInRootfs).Run(); err != nil {
					fmt.Printf("Volume bağlama hatası: %v\n", err)
				} else {
					// Konteyner kapandığında otomatik ayırmak için
					defer syscall.Unmount(targetInRootfs, 0)
				}
			}
		}
	}

	must(syscall.Chroot(tempRootfs))
	must(os.Chdir("/"))

	os.MkdirAll("/proc", 0755)
	must(syscall.Mount("proc", "/proc", "proc", 0, ""))

	// WORKDIR bilgisini oku (builder tarafından kaydedilmiş)
	workdir := "/"
	if wdData, err := os.ReadFile("/.mybox_workdir"); err == nil {
		wd := strings.TrimSpace(string(wdData))
		if wd != "" {
			workdir = wd
		}
	}

	entrypoint, err := os.ReadFile("/.mybox_entrypoint")
	var cmd *exec.Cmd
	if err == nil {
		cmdStr := strings.TrimSpace(string(entrypoint))
		// JSON array formatını temizle: ["go", "run", "server.go"] -> go run server.go
		replacer := strings.NewReplacer("[", "", "]", "", "\"", "", ",", "")
		cleanCmd := strings.TrimSpace(replacer.Replace(cmdStr))
		args := strings.Fields(cleanCmd)

		if len(args) > 0 {
			// WORKDIR'e geç
			if _, err := os.Stat(workdir); err == nil {
				os.Chdir(workdir)
				fmt.Printf("[*] Çalışma dizini: %s\n", workdir)
			}

			bin := args[0]
			rest := args[1:]

			// .sh dosyası → chmod +x yap ve /bin/sh ile çalıştır
			if strings.HasSuffix(bin, ".sh") {
				os.Chmod(bin, 0755)
				fmt.Printf("[*] Shell script çalıştırılıyor: %s\n", strings.Join(args, " "))
				cmd = exec.Command("/bin/sh", append([]string{bin}, rest...)...)
			} else if bin == "go" && len(rest) >= 2 && rest[0] == "run" {
				// 'go run server.go' gibi go kaynak dosyası → direkt go run
				fmt.Printf("[*] Go uygulaması başlatılıyor: %s\n", strings.Join(args, " "))
				cmd = exec.Command("go", rest...)
			} else {
				fmt.Printf("[*] Başlatılıyor: %s\n", strings.Join(args, " "))
				cmd = exec.Command(bin, rest...)
			}
		} else {
			cmd = exec.Command("/bin/sh")
		}
	} else {
		// Entrypoint yok: /bin/sh veya varsa otomatik tespit et
		fmt.Println("[*] Entrypoint bulunamadı, shell başlatılıyor...")
		cmd = exec.Command("/bin/sh")
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Konteyner içindeki ana uygulama bittiğinde konteyner ölür.
	if err := cmd.Run(); err != nil {
		fmt.Printf("Konteyner içi hata: %v\n", err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
