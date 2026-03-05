# MyBox: Çalıştırma ve Kurulum Kılavuzu (RUN.md) 🛡️🚀🎯

Bu dosya, MyBox projesinin nasıl kurulacağını, çalıştırılacağını ve arkasındaki teknik mantığı özetler.

---

## 🛠️ 1. Kurulum ve Hazırlık

MyBox'ı sisteminize kurmak için proje ana dizininde bulunan `install.sh` scriptini kullanmalısınız.

```bash
# Proje dizinine girin
cd /path/to/MyBox-Project

# Kurulumu başlatın (Binary'yi sisteme kaydeder)
sudo bash install.sh
```

---

## 🚀 2. Projeleri Başlatma

Aşağıdaki projeler, kendi izole ağ alanlarında ve özel portlarında çalışacak şekilde yapılandırılmıştır.

### 🔹 Ana Proje (Port: 8080)
```bash
cd /path/to/MyBox-Project
./run.sh
```
*   **Erişim:** `http://<yerel-ip-adresiniz>:8080` (Örn: `http://192.168.1.152:8080`)

### 🔹 İkinci Proje (Port: 8081)
```bash
cd /path/to/New-Project
./run.sh
```
*   **Erişim:** `http://<yerel-ip-adresiniz>:8081` (Örn: `http://192.168.1.152:8081`)

### 🔹 Oyun/Üçüncü Proje (Port: 8082)
```bash
cd /path/to/neww
./run.sh
```
### 🔹 Yeni Proje Ekleme (Ölçeklendirme)
Eğer 2. veya 3. bir proje eklemek isterseniz:
1.  Proje klasöründeki `run.sh` dosyasını açın.
2.  En üstteki `HOST_PORT` (örn: `8083`) ve `IMAGE_NAME` (örn: `new-app`) değişkenlerini benzersiz değerlerle güncelleyin.
3.  İLGİLİ KLASÖRDE!!! `./run.sh` komutunu çalıştırın. MyBox otomatik olarak yeni izole alanı oluşturacaktır.

---

## ⚡ 3. Hızlı Başlatma ve Önbellek (Caching) Mantığı

MyBox, geliştirme sürecini hızlandırmak için akıllı bir önbellek mekanizması kullanır. `run.sh` scriptini çalıştırdığınızda şu süreç işler:

1.  **İlk Çalıştırma (Base Image Cache):** `MyBoxFile` içindeki `FROM` satırında belirtilen temel imaj (örn: `python:3.12-alpine`) bir kez Docker Hub'dan çekilir ve `/var/lib/mybox/cache` dizinine kaydedilir.
2.  **Sonraki Çalıştırmalar:** `run.sh` her çalıştığında `mybox build` komutu tetiklense de, sistem temel imajın zaten yerelde olduğunu anlar ve internetten tekrar indirmez. Bu, build süresini saniyelere düşürür.
3.  **Salted Build (İzole İnşa):** Her proje kendi klasör yoluna göre mühürlendiği için, bir projenin build dosyaları diğerlerini etkilemez ve bellek/disk üzerinde her zaman en güncel haliyle hazır bekler.

---

## 🏗️ 4. Teknik Mimari (Özet)

MyBox, modern konteyner teknolojilerinin (Namespaces, Cgroups, Layered FS) hafif bir implementasyonudur.

### 🛡️ Uygulanan Temel Özellikler:

1.  **Akıllı ve İzole Önbellek (Salted Build):**
    *   Farklı projelerin içeriklerinin birbirine karışmasını engellemek için her proje yoluna (`buildContext`) özel SHA-256 tabanlı bir hash üretilir.
    *   Build işlemleri `/tmp/mybox_build_<hash>` altında yapılır, böylece 8080'deki bir dosya 8082'yi asla kirletmez.

2.  **Bellek Sınırlandırma (RAM Chunking):**
    *   `--memory` bayrağı ile belirtilen limitler, Linux **Cgroups (v2)** üzerinden uygulanır.
    *   Sistem, konteynere belleği kontrollü parçalar (chunks) halinde sunar ve limit aşımında güvenli durdurma sağlar.

3.  **Cerrahi Ağ Temizliği (Surgical Networking):**
    *   Geleneksel `pkill` yerine, sadece ilgili portu (8080, 8081 vb.) kullanan süreci durduran bir mekanizma eklenmiştir.
    *   `iptables` kuralları cerrahi bir hassasiyetle temizlenir, böylece bir konteynerin durması diğerlerinin internetini kesmez.

4.  **İzole Dosya Sistemi (Rootfs):**
    *   Konteynerler `chroot` ve `mount namespaces` kullanarak ana sistemden tamamen izole edilir.
    *   `COPY . /app` komutu, projeyi tertemiz bir `/app` dizini içine kopyalar.

---

## 🏁 4. Doğrulama
*   Aynı anda birden fazla konteyner çalıştırılabilir.
*   `mybox ps` komutu ile çalışan konteynerlerin PID, IP ve PORT bilgilerini görebilirsiniz.
*   Her durdurma işleminde ağ kuralları ve cgroup'lar otomatik temizlenir.

**Geliştirici:** Onur 🏁🏆🎯🚀 ✅
