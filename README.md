# MyBox: Akıllı ve İzole Linux Konteyner Motoru (v1.2.7) 🚀

MyBox, Linux çekirdeği özelliklerini (Namespaces, Cgroups, Chroot) kullanarak oluşturulmuş, hafif, güvenli ve yüksek performanslı bir konteyner çalışma zamanıdır. "Smart Simplicity" felsefesiyle, karmaşıklıktan uzak ama modern konteyner ihtiyaçlarını (İzolasyon, Kaynak Sınırı) karşılayacak şekilde tasarlanmıştır.

---

## 🌟 Öne Çıkan Özellikler

### 1. Akıllı Önbellek (Salted Build Cache) 🛡️
Build işlemi sırasında her proje dizinine özel benzersiz bir hash üretilir. Bu sayede farklı projeler (örn: Port 8080 ve 8082) asla birbirinin dosyalarını "kirletmez".

### 2. Bellek Sınırlandırma (RAM Chunking) 🧠
`--memory` bayrağı ile belirtilen limitler, **Linux Cgroups v2** üzerinden gerçek zamanlı olarak uygulanır. Konteynerin RAM kullanımı kontrol altında tutulur.

### 3. Cerrahi Ağ Yönetimi (Surgical Networking) ⚔️
Ağ kuralları ve durdurma işlemleri port bazlı yapılır. Bir konteyneri durdurmak, aynı sistemde çalışan diğer MyBox projelerinizin ağını veya süreçlerini asla etkilemez.

### 4. Gelişmiş İzolasyon (Namespaces & Mounting) 🛡️
`UTS`, `PID`, `NS`, `NET` ve `IPC` namespace'leri kullanılarak tam izolasyon sağlanır. `chroot` ve akıllı mount mekanizması ile konteynerler ana sistemden güvenli bir şekilde ayrılır.

### 5. SSD Dostu ve Hızlı ⚡
Gereksiz katmanlama karmaşasından kaçınarak doğrudan ve hızlı dosya kopyalama mantığı kullanılır, bu da SSD ömrünü korur ve build sürelerini minimize eder.

---

## 🛠️ Hızlı Başlangıç

### Kurulum
```bash
sudo bash install.sh
```

### Build ve Çalıştır
```bash
# Proje dizininde
./run.sh
```

Ayrıntılı kullanım kılavuzu için [RUN.md](RUN.md) dosyasını inceleyin.

---

## 📊 Teknik Mimari
- **Dil:** Go (Pure)
- **Güvenlik:** SafeCleanup (Host dosya sistemi koruması)
- **Ağ:** Bridge (mybox0) + NAT + Port Forwarding
- **Kısıtlama:** Cgroups v2 (CPU/Memory)

---

**Geliştirici:** Onur 🏁🏆🎯🚀 ✅
