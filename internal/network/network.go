package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// SetupBridge köprü arayüzü yoksa oluşturur ve bir IP adresi atar.
func SetupBridge(bridgeName, bridgeIP string) error {
	// Köprünün var olup olmadığını kontrol et
	if err := exec.Command("ip", "link", "show", bridgeName).Run(); err == nil {
		fmt.Printf("[*] Köprü %s zaten mevcut\n", bridgeName)
		return nil
	}

	fmt.Printf("[*] %s köprüsü %s IP adresi ile oluşturuluyor\n", bridgeName, bridgeIP)

	// ip link add name mybox0 type bridge
	if err := exec.Command("ip", "link", "add", "name", bridgeName, "type", "bridge").Run(); err != nil {
		return fmt.Errorf("köprü oluşturulamadı: %v", err)
	}

	// ip addr add 10.0.0.1/24 dev mybox0
	if err := exec.Command("ip", "addr", "add", bridgeIP, "dev", bridgeName).Run(); err != nil {
		return fmt.Errorf("köprüye IP atanamadı: %v", err)
	}

	// ip link set dev mybox0 up
	if err := exec.Command("ip", "link", "set", "dev", bridgeName, "up").Run(); err != nil {
		return fmt.Errorf("köprü aktifleştirilemedi: %v", err)
	}

	return nil
}

// CreateVethPair bir veth çifti (sanal kablo) oluşturur.
func CreateVethPair(hostVeth, contVeth string) error {
	// Temizlik: Eğer eski arayüzler kaldıysa önce onları sil
	exec.Command("ip", "link", "delete", hostVeth).Run()

	fmt.Printf("[*] Veth çifti oluşturuluyor: %s <-> %s\n", hostVeth, contVeth)
	// ip link add veth-h type veth peer name veth-c
	if err := exec.Command("ip", "link", "add", hostVeth, "type", "veth", "peer", "name", contVeth).Run(); err != nil {
		return fmt.Errorf("veth çifti oluşturulamadı: %v", err)
	}
	return nil
}

// AttachVethToBridge veth çiftinin host ucunu köprüye bağlar.
func AttachVethToBridge(hostVeth, bridgeName string) error {
	fmt.Printf("[*] %s köprüye (%s) bağlanıyor\n", hostVeth, bridgeName)
	// ip link set veth-h master mybox0
	if err := exec.Command("ip", "link", "set", hostVeth, "master", bridgeName).Run(); err != nil {
		return fmt.Errorf("veth köprüye bağlanamadı: %v", err)
	}
	// ip link set veth-h up
	if err := exec.Command("ip", "link", "set", hostVeth, "up").Run(); err != nil {
		return fmt.Errorf("host veth ucu aktifleştirilemedi: %v", err)
	}
	return nil
}

// MoveVethToNamespace veth çiftinin konteyner ucunu konteynerin ağ alanına (namespace) taşır.
func MoveVethToNamespace(contVeth string, pid int) error {
	fmt.Printf("[*] %s, PID %d'nin ağ alanına taşınıyor\n", contVeth, pid)
	// ip link set veth-c netns <pid>
	if err := exec.Command("ip", "link", "set", contVeth, "netns", fmt.Sprintf("%d", pid)).Run(); err != nil {
		return fmt.Errorf("veth alana taşınamadı: %v", err)
	}
	return nil
}

// SetupContainerNetwork konteyner içindeki ağ arayüzünü yapılandırır.
// Bu fonksiyon konteynerin ağ alanı içinden çağrılmalıdır.
func SetupContainerNetwork(contVeth, contIP, gateway string) error {
	fmt.Printf("[*] Konteyner ağı kuruluyor: %s, IP: %s, Ağ Geçidi: %s\n", contVeth, contIP, gateway)

	// ip link set dev vc-xxxx name eth0
	// Standart olması için içeride ismini eth0 olarak değiştiriyoruz
	if err := exec.Command("ip", "link", "set", contVeth, "name", "eth0").Run(); err != nil {
		return fmt.Errorf("veth ismi eth0 olarak değiştirilemedi: %v", err)
	}

	// ip addr add 10.0.X.Y/24 dev eth0
	if err := exec.Command("ip", "addr", "add", contIP, "dev", "eth0").Run(); err != nil {
		return fmt.Errorf("eth0'a IP atanamadı: %v", err)
	}

	// ip link set dev eth0 up
	if err := exec.Command("ip", "link", "set", "eth0", "up").Run(); err != nil {
		return fmt.Errorf("eth0 aktifleştirilemedi: %v", err)
	}

	// ip link set dev lo up
	if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
		return fmt.Errorf("loopback (lo) aktifleştirilemedi: %v", err)
	}

	// Gateway farklı subnet'te olabilir (10.0.0.1 vs 10.0.91.0/24 gibi).
	// Önce 'scope link' route ile gateway'i direkt erişilebilir yap,
	// ardından default route ekle.
	exec.Command("ip", "route", "add", gateway, "dev", "eth0", "scope", "link").Run()

	// ip route add default via 10.0.0.1
	if err := exec.Command("ip", "route", "add", "default", "via", gateway).Run(); err != nil {
		return fmt.Errorf("varsayılan rota eklenemedi: %v", err)
	}

	return nil
}

// SetupNAT iptables kullanarak NAT yapılandırmasını yapar.
func SetupNAT(bridgeName string) error {
	fmt.Printf("[*] %s köprüsü için NAT kuruluyor\n", bridgeName)

	// IP yönlendirmenin (forwarding) etkin olduğundan emin ol
	if err := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run(); err != nil {
		return fmt.Errorf("IP yönlendirme etkinleştirilemedi: %v", err)
	}

	// POSTROUTING: Konteynerden çıkan paketler için NAT
	cmd := exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING", "-s", "10.0.0.0/8", "!", "-o", bridgeName, "-j", "MASQUERADE")
	if err := cmd.Run(); err != nil {
		exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "10.0.0.0/8", "!", "-o", bridgeName, "-j", "MASQUERADE").Run()
	}

	// FORWARD: Bridge üzerinden geçen paketlere izin ver (Cerrahi Check)
	if exec.Command("iptables", "-C", "FORWARD", "-i", bridgeName, "-j", "ACCEPT").Run() != nil {
		exec.Command("iptables", "-A", "FORWARD", "-i", bridgeName, "-j", "ACCEPT").Run()
	}
	if exec.Command("iptables", "-C", "FORWARD", "-o", bridgeName, "-j", "ACCEPT").Run() != nil {
		exec.Command("iptables", "-A", "FORWARD", "-o", bridgeName, "-j", "ACCEPT").Run()
	}

	return nil
}

// SetupPortForwarding host üzerindeki bir portu konteynerin IP ve portuna eşler.
func SetupPortForwarding(hostPort, contIP, contPort string) error {
	fmt.Printf("[*] Port Yönlendirme kuruluyor: Host %s -> Konteyner %s:%s\n", hostPort, contIP, contPort)

	// IP yönlendirmenin etkin olduğundan emin ol
	exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()

	// Önce o portla ilgili TÜM eski kuralları temizle (Nuclear Cleanup)
	purgePortRules(hostPort)

	// 1. PREROUTING: Dışarıdan gelen trafik için
	exec.Command("iptables", "-t", "nat", "-I", "PREROUTING", "-p", "tcp", "--dport", hostPort, "-j", "DNAT", "--to-destination", contIP+":"+contPort).Run()

	// 2. OUTPUT: Host makinenin kendinden gelen trafik için
	exec.Command("iptables", "-t", "nat", "-I", "OUTPUT", "-p", "tcp", "--dport", hostPort, "-j", "DNAT", "--to-destination", contIP+":"+contPort).Run()

	// 3. POSTROUTING: Yanıtın doğru dönmesi için paket maskeleme (ContPort'a Duyarlı)
	// Önce temizle ki mükerrer kural olmasın
	exec.Command("sh", "-c", fmt.Sprintf("iptables -t nat -S POSTROUTING | grep \"--dst %s/32\" | grep \"dport %s \" | sed 's/-A/-D/' | xargs -L 1 iptables -t nat 2>/dev/null", contIP, contPort)).Run()
	exec.Command("iptables", "-t", "nat", "-I", "POSTROUTING", "-p", "tcp", "--dst", contIP, "--dport", contPort, "-j", "MASQUERADE").Run()

	return nil
}

// purgePortRules belirtilen host portuna ait tüm eski iptables kurallarını kazır.
func purgePortRules(port string) {
	// 1. PREROUTING ve OUTPUT için temizlik (Host Portuna göre)
	for _, chain := range []string{"PREROUTING", "OUTPUT"} {
		for i := 0; i < 20; i++ {
			cmd := fmt.Sprintf("iptables -t nat -S %s | grep \"dport %s \" | head -n 1 | sed 's/-A/-D/'", chain, port)
			out, _ := exec.Command("sh", "-c", cmd).Output()
			ruleD := strings.TrimSpace(string(out))
			if ruleD == "" || !strings.Contains(ruleD, "-D") {
				break
			}
			exec.Command("sh", "-c", "iptables -t nat "+ruleD).Run()
		}
	}
	// 2. POSTROUTING için temizlik (Nuclear: Eski MASQUERADE kurallarını temizle)
	// Burada port bilgisi değişken olabildiği için en son eklenen geçersiz kuralları temizliyoruz
	exec.Command("sh", "-c", "iptables -t nat -S POSTROUTING | grep \"MASQUERADE\" | grep \"tcp\" | grep \"dport 80 \" | sed 's/-A/-D/' | xargs -L 1 iptables -t nat 2>/dev/null").Run()
}

// RemovePortForwarding konteyner durduğunda ilgili iptables kurallarını siler.
func RemovePortForwarding(hostPort, contIP, contPort string) error {
	if hostPort == "" {
		return nil
	}
	purgePortRules(hostPort)
	return nil
}

// GetLocalIP bilgisayarın yerel ağdaki (non-loopback) IP adresini bulur.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				// mybox0 gibi sanal arayüzleri atla
				if ipnet.IP.String() != "10.0.0.1" {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "127.0.0.1"
}
