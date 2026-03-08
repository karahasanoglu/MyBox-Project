# MyBox — Sıfırdan Yazılmış Linux Container Runtime

```
  __  __       ____
 |  \/  |_   _| __ )  _____  __
 | |\/| | | | |  _ \ / _ \ \/ /
 | |  | | |_| | |_) | (_) >  <
 |_|  |_|\__, |____/ \___/_/\_\
          |___/  Container Runtime v1.0.7
```

> **MyBox**, Docker'ın temelindeki teknolojileri (Linux Namespaces, Cgroups, Networking)
> kullanarak **sıfırdan Go diliyle** yazılmış bir konteyner çalıştırma motorudur.
> Eğitim ve araştırma amaçlı geliştirilmiştir.

---

## 📌 İçindekiler

1. [Proje Hakkında](#-proje-hakkında)
2. [Özellikler](#-özellikler)
3. [Sistem Gereksinimleri](#-sistem-gereksinimleri)
4. [Kurulum](#-kurulum)
5. [Komut Referansı](#-tam-komut-referansı)
6. [MyBoxFile Sözdizimi](#-myboxfile-sözdizimi)
7. [Örnek Kullanım Senaryosu](#-örnek-kullanım-senaryosu)
8. [Mimari ve Nasıl Çalışır](#-mimari-ve-nasıl-çalışır)
9. [Ağ Altyapısı](#-ağ-altyapısı)
10. [Proje Yapısı](#-proje-yapısı)
11. [Bilinen Sınırlılıklar](#-bilinen-sınırlılıklar)
12. [🆕 Yeni Kullanıcı Rehberi](#-yeni-kullanıcı-rehberi)

---

## 🎯 Proje Hakkında

MyBox, "Docker kaputun altında nasıl çalışır?" sorusuna pratik bir cevaptır.

Gerçek bir Docker imajının tersine, MyBox:
- **Hiçbir container daemon kullanmaz** (dockerd yok)
- **Hiçbir overlay filesystem kullanmaz** (her çalıştırmada tar açılır)
- **Doğrudan Linux system call'ları kullanır** (`clone`, `chroot`, `mount`, `unshare`)

Bu proje; namespaces, cgroups, veth-pair ve iptables gibi Linux çekirdek özelliklerini
bizzat kodlayarak konteyner teknolojisinin özünü anlamak için yazılmıştır.

---

## ✨ Özellikler

### Build (İmaj Oluşturma)
- `FROM` → Docker Hub'dan gerçek OCI imajı indirir (`skopeo` ile)
- `WORKDIR` → Konteyner içinde çalışma dizini ayarlar
- `COPY` → Host'tan dosyaları imaja kopyalar (akıllı hariç tutma: `mybox_rootfs`, `.tar`, `.git`)
- `RUN` → `chroot` içinde paket kurulumu ve komut çalıştırır (DNS otomatik yapılandırılır)
- `CMD` → Konteyner başlarken otomatik çalışacak komutu tanımlar
- `EXPOSE` → Port bilgilendirmesi

### Run (Konteyner Çalıştırma)
- **Process İzolasyonu:** UTS + PID + Mount + Network namespace
- **Filesystem Jail:** `chroot` + `/proc` auto-mount
- **WORKDIR Desteği:** `CMD` doğru dizinden otomatik çalışır
- **Akıllı Entrypoint:** `.sh` scriptler, `go run`, binary — hepsi desteklenir
- **Port Forwarding:** `-p hostPort:contPort` (iptables DNAT)
- **Volume Mount:** `-v /host:/cont` (bind mount)
- **Resource Limiting:** `--memory 512m`, `--cpus 0.5` (Cgroups v2)
- **Network:** NAT + bridge ile internet erişimi

### Yönetim
- `mybox images` → Oluşturulan tarih ve boyutuyla imajları listeler
- `mybox ps` → Çalışan konteynerleri PID/IP/Port/Başlangıç zamanıyla listeler
- `mybox rmi` → Bir veya birden fazla imajı siler
- `mybox stop` → Konteyneri düzgün durdurur (root process için otomatik sudo)

---

## 🖥 Sistem Gereksinimleri

| Gereksinim | Versiyon / Not |
|---|---|
| **İşletim Sistemi** | Linux (Yerel, VM, WSL2). Windows/macOS desteklenmez. |
| **Go** | 1.22 veya üzeri |
| **Skopeo** | `sudo apt install skopeo` |
| **iproute2** | `ip`, `bridge` komutları (genellikle kurulu) |
| **iptables** | NAT ve port yönlendirme için |
| **tar** | İmaj paketleme/açma |
| **Root yetkisi** | `sudo` — ağ, namespace ve cgroup işlemleri için |

---

## ⚡ Kurulum

### İlk Kurulum (Tek Seferlik)

```bash
# Proje dizinine git
cd /path/to/MyBox-Project

# Kurulum scriptini çalıştır
sudo bash install.sh
```

Bu komut:
1. `/var/lib/mybox/images/` ve `/var/lib/mybox/containers/` dizinlerini oluşturur
2. Projeyi derler (`go build`)
3. Binary'yi `/usr/local/bin/mybox` olarak sisteme yükler

**Kurulumdan sonra `mybox` komutu her dizinde, her terminalden kullanılabilir.**

### Dizinleri Yeniden Oluşturmak (Herhangi Bir Dizinde)

```bash
sudo mybox install
```

> Bu komut sadece `/var/lib/mybox/` dizin yapısını yeniden oluşturur.
> Binary zaten kuruluysa yeniden derleme yapmaz.

---

## 📋 Tam Komut Referansı

### `mybox build`

```bash
# Mevcut dizindeki MyBoxFile'ı kullanarak imaj oluştur
mybox build myapp
mybox build -t myapp        # Aynı işlem, -t flag ile

# Sonuç: /var/lib/mybox/images/myapp.tar
```

### `mybox images`

```bash
mybox images
```

```
REPOSITORY                TAG             SIZE       CREATED
---------------------------------------------------------------------------
myapp                     myapp           49 MB      2026-02-27 22:30
python-web                python-web      89 MB      2026-02-26 18:15
```

### `mybox rmi`

```bash
mybox rmi myapp                    # Tek imaj sil
sudo mybox rmi img1 img2 img3      # Birden fazla imaj sil
```

### `mybox run`

```bash
# Temel kullanım
sudo mybox run myapp

# Port yönlendirme ile (host:konteyner)
sudo mybox run -p 8080:80 myapp

# Volume mount ile
sudo mybox run -v $(pwd):/app myapp

# CPU ve RAM limitleri ile
sudo mybox run --cpus 0.5 --memory 512m myapp

```

### `mybox ps`

```bash
mybox ps
```

```
PID        IMAGE                  IP              PORTS              STATUS     CREATED
----------------------------------------------------------------------------------------------------
41023      myapp.tar              10.0.0.2        8080->80           Running    2026-02-27 22:34
```

### `mybox stop`

```bash
# PID'yi mybox ps'ten öğren
sudo mybox stop 41023
mybox stop 41023      # Root process ise otomatik sudo ile dener
```

### Diğer

```bash
mybox --version       # Sürüm bilgisi
mybox install         # Sadece dizinleri kur (herhangi bir dizinde)
```

---

## 📄 MyBoxFile Sözdizimi

MyBoxFile, Dockerfile'a benzer talimatlar içerir. Build zamanında bu dosya okunur.

```dockerfile
# Temel imaj (Docker Hub'dan çekilir)
FROM alpine:latest
FROM python:3.11-slim

# Çalışma dizini oluştur ve ayarla
WORKDIR /app

# Dosya veya dizin kopyala
COPY server.go /app/server.go    # Tek dosya
COPY . /app                      # Tüm dizin (mybox_rootfs, .git, .tar hariç)
COPY templates/ /app/templates/  # Alt dizin

# İmaj içinde komut çalıştır (internet erişimi var, DNS otomatik)
RUN apk add --no-cache python3 curl
RUN pip install flask
RUN go build -o server server.go

# Dinlenecek port (sadece bilgilendirme)
EXPOSE 80
EXPOSE 8080

# Konteyner başladığında çalışacak komut
CMD ["python3", "-m", "http.server", "80"]
CMD ["./server"]
CMD ["/app/start.sh"]
CMD ["go", "run", "server.go"]
```

> **Not:** `CMD` komutundaki binary, `WORKDIR` dizininde otomatik olarak çalıştırılır.
> `.sh` dosyaları için otomatik `chmod +x` uygulanır.

---

## 🧪 Örnek Kullanım Senaryosu

### Senaryo: Python HTTP Sunucusu

**1. Proje klasörü oluştur:**
```
my-web-app/
├── MyBoxFile
└── index.html
```

**2. MyBoxFile içeriği:**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache python3
WORKDIR /app
COPY index.html /app/
CMD ["python3", "-m", "http.server", "80"]
```

**3. İmaj oluştur:**
```bash
cd my-web-app
mybox build myweb
```

**4. Konteyneri çalıştır:**
```bash
sudo mybox run -p 8080:80 myweb
# [!] Konteynere şuradan erişebilirsiniz: http://192.168.1.100:8080
```

**5. Tarayıcıdan eriş:** `http://localhost:8080`

**6. Durumu kontrol et:**
```bash
mybox ps
# PID: 12345  IMAGE: myweb.tar  STATUS: Running
```

**7. Durdur:**
```bash
sudo mybox stop 12345
```

---

## 🏗 Mimari ve Nasıl Çalışır

### Build Süreci

```
MyBoxFile okunur
     │
     ├─ FROM alpine → skopeo ile Docker Hub'dan indirilir
     │                katmanlar ./mybox_rootfs/ içine açılır
     │
     ├─ RUN → sudo chroot ./mybox_rootfs /bin/sh -c "komut"
     │
     ├─ COPY → rsync (veya cp) ile dosyalar rootfs'e kopyalanır
     │
     ├─ WORKDIR → .mybox_workdir dosyasına kaydedilir
     │
     └─ CMD → .mybox_entrypoint dosyasına kaydedilir
                        │
                        └─ sudo tar -cf myapp.tar ./mybox_rootfs/
                           → /var/lib/mybox/images/myapp.tar
```

### Run Süreci

```
mybox run -p 8080:80 myapp
     │
     ├─ Parent Process:
     │   ├─ Bridge (mybox0) + NAT kurulur
     │   ├─ clone() → NEWUTS + NEWPID + NEWNS + NEWNET namespace
     │   ├─ Child PID alınır
     │   ├─ Cgroups kurulur (memory/cpu limit)
     │   ├─ veth-pair oluşturulur, namespace'e taşınır
     │   ├─ Port forwarding (iptables DNAT) kurulur
     │   └─ State JSON'a kaydedilir → /var/lib/mybox/containers/PID.json
     │
     └─ Child Process (izole namespace içinde):
         ├─ tar -xf myapp.tar → /tmp/mybox_rootfs_PID/
         ├─ Ağ yapılandırması (ip addr, ip route)
         ├─ Volume bind mount (-v)
         ├─ chroot /tmp/mybox_rootfs_PID/
         ├─ mount proc
         ├─ .mybox_workdir okunur → cd /app
         ├─ .mybox_entrypoint okunur → python3 -m http.server 80
         └─ Uygulama başlatılır ✅
```

---

## 🌐 Ağ Altyapısı

```
          İNTERNET
              │
         [eth0 / Wi-Fi]
              │
         iptables NAT (MASQUERADE)
              │
    [mybox0 bridge: 10.0.0.1]
              │
         [veth-h]──────[veth-c → eth0: 10.0.0.2]
                              │
                         [KONTEYNER]
                       python3 :80
                              │
              ┌───────────────┘
    iptables DNAT: 8080 → 10.0.0.2:80
              │
         localhost:8080
```

Detaylı teknik açıklama: [`docs/NETWORKING.md`](docs/NETWORKING.md)

---

## 📁 Proje Yapısı

```
MyBox-Project/
│
├── cmd/main/
│   └── main.go               ← CLI: tüm komutlar (build, run, ps, stop, rmi, images...)
│
├── internal/
│   ├── builder/
│   │   └── builder.go        ← MyBoxFile parser + skopeo entegrasyonu + imaj build
│   ├── runtime/
│   │   └── runtime.go        ← Parent/Child fork, chroot, state yönetimi
│   ├── network/
│   │   └── network.go        ← Bridge, veth-pair, NAT, port yönlendirme
│   └── cgroup/
│       └── cgroup.go         ← Cgroups v2: RAM ve CPU sınırlandırma
│
├── docs/
│   ├── README.md             ← Hızlı başlangıç kılavuzu
│   └── NETWORKING.md         ← Ağ altyapısı teknik detayları
│
├── examples/
│   └── web-server-demo/      ← Hazır çalışır örnek
│       ├── MyBoxFile         ← alpine + server.sh
│       ├── index.html        ← Web arayüzü
│       └── server.sh         ← Başlangıç scripti
│
├── scripts/
│   └── test_cgroups.sh       ← Cgroup test betiği
│
├── install.sh                ← Tek seferlik sistem kurulum scripti
├── go.mod                    ← Go modül tanımı (module: mybox, go 1.22)
└── FINAL_README.md           ← Bu dosya
```

---

## ⚠️ Bilinen Sınırlılıklar

| Konu | Durum |
|---|---|
| **Overlay Filesystem** | Yok. Her `run` için tar yeniden açılır (ephemeral) |
| **Image layer cache** | Yok. Her build sıfırdan başlar |
| **Daemon** | Yok. Arka plan servisi çalışmaz |
| **Windows / macOS** | Desteklenmez (Linux namespace gerektirir) |

---

## 📝 Geliştirme Notları

```bash
# Projeyi derle (binary oluşturmadan test)
go build ./...

# Değişiklik sonrası sisteme yükle
sudo go build -o /usr/local/bin/mybox ./cmd/main

# Veya install.sh ile
sudo bash install.sh
```

---

*MyBox v1.0.0 — Go 1.22 | Linux Namespaces | Cgroups v2 | iptables*

---

## 🆕 Yeni Kullanıcı Rehberi

> Bu bölüm, projeyi GitHub'dan çeken veya ilk kez kullanan kişiler için adım adım rehberdir.

### 📦 Adım 1 — Sistemi Hazırla

```bash
# Ubuntu / Debian için gerekli paketler
sudo apt update
sudo apt install -y golang skopeo iproute2 iptables

# Go versiyonunu kontrol et (1.22+ olmalı)
go version
```

### ⚙️ Adım 2 — MyBox'ı Kur (Tek Seferlik)

```bash
# Projeyi klonla
git clone https://github.com/kullanici/MyBox-Project.git
cd MyBox-Project

# Kurulum scriptini çalıştır
sudo bash install.sh
```

Başarılı kurulum çıktısı:
```
╔══════════════════════════════════════════════╗
║   MyBox başarıyla kuruldu!                   ║
║   Artık her yerden 'mybox' kullanabilirsiniz ║
╚══════════════════════════════════════════════╝
```

Artık terminali kapatsanız da `mybox` her yerden çalışır:
```bash
mybox --version
# MyBox Version: 1.0.7
```

---

### 🐳 Adım 3 — İlk Containerını Oluştur

#### Projen için klasör aç:
```bash
mkdir ~/benim-projem
cd ~/benim-projem
```

#### Projenin türüne göre `MyBoxFile` oluştur:

**HTML / CSS / JS (Statik Web):**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache python3
WORKDIR /app
COPY . /app
EXPOSE 80
CMD ["python3", "-m", "http.server", "80"]
```

**Python (Flask, FastAPI...):**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache python3 py3-pip
WORKDIR /app
COPY . /app
RUN pip install -r requirements.txt
EXPOSE 5000
CMD ["python3", "app.py"]
```

**Node.js:**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache nodejs npm
WORKDIR /app
COPY . /app
RUN npm install
EXPOSE 3000
CMD ["node", "server.js"]
```

**Go:**
```dockerfile
FROM alpine:latest
RUN apk add --no-cache go
WORKDIR /app
COPY . /app
RUN go build -o server .
EXPOSE 8080
CMD ["./server"]
```

---

### 🔨 Adım 4 — İmajı Build Et

```bash
# Proje klasörünün içindeyken:
cd ~/benim-projem
mybox build benim-uygulama

# İmajları listele → oluştu mu?
mybox images
```

```
REPOSITORY                TAG             SIZE       CREATED
---------------------------------------------------------------------------
benim-uygulama            benim-uygulama  52 MB      2026-02-28 00:30
```

---

### 🚀 Adım 5 — Containerı Çalıştır

```bash
# Port 8080'den erişmek için:
sudo mybox run -p 8080:80 benim-uygulama

# Arka planda çalıştırmak için:
sudo mybox run -p 8080:80 benim-uygulama &
```

Terminalde görürsün:
```
[!] Konteynere şuradan erişebilirsiniz: http://192.168.1.X:8080
```

Tarayıcıdan `http://localhost:8080` adresine gir. ✅

---

### 📊 Adım 6 — Yönet

```bash
# Çalışan containerları gör
mybox ps

# Çıktı:
# PID   IMAGE              IP            PORTS      STATUS    CREATED
# 1234  benim-uygulama    10.0.X.Y      8080->80   Running   2026-02-28 00:30

# Durdur
sudo mybox stop 1234

# İmajı sil (artık kullanmıyorsan)
mybox rmi benim-uygulama
```

---

### 📊 Aynı Anda Birden Fazla Container

Her container farklı port kullanmalıdır:

```bash
# Terminal 1 — Proje A
sudo mybox run -p 8080:80 proje-a &

# Terminal 2 — Proje B (farklı port)
sudo mybox run -p 9090:80 proje-b &

# İkisini birden gör
mybox ps
# PID   IMAGE     IP            PORTS      STATUS
# 1234  proje-a   10.0.45.12    8080->80   Running
# 5678  proje-b   10.0.89.33    9090->80   Running
```

> ⚠️ Aynı host portunu iki container'a veremezsin (8080'i iki kez kullanamazsın).

---

### ❓ Sık Karşılaşılan Sorunlar

| Sorun | Çözüm |
|---|---|
| `mybox: command not found` | `sudo bash install.sh` çalıştır |
| `skopeo: not found` | `sudo apt install skopeo` |
| Port açılmıyor | `sudo mybox run` kullandığından emin ol |
| `permission denied` | Container yönetimi için `sudo` gerekli |
| Build başarısız | `MyBoxFile`'ın proje klasöründe olduğunu kontrol et |
| Container hemen kapanıyor | `CMD` komutunun doğru çalıştığını test et |
