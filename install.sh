#!/bin/bash
# ============================================================
# MyBox - Gelişmiş Kurulum ve Yetkilendirme Scripti (v1.0.5)
# Kullanım: sudo bash install.sh
# ============================================================

set -e

# Renk kodları
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # Renk Yok

# Terminal Görseli (ASCII Art)
echo -e "${BLUE}"
echo "  __  __                ____"
echo " | \/  |_   _| __ )  _____  __"
echo " | |\/| | | | | _ \ / _ \ \/ /"
echo " | |  | | |_| | |_) | (_) >  <"
echo " |_|  |_|\__, |____/ \___/_/\_\\"
echo "          |___/  Container Runtime v1.0.5"
echo -e "${NC}"

# Root kontrolü
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}[-] Bu script root yetkisi gerektirir. 'sudo bash install.sh' kullanın.${NC}"
    exit 1
fi

# Proje dizininde olduğumuzu kontrol et (Arkadaşının detaylı hata mesajı ile)
if [ ! -f "go.mod" ] || [ ! -d "cmd/main" ]; then
    echo -e "${RED}[-] Hata: Bu script MyBox proje dizininden çalıştırılmalıdır.${NC}"
    echo -e "${YELLOW}    Örnek: cd $(pwd) && sudo bash install.sh${NC}"
    exit 1
fi

# 1. Sistem Bağımlılıklarının Kurulumu
echo -e "${YELLOW}[*] Sistem gereksinimleri kontrol ediliyor...${NC}"
apt-get update
apt-get install -y golang skopeo iproute2 iptables tar rsync

# 2. Go Modül Bağımlılıkları (Arkadaşının versiyonundan)
echo -e "${YELLOW}[*] Bağımlılıklar indiriliyor (go mod tidy)...${NC}"
go mod tidy

# 3. Gerekli Dizinlerin Oluşturulması
echo -e "${YELLOW}[*] Gerekli dizinler oluşturuluyor...${NC}"
mkdir -p /var/lib/mybox/images
mkdir -p /var/lib/mybox/containers
chmod -R 777 /var/lib/mybox
echo -e "${GREEN}[+] Dizinler hazır: /var/lib/mybox/${NC}"

# 4. MyBox Derleme
echo -e "${YELLOW}[*] MyBox derleniyor...${NC}"
go build -o /usr/local/bin/mybox ./cmd/main
chmod +x /usr/local/bin/mybox
echo -e "${GREEN}[+] Binary yüklendi: /usr/local/bin/mybox${NC}"

# 5. KRİTİK: Frontend Yetki Çözümü (setuid)
echo -e "${YELLOW}[*] Özel yetkiler (setuid) atanıyor...${NC}"
chown root:root /usr/local/bin/mybox
chmod u+s /usr/local/bin/mybox
echo -e "${GREEN}[+] setuid biti atandı. UI veya standart kullanıcı ile çalıştırılabilir.${NC}"

# 6. IP Forwarding Aktif Etme (Kalıcı yapılandırma)
echo -e "${YELLOW}[*] IP yönlendirme yapılandırılıyor...${NC}"
sysctl -w net.ipv4.ip_forward=1 > /dev/null
echo "net.ipv4.ip_forward=1" > /etc/sysctl.d/99-mybox.conf

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   MyBox başarıyla kuruldu ve yetkilendirildi!            ║${NC}"
echo -e "${GREEN}║   Artık sistemin her yerinden 'mybox' kullanabilirsiniz. ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Komut Referansı (Birleştirilmiş liste)
echo -e "Hızlı başlangıç:"
echo -e "  ${BLUE}mybox --version${NC}         → Sürümü göster"
echo -e "  ${BLUE}mybox images${NC}            → İmajları listele"
echo -e "  ${BLUE}mybox build myapp${NC}       → İmaj oluştur (MyBoxFile gerekli)"
echo -e "  ${BLUE}mybox run myapp${NC}         → Konteyner başlat (sudo gerektirmez)"
echo -e "  ${BLUE}mybox serve${NC}             → API/UI sunucusunu başlat"
echo -e "  ${BLUE}mybox ps${NC}                → Çalışan konteynerleri listele"