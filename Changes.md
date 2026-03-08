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

## 1.0.5 -> 1.0.6 ##

mybox-ui tamamı
install sh değişikliği (artık herşeye otomatik yetki veriyor)
launch.sh ekkendi

cmd/main/main.go güncellendi (-EFK)
internalde builder, runtime, network güncellendi (-EFK)

## GÜNCEL DURUM ##

İnstall etme işlemi nispeten kolay, gereksinimlerin olmadığı durum sıkıntı yapabilir belki
Front temelde çalışıyor
Front eksikleri: Kaynak güncellenebiliyor, şu anki kaynak durumu görülmüyor.
Belirsiz gereksinim: Bir log sistemi var, ama apiyi logluyor. Api çağrılarına bağlı olduğu için bilmiyom tam istediğimiz bu değil herhalde.

### Mümkün geliştirmeler ###

Apide ve frontta ufak değişiklikler (kaynak gösterimi ve log geliştirmesi, -K buna bakacak)
Şu anda koneynerler siliniyor (Onur buna bakıyrodu da bilmiyom eski versiyondan )
md dosyalarının düzelmesi

### Çalıştırma ###

Readme güncellemeye üşendim buraya yapıştırcam, linux 1.0.6 çalıştırma yöntemi
sudo bash install.sh (ana directoryde)
wails build -v 2 -tags webkit2_41 (mybox-ui'de, bunu otomatize edebilir miyiz bilmiyom deneyelim, bu aşamada gereksinimler yoksa sıkıntı çıkartıyor)
sudo bash launch.sh (ana directoryde, sudo lazım mı emin değilim ama koyuyom, önceki aşamadki wails binary dosyası oluşturma gerçekleşmediyse dosya yok agam diyor sadece)