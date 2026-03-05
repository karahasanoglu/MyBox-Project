#!/bin/bash

# Renkler (Görsellik için)
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # Renk Yok

echo -e "${BLUE}[*] MyBox Projesi Hazirlaniyor...${NC}"

# --- YAPILANDIRMA (Birden fazla konteyner için buraları değiştirin) ---
HOST_PORT=8080          # Host makinede açılacak port
CONT_PORT=80            # Konteyner içindeki port (genelde 80)
IMAGE_NAME="myapp"      # İmaj adı
MEMORY_LIMIT="512m"     # RAM sınırı
# -----------------------------------------------------------------------

echo -e "${BLUE}[*] MyBox Projesi Hazirlaniyor...${NC}"

# 1. Adım: Cerrahi Temizlik (Sadece bu portu kullananı durdur)
echo -e "${BLUE}[*] Port $HOST_PORT kontrol ediliyor...${NC}"
# Sadece bu portu kullanan konteynerin PID'sini bul
OLD_PID=$(mybox ps | grep "$HOST_PORT:$CONT_PORT" | awk '{print $1}')
if [ ! -z "$OLD_PID" ]; then
    echo -e "${YELLOW}[*] Port $HOST_PORT dolu. Eski konteyner durduruluyor: PID $OLD_PID${NC}"
    sudo mybox stop $OLD_PID > /dev/null 2>&1 || true
fi

# 2. Adım: İmajı Build Et (Dizine Duyarlı - Salted)
echo -e "${BLUE}[*] Imaj build ediliyor ($IMAGE_NAME)...${NC}"
sudo mybox build -t $IMAGE_NAME .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[+] Build basarili!${NC}"
else
    echo -e "\033[0;31m[-] Build sirasinda bir hata olustu.${NC}"
    exit 1
fi

# 3. Adım: Konteyneri Çalıştır
echo -e "${BLUE}[*] Konteyner baslatiliyor (Port: $HOST_PORT:$CONT_PORT)...${NC}"
echo "------------------------------------------------------------"
sudo mybox run -p $HOST_PORT:$CONT_PORT --memory $MEMORY_LIMIT $IMAGE_NAME
