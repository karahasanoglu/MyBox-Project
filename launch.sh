#!/bin/bash

# 1. Temizlik: Eğer portu kullanan eski bir servis varsa temizle
sudo killall -9 mybox 2>/dev/null || true

# 2. API'yi arka planda başlat (Root yetkisi ile)
echo "[*] MyBox API başlatılıyor..."
sudo mybox serve > /tmp/mybox-api.log 2>&1 &
API_PID=$!

# API'nin hazır olması için kısa bir bekleme
sleep 1

# 3. UI Uygulamasını başlat (Sandbox hatasını önlemek için normal kullanıcı olarak)
echo "[*] MyBox Manager UI başlatılıyor..."
if [ "$SUDO_USER" ]; then
    # Eğer script sudo ile çalıştırıldıysa, UI'yi asıl kullanıcı yetkisiyle açar
    sudo -u "$SUDO_USER" ./mybox-ui/build/bin/mybox-ui
else
    ./mybox-ui/build/bin/mybox-ui
fi

# 4. UI kapatıldığında arka plandaki API'yi de temizle
echo "[*] Kapatılıyor..."
sudo kill $API_PID