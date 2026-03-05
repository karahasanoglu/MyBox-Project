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
	data, _ := json.MarshalIndent(state, "", "  ")
	filename := filepath.Join(stateDir, fmt.Sprintf("%d.json", state.PID))
	return ioutil.WriteFile(filename, data, 0644)
}

func removeState(pid int) {
	filename := filepath.Join(stateDir, fmt.Sprintf("%d.json", pid))
	os.Remove(filename)
}

func Parent(flags []string, imagePath string, containerArgs []string) {
	bridgeName := "mybox0"
	bridgeIP := "10.0.0.1/24"

	must(network.SetupBridge(bridgeName, bridgeIP))
	must(network.SetupNAT(bridgeName))

	contID := fmt.Sprintf("%04x", rand.Intn(0x10000))
	hostVeth := "vh-" + contID
	contVeth := "vc-" + contID

	ipX := rand.Intn(253) + 1
	ipY := rand.Intn(253) + 2
	containerIP := fmt.Sprintf("10.0.%d.%d", ipX, ipY)
	containerIPCIDR := containerIP + "/24"

	// 1. Bayrakları işle (Port yönlendirme ve Kaynak Sınırları)
	var hostPort, contPort string
	var memLimit, cpuLimit string
	for i := 0; i < len(flags); i++ {
		switch flags[i] {
		case "-p":
			if i+1 < len(flags) {
				ports := strings.Split(flags[i+1], ":")
				if len(ports) == 2 {
					hostPort, contPort = ports[0], ports[1]
					must(network.SetupPortForwarding(hostPort, containerIP, contPort))
					localIP := network.GetLocalIP()
					fmt.Printf("\n[!] Konteynere şuradan erişebilirsiniz: http://%s:%s\n\n", localIP, hostPort)
				}
				i++
			}
		case "--memory":
			if i+1 < len(flags) {
				memLimit = flags[i+1]
				i++
			}
		case "--cpus":
			if i+1 < len(flags) {
				cpuLimit = flags[i+1]
				i++
			}
		}
	}

	// 2. Child sürecini başlat (Sadece komutu gönder)
	fmt.Printf("[*] Konteyner başlatılıyor: %s\n", filepath.Base(imagePath))
	cmd := exec.Command("/proc/self/exe", append([]string{"child", imagePath}, containerArgs...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"MYBOX_CONT_VETH="+contVeth,
		"MYBOX_CONT_IP="+containerIPCIDR,
		"MYBOX_CONT_ID="+contID,
	)

	must(cmd.Start())
	pid := cmd.Process.Pid

	// KAYNAK SINIRLARINI UYGULA (Bellek Chunking)
	must(cgroup.SetupCgroups(pid, memLimit, cpuLimit))
	defer cgroup.CleanupCgroups(pid)
	defer cleanupRootfs(contID)

	must(network.CreateVethPair(hostVeth, contVeth))
	must(network.AttachVethToBridge(hostVeth, bridgeName))
	must(network.MoveVethToNamespace(contVeth, pid))

	exec.Command("ip", "route", "add", containerIP+"/32", "dev", bridgeName).Run()
	defer exec.Command("ip", "route", "del", containerIP+"/32", "dev", bridgeName).Run()

	state := ContainerState{
		PID:       pid,
		ID:        contID,
		Image:     filepath.Base(imagePath),
		IP:        containerIP,
		HostPort:  hostPort,
		ContPort:  contPort,
		StartTime: time.Now(),
		Status:    "Running",
		Command:   strings.Join(containerArgs, " "),
	}
	saveState(state)

	cmd.Wait()
	network.RemovePortForwarding(hostPort, containerIP, contPort)
	removeState(pid)
}

func Child() {
	fmt.Println("\n---- [MyBox] Konteyner Başlatılıyor ----")
	imagePath := os.Args[2]
	contID := os.Getenv("MYBOX_CONT_ID")
	tempRootfs := "/tmp/mybox_rootfs_" + contID

	os.MkdirAll(tempRootfs, 0755)
	exec.Command("tar", "-xf", imagePath, "-C", tempRootfs).Run()

	contVeth := os.Getenv("MYBOX_CONT_VETH")
	containerCIDR := os.Getenv("MYBOX_CONT_IP")
	gateway := "10.0.0.1"

	time.Sleep(1 * time.Second)
	must(network.SetupContainerNetwork(contVeth, containerCIDR, gateway))

	must(syscall.Sethostname([]byte("mybox-container")))
	must(syscall.Chroot(tempRootfs))
	must(os.Chdir("/"))

	os.MkdirAll("/proc", 0755)
	must(syscall.Mount("proc", "/proc", "proc", 0, ""))

	// WORKDIR'e git (Cerrahi Kontrol)
	workdir := "/"
	if wdData, err := os.ReadFile("/.mybox_workdir"); err == nil {
		workdir = strings.TrimSpace(string(wdData))
	}
	
	// Eğer WORKDIR belirtilmişse ve yoksa oluştur (güvenlik için)
	if workdir != "/" {
		os.MkdirAll(workdir, 0755)
	}
	
	fmt.Printf("[*] Çalışma dizinine geçiliyor: %s\n", workdir)
	os.Chdir(workdir)

	// KRITIK: Dosya varlık kontrolü (Hata ayıklama için)
	if files, err := os.ReadDir("."); err == nil {
		fmt.Printf("[*] Mevcut dizin içeriği (%d dosya): ", len(files))
		for _, f := range files { fmt.Printf("%s ", f.Name()) }
		fmt.Println()
	}

	var finalArgs []string
	if len(os.Args) > 3 {
		finalArgs = os.Args[3:]
	} else {
		entrypoint, err := os.ReadFile("/.mybox_entrypoint")
		if err == nil {
			cmdStr := strings.TrimSpace(string(entrypoint))
			replacer := strings.NewReplacer("[", "", "]", "", "\"", "", ",", "")
			finalArgs = strings.Fields(replacer.Replace(cmdStr))
		}
	}

	if len(finalArgs) == 0 {
		finalArgs = []string{"/bin/sh"}
	}

	fmt.Printf("[*] Başlatılıyor: %s\n", strings.Join(finalArgs, " "))
	cmd := exec.Command(finalArgs[0], finalArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func ListContainers() {
	os.MkdirAll(stateDir, 0755)
	files, _ := ioutil.ReadDir(stateDir)
	fmt.Printf("%-10s %-22s %-15s %-15s %s\n", "PID", "IMAGE", "IP", "PORTS", "CREATED")
	fmt.Println(strings.Repeat("-", 80))

	for _, file := range files {
		data, _ := ioutil.ReadFile(filepath.Join(stateDir, file.Name()))
		var s ContainerState
		json.Unmarshal(data, &s)
		ports := fmt.Sprintf("%s:%s", s.HostPort, s.ContPort)
		fmt.Printf("%-10d %-22s %-15s %-15s %s\n", s.PID, s.Image, s.IP, ports, s.StartTime.Format("15:04:05"))
	}
}

func RemoveContainer(pidStr string) {
	fmt.Printf("[*] Konteyner durduruluyor: PID=%s\n", pidStr)
	exec.Command("sudo", "kill", "-9", pidStr).Run()
	pid, _ := strconv.Atoi(pidStr)
	if pid > 0 {
		removeState(pid)
	}
}

func safeCleanup(rootfs string) {
	if rootfs == "" || rootfs == "/" { return }
	data, _ := os.ReadFile("/proc/mounts")
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.Fields(lines[i])
		if len(fields) > 1 && strings.HasPrefix(fields[1], rootfs) {
			exec.Command("umount", "-l", fields[1]).Run()
		}
	}
	time.Sleep(100 * time.Millisecond)
	os.RemoveAll(rootfs)
}

func cleanupRootfs(id string) {
	safeCleanup("/tmp/mybox_rootfs_" + id)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
