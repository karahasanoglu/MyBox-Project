#!/bin/bash
# ============================================================
# MyBox - Tek Seferlik Kurulum Scripti
# Kullanım: sudo bash install.sh
# ============================================================

set -e

# Renk kodları
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "  __  __       ____"
echo " |  \/  |_   _| __ )  _____  __"
echo " | |\/| | | | |  _ \ / _ \ \/ /"
echo " | |  | | |_| | |_) | (_) >  <"
echo " |_|  |_|\__, |____/ \___/_/\_\\"
echo "          |___/  Container Runtime v1.0.0"
echo -e "${NC}"

# Root kontrolü
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}[-] Bu script root yetkisi gerektirir. 'sudo bash install.sh' kullanın.${NC}"
    exit 1
fi

# Proje dizininde olduğumuzu kontrol et
if [ ! -f "go.mod" ] || [ ! -d "cmd/main" ]; then
    echo -e "${RED}[-] Hata: Bu script MyBox proje dizininden çalıştırılmalıdır.${NC}"
    echo -e "    Örnek: cd /home/emirfurkan/Desktop/MyBox-Project-final/MyBox-Project && sudo bash install.sh"
    exit 1
fi

echo -e "${YELLOW}[*] Gerekli dizinler oluşturuluyor...${NC}"
mkdir -p /var/lib/mybox/images
mkdir -p /var/lib/mybox/containers
chmod -R 777 /var/lib/mybox
echo -e "${GREEN}[+] Dizinler hazır: /var/lib/mybox/${NC}"

echo -e "${YELLOW}[*] MyBox derleniyor...${NC}"
go build -o /usr/local/bin/mybox ./cmd/main
chmod +x /usr/local/bin/mybox
echo -e "${GREEN}[+] Binary yüklendi: /usr/local/bin/mybox${NC}"

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   MyBox başarıyla kuruldu!                   ║${NC}"
echo -e "${GREEN}║   Artık her yerden 'mybox' kullanabilirsiniz ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Hızlı başlangıç:"
echo -e "  ${BLUE}mybox --version${NC}       → Sürümü göster"
echo -e "  ${BLUE}mybox build myapp${NC}     → İmaj oluştur  (MyBoxFile gerekli)"
echo -e "  ${BLUE}mybox images${NC}          → İmajları listele"
echo -e "  ${BLUE}sudo mybox run myapp${NC}  → Konteyner başlat"
echo -e "  ${BLUE}mybox ps${NC}              → Çalışanları listele"
echo -e "  ${BLUE}sudo mybox stop <PID>${NC} → Konteyneri durdur"
echo -e "  ${BLUE}mybox rmi myapp${NC}       → İmajı sil"
