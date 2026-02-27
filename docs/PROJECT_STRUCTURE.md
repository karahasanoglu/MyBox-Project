# 🗂 MyBox Proje Yapısı (Architecture)

MyBox, baştan sona Go diliyle yazılmış, Linux'un çekirdek özelliklerini
(Namespaces, Cgroups, Networking) kullanarak izolasyon sağlayan bir Konteyner
Çalıştırma Motoru'dur. **Standart Go Proje Yapısı**'na (Standard Go Project Layout)
uygun şekilde tasarlanmıştır.

---

## 🏗 `cmd/` (Giriş Noktası)

### `cmd/main/main.go`
Tüm CLI komutlarının yönetildiği merkezi dosya.

| Fonksiyon | Komut | Açıklama |
|---|---|---|
| `handleBuild()` | `mybox build` | İmaj oluşturma mantığı |
| `handleRun()` | `mybox run` | Konteyner başlatma |
| `handlePS()` | `mybox ps` | Çalışan konteynerleri listele |
| `handleStop()` | `mybox stop` | Konteyneri durdur |
| `handleImages()` | `mybox images` | İmaj listesi |
| `handleRMI()` | `mybox rmi` | İmaj silme |
| `handleInstall()` | `mybox install` | Dizin kurulumu |

---

## 🧠 `internal/` (Çekirdek Modüller)

`internal/` altındaki paketler yalnızca bu proje tarafından kullanılabilir.

### `internal/builder/builder.go`
Konteyner imajlarını inşa eden motor — projenin "mutfağıdır".

- `BuildImage()` → MyBoxFile'ı satır satır okuyup işler
- `pullAndExtractImage()` → `skopeo` ile Docker Hub'dan OCI katmanlarını indirir
- `saveAsImage()` → Hazırlanan rootfs'i `.tar` olarak paketler
- COPY: `mybox_rootfs`, `.tar`, `.git` dizinlerini otomatik hariç tutar
- WORKDIR: `.mybox_workdir` dosyasına kaydeder (runtime okur)
- CMD: `.mybox_entrypoint` dosyasına kaydeder (runtime okur)

### `internal/runtime/runtime.go`
İmajı alıp çalışan konteynere dönüştüren motor — "heykele can veren" kısım.

- `Parent()` → Bridge & NAT kurar, child'ı fork eder, cgroup & network kurar, state kaydeder
- `Child()` → tar açar, ağ kurar, chroot yapar, entrypoint'i çalıştırır
- `ListContainers()` → State JSON'larını okur, canlı PID'leri listeler (EPERM farkındalığı)
- `RemoveContainer()` → kill (sudo fallback) + state dosyası silme
- `saveState()` / `removeState()` → `/var/lib/mybox/containers/<PID>.json` yönetimi

### `internal/network/network.go`
Tüm ağ altyapısı burada.

- `SetupBridge()` → `mybox0` köprüsünü oluşturur (10.0.0.1)
- `CreateVethPair()` → Sanal kablo çifti (veth-h ↔ veth-c)
- `AttachVethToBridge()` → Host ucunu köprüye bağlar
- `MoveVethToNamespace()` → Konteyner ucunu izole network namespace'e taşır
- `SetupContainerNetwork()` → Konteyner içinde IP + route tanımlar
- `SetupNAT()` → iptables MASQUERADE + ip_forward
- `SetupPortForwarding()` → iptables DNAT (PREROUTING + OUTPUT)
- `GetLocalIP()` → Host'un fiziksel IP adresini döner

### `internal/cgroup/cgroup.go`
Konteyner kaynak sınırlamaları.

- `SetupCgroups()` → `/sys/fs/cgroup/mybox_<PID>/` oluşturur
  - `memory.max` → RAM sınırı (512m, 1g, ...)
  - `cpu.max` → CPU quota/period hesaplaması
  - `cgroup.procs` → Child PID'yi gruba ekler
- `CleanupCgroups()` → Konteyner bitince cgroup dizinini temizler

---

## 📚 `docs/` (Dokümantasyon)

- **`README.md`** → Hızlı başlangıç: kurulum, komutlar, MyBoxFile sözdizimi
- **`NETWORKING.md`** → Ağ altyapısının teknik detayları ve test adımları

---

## 🛠 `scripts/` (Yardımcı Araçlar)

- **`test_cgroups.sh`** → RAM ve CPU cgroup ayarlarını terminalden manuel test etmek için

---

## 🚀 `examples/` (Örnek Uygulamalar)

### `examples/web-server-demo/`
MyBox'ın gücünü görmek için hazır çalışır örnek.

```
web-server-demo/
├── MyBoxFile    ← alpine + python3 + server.sh
├── index.html   ← Web arayüzü
└── server.sh    ← HTTP server başlatma scripti
```

Çalıştırmak için:
```bash
cd examples/web-server-demo
mybox build web-demo
sudo mybox run -p 8080:80 web-demo
```

---

## 📄 Kök Dizin Dosyaları

| Dosya | Açıklama |
|---|---|
| `go.mod` | Modül tanımı: `module mybox`, `go 1.22` |
| `install.sh` | **Tek seferlik** kurulum: derleme + sistem yükleme |
| `FINAL_README.md` | Kapsamlı proje dokümantasyonu |

---

## 🔄 Veri Akışı Özeti

```
[Kullanıcı]
    │
    ├─ mybox build myapp
    │       │
    │  [builder.go] ──skopeo──▶ Docker Hub
    │       │
    │  /var/lib/mybox/images/myapp.tar
    │
    └─ sudo mybox run -p 8080:80 myapp
            │
       [runtime.go Parent]
            ├─ network.go → bridge + veth + NAT
            ├─ cgroup.go  → memory/cpu limit
            └─ fork()
                   │
            [runtime.go Child] (izole namespace)
                   ├─ tar -xf myapp.tar → /tmp/mybox_rootfs_PID/
                   ├─ network.go → container IP + route
                   ├─ chroot + mount /proc
                   └─ .mybox_entrypoint çalıştır ✅
```
