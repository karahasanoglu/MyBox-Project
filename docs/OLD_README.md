# MyBox — Linux Container Runtime

> Docker'ın temel mantığını sıfırdan Go ile inşa eden, Linux çekirdek özelliklerini
> (Namespaces, Cgroups, Networking) doğrudan kullanan minimalist bir konteyner motoru.

---

## 🚀 Temel Özellikler

| Özellik | Açıklama |
|---|---|
| **Custom Build Engine** | `MyBoxFile` (Dockerfile benzeri) talimatlarını işleyerek imaj oluşturur |
| **Docker Hub Entegrasyonu** | `skopeo` ile `FROM alpine`, `FROM python` gibi imajları doğrudan çeker |
| **Process İzolasyonu** | UTS, PID, Mount, Net namespace'leri ile tam izolasyon |
| **Filesystem Jail** | `chroot` ile süreçleri belirli bir kök dizine hapseder |
| **Network İzolasyonu** | veth-pair + bridge + NAT ile konteynere internet erişimi |
| **Port Forwarding** | `-p 8080:80` ile host portunu konteynere yönlendirir |
| **Volume Mount** | `-v /host:/cont` ile dizin paylaşımı |
| **Resource Limiting** | Cgroups v2 ile RAM ve CPU sınırı |
| **Container State** | Çalışan konteynerlerin PID/IP/Port bilgisi kaydedilir |

---

## 🛠 Sistem Gereksinimleri

- **İşletim Sistemi:** Linux (Yerel, VM veya WSL2). Windows/macOS desteklenmez.
- **Go:** 1.22 veya üzeri
- **Skopeo:** `sudo apt install skopeo`
- **iproute2 / iptables:** Genellikle kurulu gelir
- **Root yetkisi:** Ağ ve namespace işlemleri için `sudo` gereklidir

---

## ⚡ Kurulum (Tek Seferlik)

```bash
# 1. Proje dizinine gir
cd /path/to/MyBox-Project

# 2. Kurulum scriptini çalıştır
sudo bash install.sh
```

Kurulum sonrası `mybox` komutu **her dizinde** kullanılabilir.

---

## 📋 Komut Referansı

```bash
# İmaj oluştur (proje dizininde MyBoxFile olmalı)
mybox build myapp
mybox build -t myapp        # -t flag'i ile aynı şey

# İmajları listele
mybox images

# İmaj sil
mybox rmi myapp
mybox rmi img1 img2 img3   # Birden fazla

# Konteyner başlat
sudo mybox run myapp
sudo mybox run -p 8080:80 myapp
sudo mybox run -p 8080:80 -v $(pwd):/app myapp
sudo mybox run --memory 512m --cpus 0.5 myapp

# Çalışan konteynerleri listele
mybox ps

# Konteyneri durdur
sudo mybox stop <PID>

# Sürüm
mybox --version
```

---

## 📄 MyBoxFile Sözdizimi

```dockerfile
# Temel imaj — Docker Hub'dan çekilir
FROM alpine:latest

# Çalışma dizini
WORKDIR /app

# Dosya kopyala (host → imaj)
COPY . /app
COPY server.go /app/server.go

# İmaj içinde komut çalıştır
RUN apk add --no-cache python3
RUN go build -o server server.go

# Konteynerin dinleyeceği port (bilgilendirme)
EXPOSE 80

# Konteyner başladığında çalışacak komut
CMD ["python3", "-m", "http.server", "80"]
CMD ["./server"]
CMD ["/app/start.sh"]
```

---

## 🌐 Ağ Mimarisi

```
Ana Makine (Host)
├── eth0  ──── İnternet
│               │
│           iptables NAT
│               │
├── mybox0 (bridge, 10.0.0.1)
│       │
│   veth-h ─────────── veth-c (10.0.0.2)
│                           │
│                      [KONTEYNER]
│                       └── eth0
```

Daha fazla detay için: [`NETWORKING.md`](NETWORKING.md)

---

## 📁 Proje Yapısı

```
MyBox-Project/
├── cmd/main/main.go          ← CLI giriş noktası (tüm komutlar burada)
├── internal/
│   ├── builder/builder.go    ← MyBoxFile parser + imaj build motoru
│   ├── runtime/runtime.go    ← Container başlatma/durdurma/listeleme
│   ├── network/network.go    ← Bridge, veth-pair, NAT, port yönlendirme
│   └── cgroup/cgroup.go      ← RAM/CPU sınırlandırma (Cgroups v2)
├── docs/
│   ├── README.md             ← Bu dosya
│   └── NETWORKING.md         ← Ağ altyapısı teknik detayları
├── examples/
│   └── web-server-demo/      ← Hazır örnek: Alpine + web sunucusu
│       ├── MyBoxFile
│       ├── index.html
│       └── server.sh
├── scripts/
│   └── test_cgroups.sh       ← Cgroup test betiği
├── install.sh                ← Tek seferlik sistem kurulum scripti
└── go.mod                    ← Go modül tanımı (module: mybox)
```
