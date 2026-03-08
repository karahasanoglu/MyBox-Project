## install.sh Neden Hata Veriyor?

Evet, haklısınız. Mevcut `install.sh` dosyasının içerisinde çevrimiçi kullanıma engel olan bir **dizin kontrol mekanizması** var.

Dosyaya baktığımızda 33-37. satırlar arasında şu blok var:
```bash
# Proje dizininde olduğumuzu kontrol et
if [ ! -f "go.mod" ] || [ ! -d "cmd/main" ]; then
    echo -e "${RED}[-] Hata: Bu script MyBox proje dizininden çalıştırılmalıdır.${NC}"
    exit 1
fi
```

### Sorun Nedir?
Kullanıcı `curl -sSL .../install.sh | sudo bash` komutunu çalıştırdığında, işlemi kendi klasöründe (örneğin bilgisayarındaki boş bir `Desktop/example` klasöründe) yapıyor olur. Script anında "Burada go.mod yok ki, iptal ediyorum" diyerek kapanır. Çünkü script, *halihazırda projenin tüm dosyalarına sahip olunduğunu varsayarak* yazılmış.

### Çözüm Yolu
Eğer bu `install.sh` dosyasını `curl | bash` mantığıyla çalışacak şekilde (tek satırda indirme) sunmak istiyorsak, scriptin içini internetten kuruluma uygun hale getirmeliyiz. Düzeltilmiş script arka planda şunları yapmalı:

1. Dizin kontrolü **yapmamalı**.
2. Hedef sisteme boş bir geçici (temp) klasör açmalı.
3. `git clone https://github.com/karahasanoglu/MyBox-Project.git -b version-1.0.6` komutuyla projenizi indirmeli.
4. İndirdiği klasörün içine girip (`cd MyBox-Project`) orada `go mod tidy` ve `go build -o /usr/local/bin/mybox` işlemlerini yapmalı.
5. İşlem bitince indirdiği geçici klasörü temizlemeli.

Eğer isterseniz `install.sh` dosyasını bu bahsettiğim modern "internet üstünden sorunsuz kurulabilen" formata güncelleyebilirim. Böylece arayüzden verilen o tek satırlık kodu alan herhangi bir kullanıcı saniyeler içinde kurulum yapabilir. İster misiniz güncelleyeyim mi?
