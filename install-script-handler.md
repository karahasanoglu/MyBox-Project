## İndirme (Download / Install) Tarafının Çalışma Mantığı

Kullanıcıların MyBox'ı (CLI ve Desktop) bu arayüzden indirebilmesi için 2 farklı yaklaşım sunmamız gerekir:

### 1. Terminal (CLI) Kurulumu (`curl | bash` mantığı)
Arayüzde gösterdiğimiz kısımda kullanıcı şu kodu kopyalayacak:
`curl -sSL https://raw.githubusercontent.com/.../install.sh | sudo bash`

Bunun sorunsuz çalışması için:
*   `install.sh` adında bir bash scriptini açık bir URL'de (GitHub Raw veya kendi sunucunuz) tutmanız gerekir.
*   Kullanıcı bu kodu terminaline yapıştırdığında script internetten çekilir.
*   Script içinde: 
    - Gerekli bağımlılıkları yükler (apt install ile)
    - Projenizin derlenmiş "binary" dosyasını internetten çeker (veya `go install` ile direkt GitHub deposundan derler)
    - Sistemi yapılandırır (`/usr/local/bin` altına atar vb.).

Projenizde halihazırda bir `install.sh` var. Tek yapmanız gereken bu `install.sh` dosyasını (ve gerekiyorsa projeyi) GitHub'a yükleyip o sayfanın Raw linkini HTML içindeki kopyalama yerine yerleştirmek.

### 2. Masaüstü App (Wails - Desktop UI) İndirme Butonu
Wails ile yaptığımız arayüz için kullanıcıların bilgisayarlarına bir dosya indirip çift tıklayarak yönetecekleri bir "Çalıştırılabilir Dosya" sunmalısınız. Linux için bu genellikle `.AppImage` veya `.deb` dosyasıdır.

**Nasıl Yapılır:**
1.  **Derleme (Release):** MyBox masaüstü uygulamasının son halini (daha önce uğraştığımız `wails build` işlemi ile) sorunsuz bir şekilde derlediğinizde ortaya çıkan o dosyayı (binary veya `.deb` paketini) GitHub Releases kısmına yüklersiniz.
2.  **HTML Güncellemesi:** Tasarladığımız `index.html` sayfasındaki `<a href="#" class="btn btn-primary btn-large download-app-btn">` yazan kısımdaki `href="#"` alanını, yüklediğiniz yükleme dosyasının (örneğin `https://github.com/KULLANICI/mybox/releases/download/v1.0/mybox-ui.AppImage`) direkt indirme linki ile değiştirirsiniz.

Böylece kullanıcı:
*   "Download AppImage" butonuna basınca tarayıcısı üzerinden dosya inmeye başlar.
*   "CLI Copy" butonuna basıp terminale atarsa da `install.sh` scriptiniz arka planda sistemi kurar.

**Şu an Ne Yapabiliriz?**
Eğer projenizi GitHub'a koyduysanız, indirme linkini HTML arayüzümüze hemen entegre edebilirim!
