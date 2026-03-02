# MyBox API test sürüm #

Bu belge, MyBox projesine entegre edilen RESTful API katmanı ve CLI (Komut Satırı Arayüzü) genişletmeleri hakkında teknik detayları içermektedir. Yapılan güncellemelerle birlikte MyBox, yerel CLI kullanımının yanı sıra HTTP protokolü üzerinden programatik olarak yönetilebilir hale getirilmiştir.

## Yapılan Değişiklikler Özeti ##

Sistem mimarisinde gerçekleştirilen temel güncellemeler şunlardır:

* Yeni CLI Komutu: mybox serve komutu eklenerek API sunucusunun (backend) başlatılması sağlandı.

* Router Yapılandırması: Yüksek performanslı Gin framework kullanılarak HTTP endpoint rotaları tanımlandı.

* Handler Mantığı: Konteyner ve imaj yönetimi operasyonları, modülerlik adına internal/api/handlers dizini altına taşındı.

* CORS Desteği: Electron, Tauri veya Web tabanlı arayüzlerin API ile sorunsuz iletişim kurabilmesi için CORS middleware entegrasyonu yapıldı.

* Postman Koleksiyonu: Geliştirme ve test süreçlerini standartlaştırmak amacıyla hazır bir Postman koleksiyonu projeye dahil edildi.

## API Sunucusunu Başlatma ##

API sunucusu varsayılan olarak 18080 portunda dinleme yapar. Sunucuyu ayağa kaldırmak için terminalde aşağıdaki komut çalıştırılmalıdır:

```

mybox serve 
```

## Endpointler ##

### 1. Konteyner Yönetimi ###

| Metot | Endpoint | Açıklama |
| :--- | :--- | :--- |
| `GET` | `/api/v1/containers` | Sistemde çalışan tüm konteynerleri listeler. |
| `POST` | `/api/v1/containers/run` | Tanımlanan parametrelerle yeni bir konteyner başlatır. |
| `GET` | `/api/v1/containers/:id` | Belirtilen ID'ye sahip konteynerin detaylı durumunu döndürür. |
| `DELETE` | `/api/v1/containers/:id` | Çalışan bir konteyneri durdurur ve sistemden temizler. |

### 2. İmaj Yönetimi ###

| Metot | Endpoint | Açıklama |
| :--- | :--- | :--- |
| `GET` | `/api/v1/images` | Yerel depodaki mevcut imajları listeler. |
| `POST` | `/api/v1/images/build` | `MyBoxFile` içeriğini kullanarak yeni bir imaj derler. |
| `DELETE` | `/api/v1/images/:name` | Belirli bir imajı sistemden kalıcı olarak siler. |
