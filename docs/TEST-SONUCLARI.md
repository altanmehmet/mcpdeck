# Test sonuçları

Historical pre-launch CI links refer to the private development archive. Current public
checks are available at https://github.com/altanmehmet/mcpdeck/actions.

## Public alpha hazırlığı ve yeni MCP testleri — 6 Ekim 2026

- Gitleaks 8.30.1 resmi binary checksum'u doğrulandı. Tam geliştirme geçmişinin
  53 commit'i sır taramasından geçti; sır bulgusu yok. Ayrı metin taraması,
  eski test raporundaki dahili bağlantı etiketini tespit etti. Kullanıcı eski
  geçmişin private arşivde kalmasını ve public reponun temiz snapshot ile
  başlamasını seçti; güncel raporda bağlantı etiketi anonimleştirildi.
- Doğal dil + gerçek Codex planlayıcısıyla resmi filesystem `2026.8.31`,
  Git `2026.8.18` ve Fetch `2026.8.18` MCP tarifleri üretildi. İncelenen planlar
  MCPDeck install komutuyla uygulandı; npm ve uv çalışma ortamları kullanıldı.
- Üç MCP gerçek makinedeki 17 tanımlı profilin MCPDeck bridge bağlantısında
  initialize/tools/list ile doğrulandı. Her profilde filesystem allowed directory
  listesi yalnız test sandbox'ını içerdi. Doğrudan istemci roots davranışını
  kapsayan erişim sınırı garantisi verilmez; test bridge topolojisi içindi.
- Gerçek araç çağrıları geçti: sandbox'taki rastgele dosyayı okuma, yerel Git
  durumu ve Fetch ile resmi public README'yi HTTP üzerinden okuma. İlk HTTP
  denemesi hata verdi; başka bir denemede eski example.com başlık beklentisi
  güncel içerikle eşleşmedi. Resmi README ile yapılan son çağrı başarılıydı;
  önceki başarısız denemeler başarılı sayılmadı.
- Gerçek Codex oturumu, gerçek config okuyucusunun yüklediği bridge girdisiyle
  izole HOME'da yeni filesystem aracını çağırdı. Bir adet gerçek MCP tool event'i
  ve sorguda bulunmayan rastgele dosya değeri doğrulandı. Gerçek Codex ve Copilot
  CLI okuyucuları kurulan bridge girdisini gördü. Diğer istemcilerin model
  oturumlarında çağrı yapıldığı iddia edilmez.
- Örnek ortak güvenli talimatlar 10 gerçek global dosyaya dağıtıldı. Üç MCP'nin
  disable/enable/remove işlemleri ile araçların kaybolması/geri gelmesi ve
  talimat kaldırma geçti. Gerçek kullanıcı deck'i değişmedi. Ajan ayarları,
  talimatlar ve mevcut yedekler byte-byte ve özgün modlarıyla geri yüklendi;
  geçici MCP kurulumları/test deck'i silindi. Paket yöneticisi indirme cache'leri
  kalabilir; kullanıcı hesabı veya proje verisi silinmedi.
- CI terminal testi, metin render edilmeden inceleme ekranına geçiyordu.
  Test artık taslağı ve inceleme ekranını ayrı bekliyor; düzeltmeden sonraki
  macOS/Linux/Windows x64/native ARM64 ve güvenlik CI kontrolleri geçti.
- Kurulum tariflerinin mevcut shell engeli `.exe`/büyük harf alias'larını
  kabul ediyordu. Önce başarısız regresyonla doğrulandı; aynı engel canonical
  adıyla uygulanacak şekilde düzeltildi. Normal runtime/helper executable
  isimleri korunur. Bu filtre bir işletim sistemi sandbox'ı değildir.
- Yeni public repo ayrı bir Ed25519 yayın anahtarı kullanır. Eski alpha.2
  manifest imzası, iki public anahtarı içeren allowed_signers ile tekrar
  başarıyla doğrulandı; tarihi yayın varlıkları değiştirilmedi.

## Oracle sağlık sorgusu — 6 Ekim 2026

- Kullanıcının seçtiği `saved-test-connection` kayıtlı bağlantısında MCPDeck köprüsü
  üzerinden yalnız `SELECT 1 AS MCPDECK_HEALTH FROM DUAL` denendi. Kendi SQLcl
  oturumu açılıp kapatıldı; mevcut ajan ayarları değiştirilmedi.
- Gerçek yanıt `ORA-00406` (`COMPATIBLE` en az `12.0.0.0.0` olmalı),
  `ORA-00722` (`SQL identity columns`) ve `ORA-06512` içeriyordu. Beklenen
  sorgu sonucu alınmadı; Oracle veritabanı erişimi başarılı sayılmıyor.
- İlk smoke testi yalnız MCP `isError` alanına bakıyordu; SQLcl bu hatayı metin
  içeriğinde döndürdüğü için yanlış PASS yazdı. Test artık Oracle hata kodlarını
  ve sağlık sorgusundaki gerçek `1` değerini kontrol ediyor. İlk PASS geçersizdir.
- Altı yerel regresyon testi geçti: CSV/tablo başarı yanıtı, eksik/yanlış sonuç,
  MCP hata alanı ve birden fazla içerikte alan olmadan dönen Oracle hataları.
- Veritabanı parametreleri değiştirilmedi. DBA, sunucu sürümünü/uyumluluk
  seviyesini ve SQLcl MCP desteğini değerlendirmeli; bu kontrol bir yükseltme
  veya `COMPATIBLE` değişikliği için otomatik izin değildir.
- Oracle/Sentry initialize, 13 araç keşfi, kayıtlı Oracle bağlantılarının listesi
  ve mevcut Sentry oturumuyla `whoami` yeniden geçti. Yeni OAuth login/refresh
  akışı bu testin kapsamı dışında.

## Genel talimat testleri — 6 Ekim 2026

- Bu Mac'te `go test ./...`, `go vet ./...` ve talimat/terminal paketlerinin
  `go test -race ./internal/instructions ./internal/tui` kontrolleri geçti.
- Gerçek kullanıcı dizininde 10 farklı genel talimat dosyasında ekleme,
  güncelleme, aynı talimatı yeniden ekleme, sync, Codex kişisel metnini düzenleme
  ve ortak bölümü kaldırma geçti. Kullanıcı metni, MCP ayarları, deck ve mevcut
  proje talimatları korundu. İşlem sonunda genel dosyalar ve önceden var olan
  `.mcpdeck-backup` dosyaları byte-byte ve özgün izinleriyle geri yüklendi;
  testin oluşturduğu yeni dosyalar kaldırıldı. Test metni bırakılmadı.
- Ayrı geçici HOME altında 13 otomatik profil / 11 farklı ortak hedefle binary
  testi geçti. Unicode, mevcut/modüler dosyaları görme, ortak metni değiştirme,
  tek dosyayı düzenleme, kişisel/ortak bölüm ayrımı, tekrarlı ekleme/sync,
  onaysız kaldırmayı reddetme, boyut/NUL kontrolü, kısmi hata ve retry doğrulandı.
  Otomatik adapter'ı olmayan profillerin `manual` sonucu doğrulandı.
  Aynı CLI/PTY testi hem güncel kaynak binary'si hem terminaldeki Homebrew
  `0.1.0-alpha.2` komutuyla geçti.
- Gerçek Mac PTY'de fareyle inceleme/kaydetme, kaydetmeden önce dosyaya
  yazılmaması, klavyeyle tek talimat ekleme, Unicode, mevcut dosyayı fareyle
  açıp düzenleme, yalnız seçili dosyayı kaydetme, "Use for all" taslağını iptal,
  OSC 52 pano sinyali ve çıkış geçti. Pano sinyali, terminal dışındaki başka bir
  uygulamaya yapıştırmanın doğrulandığı anlamına gelmez.
- Gerçek Codex ve Copilot CLI hesaplarıyla her istemcide üç yeni izole model
  oturumu açıldı: rastgele talimat değeri ekleme ve güncellemede model yanıtında
  doğrulandı; kaldırmadan sonra `ABSENT` döndü. Değer kullanıcı sorgusunda
  verilmedi. Sadece test talimatları gönderildi, gerçek kullanıcı ayarları veya
  hesap sonuçları yazdırılmadı. Sağlayıcı hesap kullanımı oluştu.
- İlk PTY denemelerinde Esc/Alt ayrımı, kaydetme tamamlanmadan gönderilen tuş
  ve süreç çıkışını bekleme testindeki yarış; ilk Copilot çağrısında desteklenmeyen
  `--deny-tool=*` biçimi nedeniyle hata oldu. Test sürücüleri düzeltildi; yalnız
  son başarılı çalıştırmalar yukarıdaki sonuçlara dahildir.
- Claude, Gemini, Qwen, OpenCode, Cursor agent, Kiro ve Cline çalıştırılabilir
  istemcileri bu Mac'te bulunmadı. Dosya dağıtımı geçti; bu ajanların gerçek
  oturumlarında talimat yükleme veya davranış testi yapıldığı iddia edilmiyor.
  Windows talimat editörü için elle terminal kabul testi yapılmadı.

## Release hardening — 5 Ekim 2026

- Mac üzerinde `go test ./...` ve `go vet ./...` geçti. Kısıtlı ortamda yerel
  HTTP test sunucuları açılamadığı için tam testler izin verilen ortamda tekrar
  çalıştırıldı; ilk başarısız deneme başarılı sayılmadı.
- Gerçek Codex ve Copilot CLI ayar okuyucularıyla ekle/kapat/aç/kaldır geçti.
  İki istemcide de yeniden gerçek model oturumu açılıp iki adet sunucu tarafından
  kaydedilen rastgele `proof` araç yanıtı doğrulandı. Mevcut ajan ayarları korunarak
  ayrı geçici HOME kullanıldı.
- Gerçek Oracle SQLcl ve mevcut Sentry bağlantısı ayrı geçici MCPDeck köprüsünden
  başlatıldı: initialize, 13 araç keşfi, Oracle kayıtlı bağlantı listesi ve Sentry
  araç keşfi + authenticated `whoami` geçti. Hesap sonuçları/sırlar yazdırılmadı.
  Bu, mevcut Sentry oturumunun kullanımını doğrular; sıfırdan OAuth login akışını
  veya Oracle veritabanında SQL çalıştırmayı doğrulamaz. Oracle bağlantı seçimi
  kullanıcıdan bekleniyor. Tekrarlama: `tests/real_mcp_smoke.py`.
- Binary kurulum smoke testi ve gerçek Mac PTY form, fare/tuş, senkronizasyon,
  gizli veri, kaldırma ve yardım senaryoları geçti.
- Native Windows x64 ve ARM64 üzerinde tüm Go testleri, vet, güvenlik taraması,
  ZIP kurulumu, sekiz profil yaşam döngüsü, köprü yolu taşıma, kaldırma, PATH
  koruması ve junction reddi geçti: [test run 37336699986](https://github.com/altanmehmet/mcpdeck-private-archive/actions/runs/37336699986).
  ARM64 işi gerçek host ve Go mimarisini doğruladı. Job Object alt süreç kapatma,
  bağımsız npm shim fixture argümanları ve gerçek Unicode pano geri okuma geçti.
- Altı platformun private test paketleri üretildi; Windows x64 paketinin süreç,
  pano ve kurulum/kaldırma kontrolleri geçti:
  [packages run 37336695408](https://github.com/altanmehmet/mcpdeck-private-archive/actions/runs/37336695408).
  Branch test paketi imzalanmadı ve public alpha.2 değiştirilmedi.
- İlk denemelerde panoya metin aktarımı ve PowerShell modül araması hataları
  bulundu ve düzeltildi. PATH testi, kurucunun mevcut boş segment normalleştirmesi
  dikkate alınarak kalan kayıtların sırasını/içeriğini doğruluyor. Başarısız
  denemeler başarılı sayılmadı.
- Yerel Windows ARM64 cihazı ve Windows/Apple imza kimliği yok. Elle cihaz kabulü,
  Authenticode, notarizasyon ve bağımsız dış güvenlik denetimi tamamlanmadı.

## Canlı ajan ve imza doğrulaması — 5 Ekim 2026

- `0.1.0-alpha.1` paketleri ayrı public dağıtım deposunda yayınlandı; kaynak
  deposu private kaldı. GitHub'dan tekrar indirilen dört paketin imzası/checksum'u
  doğrulandı. Public dağıtım CI'sinde macOS ve Linux üzerinde gerçek binary ve
  installer çalıştırıldı: [run 37302353105](https://github.com/altanmehmet/homebrew-mcpdeck/actions/runs/37302353105).
- Bu Mac'te gerçek `brew tap`, `brew install` ve `brew test` geçti; kurulan sürüm
  `0.1.0-alpha.1`. Önceki yerel binary yedeklenip komut Homebrew binary'sine
  bağlandı; böylece Homebrew güncellemeleri aynı terminal komutuna yansır.
- Private kaynak deposunun dört platform snapshot + imzalama işi de geçti:
  [run 37301859611](https://github.com/altanmehmet/mcpdeck-private-archive/actions/runs/37301859611).

- Codex ve Copilot CLI ile ayrı geçici HOME altında gerçek model oturumları
  çalıştırıldı. MCPDeck'in eklediği yerel MCP'nin `proof` aracı, ekleme ve tekrar
  etkinleştirme sonrasında çağrıldı. Sunucunun ürettiği rastgele kanıt hem çağrı
  kaydında hem ajan yanıtında doğrulandı: her ajan için iki gerçek çağrı geçti.
- Devre dışı bırakma ve kaldırma sonrasında gerçek CLI ayar okuyucusunda sunucu
  bulunmadığı doğrulandı. Kullanıcının mevcut ajan ayarları değiştirilmedi.
- Codex testinde yalnız `proof` aracı açıkça onaylandı; Copilot testinde yalnız
  test MCP aracı kullanıma açıldı. Sağlayıcı hesabı kullanımı gerçekleşti.
- İlk denemelerde Codex araç onayı ve izole Copilot oturumu eksikti. Testin izin
  ve oturum kurulumu düzeltildikten sonraki denemeler başarılı oldu. Ürün genel
  ajan izinlerini otomatik açmaz; bu izin yalnız geçici test oturumuna aittir.
- Claude/Gemini kurulu değil; diğer ajanlarda ve OAuth gerektiren uzak MCP'lerde
  canlı test yapılmadı. Bu sonuç Oracle veritabanı erişimini doğrulamaz.
- Dört platform arşivi için Ed25519 manifest imzası ve checksum doğrulaması geçti.
  Geçerli imza kabulü, değiştirilmiş arşiv/manifest, yanlış anahtar ve geçersiz
  dosya yolu reddi testleri geçti. macOS arm64 arşivinin kurulum testi geçti.
- Paketler Go ve bağlı modüllerin lisans metinlerini içeriyor. Apple Developer ID
  kimliği bulunmadığı için Apple imzası/notarizasyon yapılmadı. Diğer mimarilerde
  çapraz derleme, o makinede çalıştırılmış test anlamına gelmez.

## Güncel ek kontrol — 5 Ekim 2026

- Ajan dosyalarında MCPDeck yazar kilidi, dış düzenleme kontrolü, sembolik
  bağlantı korunması ve değişiklik olmayan sync'te yedeği koruma testleri geçti.
- Kısmi sync, yalnız başarısız hedefleri yeniden deneme, hedef yolu değişikliğini
  reddetme, sır içermeyen 0600 durum raporu, geri yükleme/geri alma ve CLI onay
  testleri geçti. Gerçek kullanıcı HOME yerine geçici dizin kullanıldı.
- Bu değişikliklerin PR #5 ve #6 macOS/Linux CI kontrolleri geçti: Go testleri,
  vet, derleme ve yerel MCP ile binary kurulum smoke testi.
- `govulncheck` v1.8.0, Go 1.27.1 ile macOS kaynak kodunda erişilen açık bulmadı.
  Modül seviyesinde GO-2026-5024 (x/sys Windows) kaydı var; taranan macOS kodunda
  çağrılmıyor. Bu sonuç Windows güvenliği veya kapsamlı güvenlik denetimi değildir.
- Gerçek sağlayıcı hesapları/OAuth ve tüm canlı ajan oturumları bu ek kontrolde
  yeniden denenmedi. Aşağıdaki eski sonuçlar kendi tarihleri için geçerlidir.
- Gerçek Codex ve Copilot CLI config okuyucuları, geçici HOME altında MCPDeck'in
  eklediği, kapattığı, tekrar açtığı ve kaldırdığı MCP'yi doğru gördü.
  `tests/client_config_smoke.py` model/MCP araç çağrısı veya hesap girişi yapmaz.

macOS arm64 üzerinde mevcut binary çalıştırıldı; aşağıdaki önbellek hatası
düzeltildikten sonra kaynak koddan yeni binary üretildi. Proje kökündeki
`mcpdeck`, binary senaryolarını geçen dosyayla birebir karşılaştırılarak doğrulandı.

## Düzeltilen hata

Bridge, geçerli JSON içindeki bozuk araç adlarını önbellekten kabul ediyor ve
`tools/list` çağrısını hata ile bitiriyordu. Artık eksik, boş, yanlış türde,
yinelenen veya fazla uzun araç adları içeren önbellek yeniden keşfediliyor.
Yeniden keşifte eski sayfalama imleci de temizleniyor. Bu durumlar için
`TestCorruptCacheRediscovered` regresyon testi eklendi.

## Başarılı kontroller

- `go test -count=1 ./...`
- `go test -race -count=1 ./...`
- `go vet ./...`
- Kaynak koddan binary derleme.
- Binary: sunucu ekleme, etkileşimli ekleme, yinelenen/geçersiz adları reddetme.
- Binary: varsayılan profilleri mevcut ayarları koruyarak, tekrar güvenle ekleme.
- Binary: mcpServers, VS Code, Copilot CLI ve Codex biçimlerinde Direct → Bridge
  → Direct senkronizasyonu; harici ayarları koruma.
- Binary: bozuk ayar dosyaları ve eksik ortam değişkenlerinde hedefi koruma.
- Binary: MCP el sıkışması, araç listesindeki bütün sayfaları alma, isim alanı
  yönlendirmesi, kapalı sunucuya çağrıyı reddetme, süreç yeniden kullanımı.
- Binary: çalıştırıcı mevcut değilken dolu önbellekten liste döndürme,
  önbellek temizleme ve bozuk önbelleği yeniden oluşturma.
- Binary: çağrı zaman aşımından sonra yeni çağrının çalışması.
- Binary: 100 ms boşluk ayarıyla, 30 saniyelik reaper kontrolünden sonra
  alt sürecin kapanması ve sonraki çağrıda yeniden başlaması.
- Terminal paneli: Space, b, Tab, s ve q; kaydedilen dosyaları kontrol etme.
- Geçici profil üzerinde `doctor` kontrolü.
- Gerçek `@modelcontextprotocol/server-filesystem` 2026.8.31 ile Bridge üzerinden
  araç keşfi, geçici klasörde dosya yazma/okuma ve klasör dışı erişimin reddi.

Kaynak testleri Go 1.24.7 ile çalıştırıldı. Go ve indirilen test bağımlılıkları
geçici dizinlerde tutuldu; makinenin PATH ayarı değiştirilmedi. Gerçek kullanıcı
IDE ayarları yerine geçici profiller kullanıldı.

## Doğrulanmayanlar

Canlı IDE oturumları, GitHub/Brave hesap yetkileri, gerçek PostgreSQL/SQLite
bağlantıları ve Windows/Linux çalışma davranışı bu çalıştırmada denenmedi.
Dosya biçimi testleri, ilgili IDE'nin canlı bağlantısının doğrulandığı anlamına
gelmez. README'deki HTTP, resources/prompts ve diğer kapsam sınırları devam eder.

## Ek doğrulama: gerçek Codex MCP istemcisi

22 Eylül 2026 tarihinde, kurulu Codex uygulamasının `codex-cli 0.153.4`
bağlantı motorunda canlı test yapıldı. mcpdeck paneli gerçek
`~/.codex/config.toml` dosyasına `mcpdeck-selftest` sunucusunu ekledi.
Son doğrulamada MCP ayarları komut satırından geçersiz kılınmadı;
Codex doğrudan bu ayar dosyasını kullandı.

| mcpdeck panelindeki işlem | Codex üzerinden alınan sonuç |
|---|---|
| Etkinleştir | Sunucunun 14 aracı keşfedildi; `read_text_file` test dosyasını okudu. |
| Devre dışı bırak | Sunucu listede bulunmadı; aynı çağrı `unknown MCP server 'mcpdeck-selftest'` hatasıyla reddedildi. |
| Yeniden etkinleştir | Sunucu ve 14 aracı tekrar bulundu; test dosyası yeniden okundu. |

Çağrılar Codex'in `mcpServerStatus/list` ve `mcpServer/tool/call`
arayüzleriyle, her durum için yeni geçici bağlantı oturumunda yapıldı.
Model çalıştırılmadı. Bu doğrulama, açık masaüstü sohbetinin araç listesinin
anında yenilendiğini kanıtlamaz; mevcut bağlantıların yenilenmesi gereği sürer.

Test sunucusu etkin bırakıldı. Erişimi yalnızca
`.mcpdeck-live-test/files` klasörüyle sınırlıdır. Tekrarlama betiği
`.mcpdeck-live-test/probe.py`, panel yapılandırması
`.mcpdeck-live-test/deck.json` konumundadır. Sunucu paketi `/tmp` altında
olduğu için geçici dosyalar temizlenirse test kurulumu yeniden hazırlanmalıdır.

## Ek doğrulama: otomatik dağıtım ve toplu etkinleştirme

Yeni `add` varsayılanı, sunucuyu tüm kayıtlı profillerde etkinleştirip otomatik
senkronize eder. `enable`/`disable` ve panelde e/d tüm profilleri günceller;
`--profile` ve Space ile tek-profil yönetimi korunur.

- Sekiz varsayılan profilin tamamı için otomatik ekleme ve ayar üretimi geçti.
- Toplu aç/kapat/aç ve aynı komutun tekrar çalıştırılması geçti; yinelenen
  sunucu seçimleri oluşmadı.
- Ortam değişkeni girişi ve referansların çözülmesi; sihirbazda boş profilin
  bütün profillere uygulanması geçti.
- Tek-profil seçimi, `--disabled`, `--no-sync` ve Bridge profilinin modunu
  koruma testleri geçti.
- Bozuk bir hedef dosyada diğer profillerin güncellenmesi, hatanın
  bildirilmesi ve düzeltme sonrası yeniden deneme geçti.
- `go test -count=1 ./...`, `go test -race -count=1 ./...` ve `go vet ./...` geçti.
- Yeni binary üzerinde 11 uçtan uca senaryo geçti; bunlara gerçek filesystem
  sunucusu, sekiz profil ve Bridge önbellek/zaman aşımı/boşta kapanma dahildir.
- Gerçek terminal panelinde e/d ile iki geçici profildeki seçimler doğrulandı.

Bu aşama mevcut stdio sunucu modelini kullanıyordu. Ajan ayarlarına yazmak,
çalışan ajan bağlantısını otomatik yeniden başlatmaz.

## Ek doğrulama: uzak MCP
- HTTP ve SSE için gerçek `mcp-remote@0.1.38` adaptörü, localhost test sunucusunda
  `initialize`, `tools/list` ve `tools/call` akışlarını tamamladı. Test başlığı
  sunucuya ulaştı. Bu test `MCPDECK_REMOTE_TEST_DIR` ayarıyla ayrıca çalıştırıldı.
- Uzak URL/başlık doğrulama, başlık enjeksiyonu, eksik ortam referansı, gizli
  değerlerin `status` çıktısında bulunmaması ve sekiz ajan biçimi testleri geçti.
- `go test -count=1 ./...`, geliştirme sırasında `go test -race -count=1 ./...`,
  son sürümde `go vet ./...` ve güncel binary ile 11 uçtan uca test geçti.

Sentry/Figma hesabında OAuth tamamlanmadı;
bu servislerden gerçek hesap verisi sorgulanmış sayılmaz. Sağlayıcının izin verdiği
istemciler ve ajanın yetkilendirmesi ayrıca gereklidir.

## Ek doğrulama: genel MCP içe aktarma

- JSON `mcpServers`, `servers`, `mcp_servers`, isim eşlemesi, tek nesne,
  Codex TOML, Markdown kod bloğu ve çıplak MCP URL ayrıştırması test edildi.
- Çoklu ekleme, kaynakta kapalı sunucuyu kapalı tutma, çakışan adlarda bütün
  eklemeyi reddetme ve önizlemenin ayarları değiştirmemesi doğrulandı.
- Eksik `${env:...}` ve `${input:...}` alanları tanındı; girilen `$` içeren
  değerlerin bozulmadan aktarılması ve önizlemede gizlenmesi test edildi.
- CLI üzerinden yerel ve uzak MCP birlikte eklendi. Eksik token alanı çıktı; test
  değeri girilip eklenince iki MCP test profillerine yazıldı. Kaynakta desteklenmeyen
  özel alanlar sessizce silinmeden reddedildi.

## 25 Eylül 2026: fare destekli CLI ve kolay kurulum

Ana kullanım biçimi terminal paneli olarak düzenlendi. Herdr’ın fare/klavye birlikte
kullanımı örnek alındı; ajan sütunu, MCP listesi, geniş ekranda ayrıntı sütunu ve
tıklanabilir işlem düğmeleri eklendi.

- `go test ./...`, `go test -race ./internal/tui ./cmd` ve `go vet ./...` geçti.
- Mouse press/release ayrımı, tek/tüm ajan seçimi, kaydırma, yeniden boyutlandırma,
  küçük terminalde gizli işlemlerin engellenmesi ve gizli alan maskeleme test edildi.
- `python3 tests/tui_smoke.py /tam/yol/mcpdeck` gerçek PTY oturumunda geçti:
  SGR fare olaylarıyla ekleme formu açıldı; iki MCP yapıştırıldı; eksik token
  girildi ve terminale yansıtılmadığı doğrulandı; sekiz profil dosyası kontrol edildi;
  tek/tüm ajan kapatma, yardım ve çıkış akışları tamamlandı.
- macOS ARM64 arşivinin SHA-256 değeri doğrulandı. Arşiv açılıp boşluk içeren
  geçici bir dizine kuruldu; kurulan binary üzerinde aynı PTY testi geçti.
  İkinci kurulumla güncelleme ve binary bütünlüğü doğrulandı.
- Kurulum testi kullanıcı HOME ayarlarını veya gerçek ajan yapılandırmalarını değiştirmedi.
- Terminal uygulamasına görsel erişim bilgisayar kullanım aracı tarafından engellendi.
  Yerel Terminal ekran görüntüsü doğrulaması yapılmadı; PTY testi bundan bağımsız geçti.

Paket: `dist/mcpdeck-darwin-arm64.tar.gz`. CLI kullanıcı tarafında Go veya
veritabanı gerektirmez. Ajanların MCP bağlantılarını yenileme ve servis hesaplarını
yetkilendirme gereği devam eder. Henüz genel bir indirme/otomatik güncelleme servisi yayınlanmadı.

## 25 Eylül 2026 — dokümantasyondan kurulum akışı

- `go test ./...`: geçti.
- `go test -race ./cmd ./internal/install ./internal/process ./internal/tui`: geçti.
- `go vet ./...`: geçti.
- Yeni derlenmiş binary ile `python3 tests/install_smoke.py ./mcpdeck`: geçti.
  Plan onayı, yerel kurulum adımı, gerçek MCP initialize/tools-list bağlantısı,
  sekiz ajan config biçimi, yedekler, disable/enable ve sırların çıktıya sızmaması kontrol edildi.
- `python3 tests/tui_smoke.py ./mcpdeck`: fareli mevcut akış geçti.
- Mac ARM64 paketi yeniden üretildi; SHA-256 kontrolü geçti.
- Codex planlayıcı testi yerel fixture kullanır; canlı model hesabına istek
  yapılmadı. Oracle/Figma/Sentry hesaplarıyla canlı kurulum doğrulanmadı.
- Testler geçici dizinler kullandı; kullanıcının gerçek ajan ayarları değiştirilmedi.

## 25 Eylül 2026 — birden çok planlayıcı sağlayıcısı

- Codex, Claude Code, Gemini CLI ve OpenAI/Anthropic/Gemini/OpenAI uyumlu API adaptörleri eklendi.
- `go test ./...`: geçti. Yerel HTTP yönlendirme testi için sandbox dışında
  geçici test portu açma izni kullanıldı; gerçek sağlayıcılara istek gönderilmedi.
- Son değişikliklerle `go test -race ./cmd ./internal/install`: geçti.
- `go vet ./...`: geçti.
- API istek/yanıt biçimleri, kimlik doğrulama başlıkları, hata/red yanıtları,
  yönlendirmelerde anahtarın taşınmaması ve ayar dosyasında anahtar saklanmaması test edildi.
- CLI protokolleri fixture programlarıyla, kayıtlı sağlayıcı ve dokümantasyon
  dosyasının install komutuna aktarımıyla birlikte test edildi.
- Güncel binary ile install ve fareli TUI smoke testleri geçti.
- Güncel Mac ARM64 paketi üretildi ve SHA-256 doğrulandı.
- Canlı model çağrısı yapılmadı; kurulu Claude/Gemini CLI ve gerçek API hesaplarıyla
  uçtan uca doğrulama henüz yapılmadı.

## 28 Eylül 2026 — gerçek Codex ile MCP kurulumu

- Codex CLI 0.155.0-alpha.16.4 ile resmi MCP filesystem dokümantasyonundan
  plan üretildi. İlk gerçek plan denemesi, requirements açıklama cümleleri ve
  connection.command içindeki desteklenmeyen değişken yüzünden uygulanabilir
  değildi. Plan şeması/istem bu hataları yakalayıp önleyecek şekilde güncellendi.
- Güncel binary ile ikinci Codex planı üretildi ve dosyadan tekrar başarıyla
  doğrulandı. Adımlar, pinned paket, bağlantı komutu ve kullanıcı girdisi ayrıştırıldı.
- Plan yalnız `/tmp/mcpdeck-codex-live-check` altında, tam plan özetiyle onaylandı.
  Resmi filesystem MCP başlatıldı, MCP initialize tamamlandı, `tools/list`
  sayfaları okundu ve 14 araç bulundu.
- Codex, Antigravity ve Copilot CLI biçimlerindeki üç test ayarı geçici dosyalara
  yazıldı. Gerçek ajan ayarları değiştirilmedi; MCP araçları çağrılmadı.
- Canlı Codex plan üretimi ve yerel MCP bağlantısı başarılı. Claude/Gemini/API
  sağlayıcılarıyla gerçek hesap testi yapılmadı.

## 3 Ekim 2026 — yerel ortam taraması ve kurulum sohbeti

- `mcpdeck environment` gerçek makinede Node/npm/npx, Python, uv/uvx, Java,
  Docker/Colima, Maven, Git ve Homebrew yollarını/sürümlerini sorguladı.
  Temurin ve Homebrew Java 21 dizinleri bulundu. SQLcl standart konumlarda
  algılanmadı. `.dbtools` için yalnız dizin varlığı kontrol edildi.
- Gerçek Codex sağlayıcısıyla Oracle SQLcl tarifi yeniden üretildi. Model mevcut
  Temurin Java 21.0.11'i ve mutlak JAVA_HOME yolunu kullandı; yeniden Java
  kurulumu ve gizli JAVA_HOME girdisi istemedi. SQLcl indirme/açma adımları ile
  sonradan yapılacak bağlantı ayarlarını ayırdı. Bu deneme yalnız plan üretti;
  SQLcl kurulumu, gerçek veritabanı girişi veya ajan ayarı değişikliği yapmadı.
- Gerçek terminalde sahte sağlayıcıyla soru → yeni tarif/plan kimliği → iptal
  akışı doğrulandı. Kurulum komutu çalışmadı; hedef ayar ve deck korunmuş kaldı.
- Kurulum sağlayıcı paketinin bütün testleri, yeni CLI sohbet testleri,
  sürüm sorgularında sır izolasyonu, eski planların onay kimliği uyumluluğu,
  ortam taramasının race testleri ve `go vet ./...` geçti.
- CLI yeniden derlendi ve kuruldu. Claude/Gemini/API sağlayıcılarının gerçek
  hesapla canlı sohbetleri bu çalıştırmada denenmedi.

### 3 October 2026 — removal and terminal review

- Added confirmed removal from MCPDeck and all configured agent profiles. A failed sync retains the disabled record so removal can be retried. Installed software and connection stores are kept.
- Interactive installation accepts `y`/`yes`; Enter, no, and incomplete input cancel. Noninteractive `--approve` still validates the complete reviewed plan digest.
- Added tests for removal, cancellation, partial sync retry, installation without digest copying, and wrapped terminal output. Focused CLI tests passed; the complete TUI and syncer suites and `go vet ./...` passed.
- Existing profile fixtures now put discovered agents inside temporary directories. An initial broad CLI run exposed existing machine-dependent fixture failures; this update does not claim the entire CLI suite passed.
- Rebuilt and installed the CLI. Desktop packaging is no longer part of the project.

### 3 October 2026 — externally configured MCP discovery

- Read-only discovery now lists external MCP names and their source profiles without importing or running them. Removal supports the existing profile formats, unknown connection options, safe backups, and rejection when reviewed settings change.
- All TUI and syncer tests, focused CLI removal/chat/approval tests, and `go vet ./...` passed. Tests exercised discovery and removal across default and additional profile formats while preserving other settings.
- Real machine read-only discovery found 7 external MCP names. No real MCP was removed. Updated the installed terminal command.

### 3 October 2026 — Copilot CLI compatibility and permission hardening

- Copilot CLI now respects `COPILOT_HOME/mcp-config.json` and migrates the old
  default path only when it was MCPDeck's automatic path; custom profile paths remain unchanged.
- On this machine, the real Copilot CLI 1.0.91 reports Oracle, Sentry and SonarQube
  with `copilot mcp list`. Oracle's MCPDeck-generated entry is `type: local` with
  `-mcp` and the verified Java 21 home; an already-open Copilot session still needs
  to be restarted to reload its configuration.
- Deck files with group/world-readable permissions are repaired to 0600 before use;
  if that repair fails, MCPDeck stops instead of using a potentially exposed deck.
- Store, syncer, TUI and command tests passed; `go vet ./...` passed. The CLI binary
  was rebuilt and reinstalled.

### Qwen Code ve TraeCode adapter’ları

- Qwen Code için `~/.qwen/settings.json`, `mcpServers` ve HTTP `httpUrl` biçimi eklendi.
- TraeCode için proje `.trae/mcp.json` ve `TRAE_MCP_CONFIG` override desteği eklendi.
- Yerel stdio, uzak HTTP, discovery, kaldırma ve ilgisiz ayarları koruma testleri eklendi.
- Store, syncer, TUI, model, bridge ve CLI testleri geçti; `go vet ./...` geçti.
- `go test ./... -count=1` ve yeni adapter’ların store/syncer/TUI race testleri geçti.
- Gerçek Qwen/Trae hesapları veya OAuth akışları bu ortamda çalıştırılmadı.

### 5 October 2026 — native Windows installation

- Native Windows Go tests and vet passed; the official vulnerability scanner
  found no reachable vulnerabilities. One transitive package/module finding
  remains outside the application's call graph; this does not claim the entire
  dependency tree has no advisories.
- Package workflow run `37319325637` successfully built all six archives and
  ran the Windows x64 archive/installer smoke test. Windows PowerShell 5.1,
  paths with spaces, repeat installation, actual User PATH persistence and the
  current terminal command were verified, with registry PATH restored afterward.
- A local Python MCP fixture completed initialize/tools-list. Approved installation
  then add/disable/enable/remove was verified across eight isolated profile
  formats, retaining unrelated entries and backups without printing input secrets.
- Windows ACL unit tests verified current-user/SYSTEM access and repaired broad
  deck access. Windows batch execution tests checked paths/arguments with spaces
  and rejected shell expansion inputs.
- Testing exposed a missing standalone upstream license file, PowerShell hash
  module loading and Windows short/long-path test assumptions. Packaging retains
  the exact upstream MIT README declaration; the installer uses .NET SHA-256.
- Windows ARM64 is cross compiled. Actual Windows agent accounts/configuration
  readers, interactive terminal mouse/clipboard, OAuth and Oracle database access
  were not exercised by these fixture tests. Executables are not Authenticode signed.
