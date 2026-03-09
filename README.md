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

### Hızlı Başlangıç & Masaüstü Uygulaması 
Kurulum adımları ve MyBox Manager görsel arayüzünü indirmek için [resmi sayfamızı](https://verdant-mermaid-fffa0a.netlify.app/) ziyaret edebilirsiniz.

## 📌 İçindekiler
* [Proje Hakkında](#-proje-hakkında)
* [Özellikler](#-özellikler)
* [Sistem Gereksinimleri](#-sistem-gereksinimleri)
* [Kurulum](#-kurulum)
* [Komut Referansı](#-komut-referansı)
* [MyBoxFile Sözdizimi](#-myboxfile-sözdizimi)
* [Ağ Altyapısı](#-ağ-altyapısı)
* [Mimari Özet](#mimari-ozet)
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

### Konteyner Motoru
- **Tam İzolasyon:** UTS, PID, Mount ve Network namespace'leri ile süreç seviyesinde güvenlik.
- **Kaynak Yönetimi:** Cgroups v2 altyapısı ile anlık CPU (--cpus) ve RAM (--memory) sınırlandırmaları.
- **Ağ Katmanı:** Özel bridge (köprü) mimarisi, iptables tabanlı port yönlendirme (DNAT) ve otomatik internet erişimi (NAT).

### İmaj Yönetimi (Build)
- **Docker Hub Entegrasyonu:** skopeo altyapısı ile standart OCI imajlarını taban olarak kullanabilme.
- **MyBoxFile Desteği:** Tanıdık bir sözdizimi ile özelleştirilmiş imajlar oluşturma (FROM, RUN, COPY, CMD, vb.).
- **Akıllı Entrypoint:** İmaj başlatıldığında shell script'leri, çalıştırılabilir binary'leri veya Go kodlarını otomatik algılama ve çalıştırma.

### Görsel Yönetim
- **MyBox Manager UI:** Konteynerleri, imajları ve sistem loglarını yönetebileceğiniz modern bir masaüstü arayüzü.

---

## 🖥 Sistem Gereksinimleri

| Gereksinim | Versiyon / Not |
|---|---|
| **İşletim Sistemi** | Linux (Yerel, VM, WSL2). Windows/macOS desteklenmez. |
| **Go** | 1.25 veya üzeri |
| **Skopeo** | `sudo apt install skopeo` |
| **iproute2** | `ip`, `bridge` komutları (genellikle kurulu) |
| **iptables** | NAT ve port yönlendirme için |
| **tar** | İmaj paketleme/açma |
| **Root yetkisi** | `sudo` — ağ, namespace ve cgroup işlemleri için |

| Uygulama                           | Gereksinimler                                              |
|------------------------------------|----------------------------------------------------------|
| Masaüstü Uygulaması (MyBox Manager UI) | Wails v2 framework                                       |
|                                    | Go backend + HTML/CSS/JavaScript frontend                |
|                                    | Minimum 512MB RAM                                       |
|                                    | Linux, Windows (via WSL2), or macOS                     |
| Web Uygulaması (Landing Page)   | Modern web browser (Chrome, Firefox, Safari, Edge)      |
|                                    | JavaScript enabled                                       |
|                                    | Responsive design (works on desktop and mobile)          |

---

## ⚡ Kurulum
### Seçenek 1: Hızlı Kurulum Betiği (CLI)
MyBox CLI'yi sisteminize entegre etmek için terminalinizde aşağıdaki komutu çalıştırın:

```bash
curl -sSL https://raw.githubusercontent.com/karahasanoglu/MyBox-Project/version-1.0.7/install.sh | sudo bash
```
*(Bu komut gerekli dizinleri oluşturur ve `/usr/local/bin/mybox` yoluna uygulamayı kaydeder.)*

### Seçenek 2: Görsel Arayüz (Desktop App)
Uygulamayı görsel bir arayüzle yönetmek isterseniz, resmi web sayfamızdan MyBox Manager v1.0.7 masaüstü uygulamasını indirebilirsiniz.

---

## 📋 Komut Referansı
Aşağıdaki komutları sistemin herhangi bir noktasından çalıştırabilirsiniz:

| Komut | Açıklama |
| :--- | :--- |
| `mybox build <isim>` | Mevcut dizindeki MyBoxFile'ı kullanarak yeni bir imaj inşa eder. |
| `mybox images` | Sistemdeki oluşturulmuş imajları boyutu ve tarihiyle listeler. |
| `sudo mybox run <isim>` | Belirtilen imajı başlatır. |
| `mybox ps` | Aktif olarak çalışan konteynerleri (PID, IP, Port verileriyle) listeler. |
| `sudo mybox stop <PID>` | Çalışan bir konteyneri güvenle sonlandırır ve ağ kurallarını temizler. |
| `mybox rmi <isim>` | İmajı sistemden siler. |

### Gelişmiş Konteyner Çalıştırma (Kaynak Sınırlandırma & Ağ)

MyBox, Linux Cgroups v2 entegrasyonu sayesinde süreçlerin sistem kaynaklarını tüketmesini engeller ve iptables üzerinden ağ yönlendirmesi sağlar.

```bash
# Sadece port yönlendirme (Host 8080 -> Container 80)
mybox run -p 8080:80 myapp

# RAM (Bellek) ve CPU sınırlandırması (Örn: Max 512MB RAM ve yarım CPU çekirdeği)
mybox run --memory 512m --cpus 0.5 myapp

# Volume Mount (Host dizinini konteynere bağlama)
mybox run -v /host/dizin:/app myapp

# Tüm gelişmiş parametrelerin bir arada kullanımı
mybox run -p 8080:80 -v $(pwd):/app --memory 1g --cpus 1.0 myapp
```

----

## 📄 MyBoxFile Sözdizimi
İmajlarınızı tıpkı bir Dockerfile yazar gibi oluşturabilirsiniz:

```dockerfile
# Temel imaj belirleme
FROM alpine:latest

# Çalışma dizinini ayarlama
WORKDIR /app

# Dosyaları konteyner içine kopyalama
COPY . /app

# İmaj oluşturulurken komut çalıştırma (Bağımlılık yükleme vb.)
RUN apk add --no-cache python3

# Bilgilendirme amaçlı port tanımı
EXPOSE 8080

# Konteyner başlatıldığında çalışacak varsayılan komut
CMD ["python3", "-m", "http.server", "8080"]
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

---
<a name="mimari-ozet"></a>
## 🏗️ Mimari Özet

MyBox, işlemleri şu temel hiyerarşi ile yürütür:

1. **Fork & Namespace:** Parent process, `clone` sistemi ile yeni bir Child process yaratır ve onu yeni UTS, PID, Mount ve Net namespace'lerine taşır.
2. **Cgroups:** Child process'in kaynak tüketimini (RAM/CPU) sınırlandırmak için özel cgroup kuralları sisteme işlenir.
3. **Network Setup:** Sanal bir kablo (veth-pair) oluşturulur; bir ucu host üzerindeki bridge arayüzüne (`mybox0`) bağlanırken diğer ucu konteyner alanına taşınır.
4. **Filesystem Isolation:** Özel root filesystem geçici bir klasöre açılır ve `chroot` kullanılarak süreç bu izole alana kilitlenir.
5. **Execution:** Belirtilen `CMD` komutu, izole ortamda PID 1 olarak çalıştırılır.
# 🖥 Sistem Gereksinimleri
