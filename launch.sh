#!/bin/bash

# 1. API'yi arka planda başlat (Port: 18080)
# Binary /usr/local/bin altında olduğu için direkt çağırılabilir
mybox serve > /tmp/mybox-api.log 2>&1 &
API_PID=$!

# 2. UI Uygulamasını başlat (Wails binary'si)
# Not: UI binary adınız mybox-ui/build/bin altında oluşur
./mybox-ui/build/bin/mybox-ui

# 3. UI kapatıldığında arka plandaki API'yi de temizle
kill $API_PID