# Update log #

## 1.0.0 -> 1.0.2 ##

router ve handler dosyalarının yapımı ve birkaç küçük değişim

## 1.0.2 -> 1.0.4 ##

port güvenlikleri
??

## 1.0.4 -> 1.0.5 ##

handler dosyalarını ayırma
readme isimleri düzenleme (README isimli dosya default ana sayfaya geliyor ayarlamazsak onun için,)

cgroup.go 'ya UpdateCgroups fonksiyonu (fonksiyon ayrılığı için)
container_handler.go 'ya UpdateContainerResourcesHandler fonksiyonu yazılı (temel olarak UpdateCgroups çağırıyor)

system_handler yazımı
belki update koyulabilir

StopContainerHandler güncellemesi, önce güvenli kapamayı deniyor artık

yeni fonksiyonların routera yazımı
postman koleksiyonu güncellemesi

-K: system handler da install fonksiyonu düşündüm ama mantıklı değil, front end tamamen ayrı bir uygulama olmalı o zaman yani. Bu frontendde sağlanabilir belki, backende bağlanmadan (backend backendi kuramaz yani sonuçta)
Windows için bir bat dosyası öneririm, linux için bilmiyom ama masaüstü uygulamasından olabilyor sanırım
son haline geçmemiş olsa da yavaştan testleri de yazdırırım
diğer readmeyi güncellemedim çok bişi  yok zaten eklediğim

