#!/bin/bash

# 1. Temizlik: Eğer portu kullanan eski bir servis varsa temizle
sudo killall -9 mybox 2>/dev/null || true

# Yetki Kontrolü
if [ "$EUID" -ne 0 ]; then
  echo "[-] Lütfen bu scripti sudo ile çalıştırın: sudo bash launch.sh"
  exit 1
fi

# Yetkileri garantile
chmod +x ./mybox
chmod +x ./mybox-ui 2>/dev/null || true
chmod +x ./mybox-ui/build/bin/mybox-ui 2>/dev/null || true

# 2. API'yi arka planda başlat (Root yetkisi ile)
echo "[*] MyBox API başlatılıyor..."
./mybox serve > /tmp/mybox-api.log 2>&1 &
API_PID=$!

# API'nin hazır olması için kısa bir bekleme
sleep 2

# 3. UI Uygulamasını başlat (Sandbox hatasını önlemek için normal kullanıcı olarak)
echo "[*] MyBox Manager UI başlatılıyor..."
UI_BIN="./mybox-ui"
if [ ! -f "$UI_BIN" ]; then
    UI_BIN="./mybox-ui/build/bin/mybox-ui"
fi

# UI'yi asıl kullanıcı yetkisiyle açar (root olarak UI açmak sandbox hatası verir)
if [ "$SUDO_USER" ]; then
    sudo -u "$SUDO_USER" "$UI_BIN"
else
    "$UI_BIN"
fi

# 4. UI kapatıldığında arka plandaki API'yi de temizle
echo "[*] Kapatılıyor..."
kill $API_PID