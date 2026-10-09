# MCPDeck desktop

MCPDeck'in yerel macOS arayüzü. Mevcut Go iş mantığını Wails üzerinden kullanır;
React arayüzü uygulamanın içine gömülür. Normal kullanımda tarayıcı sunucusu, Node
veya ayrı bir MCPDeck terminal uygulaması gerekmez.

## Açılış ve kullanım

Yerel paket: `desktop/build/bin/MCPDeck.app`.
Finder'dan çift tıklayın veya terminalden `open desktop/build/bin/MCPDeck.app` kullanın.
Uygulama mevcut `~/.config/mcpdeck/deck.json` dosyasını kullanır. Başka bir deck için
uygulama sürecine `MCPDECK_CONFIG=/tam/yol/deck.json` ortam değişkeni verilebilir.

- **MCP Servers:** yönetilen ve ajan ayarlarında bulunan sunucuları gösterir.
  Agent seçicisinden bir ajan seçerek yalnız o ajanda açın/kapatın. “All configured
  agents” bütün profillere uygular. “Remove everywhere” bütün ilgili ajanlardan
  kaldırır; kurulu programları silmez. Her işlem önce hedefleri gösterir.
- **Agents:** kaydedilen Direct/Bridge modu ve son senkronizasyon sonucunu gösterir.
  Sync, Use Direct/Bridge, Restore backup işlemleri inceleme sonrasında uygulanır.
  Başarısız sonuçlar için Review retry yalnız başarısız hedefleri yeniden dener.
  “Detected” ve “synced” bir MCP bağlantısının çalıştığı anlamına gelmez.
- **Instructions:** ortak talimatları değiştirin veya tek ajanın kişisel metnini
  düzenleyin. Önce Review changes, sonra Apply. Kişisel metin ve ortak blok ayrılır;
  boş ortak metin ortak blokların kaldırılmasını açıkça gösterir.
- **New MCP:** isteği yazın, planlayıcının tarifini inceleyin, gerekli özel değerleri
  ayrı gizli alanlara girin ve kurulumu onaylayın. Komutlar kullanıcı yetkisiyle çalışır.
  Bu ekran sağlayıcı hesabı kullanabilir; normal giriş ve gerekli runtime'lar hazır olmalı.
- **Settings:** CLI veya API planlayıcısını seçin. API anahtarları ortam değişkeni adıyla
  referans edilir. macOS açılışında mevcut PATH korunarak Homebrew ve bilinen kullanıcı bin dizinleri
  eklenir. Shell başlangıç dosyaları çalıştırılmaz; özel runtime yolları ve API anahtarı
  değişkenleri ayrıca uygulama ortamında bulunmalıdır.

Pencereyi kapatırken kaydedilmemiş düzenlemeler veya süren kurulum varsa uyarı gösterilir.
İptal, tamamlanmış kurulum adımlarını geri almaz. Geri yükleme, son ajan dosyası yedeğini
uygular; deck içindeki seçim/mod değişmez. Sonraki Sync bu kayıtlı seçimi yeniden uygular.
İncelemeden sonra hedef dosya veya yedek değişirse geri yükleme reddedilir.

Bridge ayarları aynı uygulamanın executable yolunu kullanır. Uygulama `bridge` argümanını
aldığında pencere açmadan MCP stdio sunucusu çalıştırır. `.app` dosyasını başka dizine
taşırsanız ajanları tekrar Sync edin.

## macOS paketi oluşturma

Go 1.25+, Node.js, Xcode Command Line Tools gerekir. İlk kez:

```sh
cd desktop/frontend
npm ci
cd ../..
sh desktop/scripts/build-macos.sh
```

Node farklı konumdaysa `MCPDECK_NODE=/tam/yol/node` kullanın. Script frontend testlerini
çalıştırır, frontend'i derler, Go uygulamasını gömülü varlıklarla derler, `.app` yapısını
ve ikonlarını oluşturur ve yerel ad-hoc imza uygular. Bu Apple notarization değildir.
Windows/Linux için bu macOS paketleme scripti kullanılmaz; Wails platform araçları gerekir.
Bu çalışma kapsamında Windows/Linux masaüstü paketleri doğrulanmamıştır.

## Geliştirici önizlemesi

```sh
cd desktop
# Bir terminalde:
go run ./preview --port 4184 --frontend-port 4183
# Başka terminalde, desktop/frontend altında:
MCPDECK_PREVIEW_PORT=4184 npm run dev -- --port 4183 --strictPort
```

Önizleme yalnız loopback'te çalışır, gerçek kullanıcı ayarlarını kullanmaz ve sağlayıcı
hesaplarına bağlanmaz. Örnek git girdisi gerçek MCP bağlantı testi için değildir.
Native örnek modu için paket içindeki executable'ı `MCPDECK_CLIENT_DEMO=1` ile çalıştırın.
Otomatik ajan keşfi örnek modunda kapalıdır; örnek işlemler geçici dizinle sınırlıdır.
Native moddaki sağlayıcı kurulumu tarayıcı örneğinde kullanılmaz.

## Testler

```sh
# Proje kökünde:
go test ./...
go test -race ./internal/client ./internal/store ./internal/syncer
go vet ./...
# desktop/frontend altında:
npm test
# desktop altında; önce paketi oluşturun:
MCPDECK_TEST_BINARY="$PWD/build/bin/MCPDeck.app/Contents/MacOS/MCPDeck" go test -tags production ./...
```

Paket testi executable'ın gerçek `initialize` ve `tools/list` yanıtlarını kontrol eder;
sadece pencereyi açabilmek Bridge doğrulaması sayılmaz. Yeni ajan seçimi, kısmi sync
sonucu, retry, stale restore ve demo izolasyonu Go regresyon testleriyle kapsanır.

## Küçük Mac ve Windows paketleri

Her işlemci için ayrı paket oluşturulur; evrensel binary veya tam Chromium dağıtılmaz.
Mac için sistem WebKit, Windows için ortak Microsoft WebView2 kullanılır. Windows
paketi yalnız küçük resmi WebView2 kurucusunu içerir; çalışma zamanı eksikse uygulama
kurulumu teklif eder. Çalışma zamanının ek indirme/disk alanı uygulama boyutuna dahil
edilmez. Go ve Node yalnız geliştirme/derleme araçlarıdır.

- **Mac:** `MCPDeck-macos-arm64.zip` (Apple Silicon) veya `MCPDeck-macos-amd64.zip`
  (Intel). ZIP'i açın, MCPDeck.app dosyasını Applications'a veya ~/Applications'a taşıyın.
- **Windows:** `MCPDeck-windows-amd64.zip` (Intel/AMD x64) veya
  `MCPDeck-windows-arm64.zip`. ZIP'i açıp Install.cmd dosyasına çift tıklayın.
  Kullanıcı hesabına kurulur; Start menüsü kısayolu ve uygulama kaldırma kaydı eklenir.
  Taşınabilir kullanım için MCPDeck.exe dosyasını doğrudan çalıştırın.
- Windows güncellemesinden önce MCPDeck ve bu executable'ı kullanan Bridge oturumlarını
  kapatın. Kurucu tek uygulama dosyasını günceller; her sürüm için ekstra kopya biriktirmez.
  Kaldırma, ortak ayarları veya kurulmuş MCP sunucularını silmez.

Paketleme uygulama için **35 MiB**, ZIP için **20 MiB** sınırı uygular. Boyut aşılırsa
paket üretimi hata verir. `*.size.json` dosyaları gerçek binary, uygulama ve arşiv
boyutlarını kaydeder. Her arşivin ayrı SHA-256 dosyası vardır. Kaynak kod,
node_modules, npm/uv önbellekleri, test verileri ve kullanıcı yapılandırmaları pakete girmez.
MCP sunucularının kendi runtime ve veri dosyaları ayrıca yer kaplar.

Windows oluşturma (yerel Windows / PowerShell):

```powershell
./desktop/scripts/build-windows.ps1 -Architecture amd64
./desktop/scripts/build-windows.ps1 -Architecture arm64
```

Mac Intel derlemesi için `MCPDECK_ARCH=amd64` ve ayrı `MCPDECK_APP_DIR` belirtilebilir.
Paket oluşturma, desktop klasöründe `go run ./package -platform darwin -arch arm64
-source build/bin/MCPDeck.app` şeklindedir. Paketlerin GitHub Actions akışı
`.github/workflows/desktop.yml` içindedir; iş akışı eklenmesi başarılı CI çalışması
veya public yayın yapıldığı anlamına gelmez.

Mac yerel paket ve Bridge testi burada doğrulandı. Windows x64/ARM64 ve Intel Mac
binary'leri bu Mac üzerinde cross compile edildi; gerçek Windows kurulum/kaldırma,
WebView2 ve pencere akışları Windows'ta ayrıca doğrulanmalıdır. Paketler şu anda
Apple notarization / Windows Authenticode imzası içermez.

Kaynaklar: [Wails Windows runtime stratejileri](https://wails.io/docs/guides/windows/),
[Microsoft WebView2 dağıtımı](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution).

## Ajanın mevcut talimatları

Agents ekranındaki Instructions düğmesi seçili ajanın global talimat dosyasını açar.
Current instructions, o dosyada kayıtlı kişisel metni ve ortak MCPDeck bölümünü birlikte
gösterir. Edit personal text yalnız kişisel metni düzenler; ortak bölüm korunur.
Reload saved file dışarıdan yapılmış güncellemeyi tekrar okur. Bu görünüm proje
klasörünün kurallarını veya ajan oturumundaki bütün talimat kaynaklarını toplamaz.
Sample workspace etiketi varsa görünenler izole örnek dosyalardır.

## En kolay kurulum

ZIP çıkarma gerektirmeyen kurucular:

- **Mac Apple Silicon:** MCPDeck-Setup-macos-arm64.pkg dosyasına çift tıklayın,
  sistem kurulum ekranında devam edin. Uygulama /Applications/MCPDeck.app konumuna kurulur.
- **Mac Intel:** aynı işlem için MCPDeck-Setup-macos-amd64.pkg kullanın.
- **Windows x64:** MCPDeck-Setup-amd64.exe dosyasına çift tıklayın ve kurulumu onaylayın.
  Uygulama kullanıcı hesabına kurulur, Start menüsü kısayolu eklenir ve açılır.
- **Windows ARM64:** MCPDeck-Setup-arm64.exe kullanın.

Windows kurucusu kendi doğrulanan uygulama paketini taşır; ZIP'i kullanıcıya açtırmaz,
geçici dosyaları işlem sonunda temizler. Bütünlük ve mimari kontrolünden sonra mevcut
per-user kurulum scriptini çalıştırır. Kurulu uygulama güncellemesinden önce onu ve
Bridge oturumlarını kapatın. Mac PKG, sistem Applications konumuna kurduğu için sistem
kurucusu yönetici izni isteyebilir; yönetici izni olmadan ZIP'teki uygulamayı
~/Applications dizinine taşıyabilirsiniz. PKG özel postinstall scripti çalıştırmaz.

PKG üretimi: proje kökünde `sh desktop/scripts/package-pkg.sh`. Intel için
MCPDECK_ARCH ve MCPDECK_APP_DIR değişkenlerini belirtin. Windows build scripti artık
Setup EXE'sini de üretir. Mevcut ZIP'ten ayrıca oluşturmak için
`desktop/scripts/build-setup.ps1 -Architecture amd64` kullanılabilir.
DMG üretim scripti de mevcuttur, ancak bu yerel ortamda hdiutil disk aygıtına erişemedi;
bu nedenle burada doğrulanan Mac kurucusu PKG'dir.

Kurucu dosyaları 20 MiB sınırını aşarsa paketleme durur. Windows Setup payload'ı
checksum, dosya yolları ve mimari açısından test edilir. Gerçek Windows GUI kurulumu
bu Mac'te çalıştırılmış değildir. Genel yayın öncesinde Apple Developer ID/notarization
ve Windows Authenticode imzaları ayrıca tamamlanmalıdır.

## Ajan bazında dosya seçimi

Instructions → Agent ile ajanı seçin. Instruction file listesinde bulunan dosyalar,
boş dosyalar ve henüz oluşturulmamış standart hedefler ayırt edilir. Aynı dosyayı
kullanan Gemini CLI/Antigravity veya Copilot varyantları tek dosya kaydında görünür.
Paylaşılan dosya düzenlemesinin inceleme ekranı etkilenen bütün ajanları gösterir.
Reload sources dışarıdan eklenen global dosyaları yeniden tarar. VS Code Local
profil kuralları, Agent Host / CLI ortak dosyalarından ayrı görünür.
