# Turkish reference (historical usage details)

The public source repository is now available. Statements about private source below describe the pre-launch setup; use the main README for current installation.

# mcpdeck

Terminal MCP manager for macOS, Linux and Windows: configure supported coding agents,
install and repair MCP servers with reviewed plans, and share global instructions.
**Alpha software**, licensed under [MIT](../LICENSE). See [Contributing](../CONTRIBUTING.md)
and the [security policy](../SECURITY.md).

[Packages, signatures and release process](RELEASING.md).
Application source is public and MIT licensed. Binary alpha packages and the Homebrew tap
are distributed separately at [homebrew-mcpdeck](https://github.com/altanmehmet/homebrew-mcpdeck).
No stable release has been published.

### Install with Homebrew

```sh
brew tap altanmehmet/mcpdeck
brew install altanmehmet/mcpdeck/mcpdeck
mcpdeck
```

This installs the alpha binary without requiring Go or access to this repository.

### Install on Windows

Use the native Windows ZIP and its `install.ps1` installer in PowerShell.
No WSL, Go or administrator access is required. See the
[Windows installation guide](WINDOWS.md) for commands and test coverage.

### Build from source

Requires Go 1.24 or newer. The source repository is public.

```sh
git clone https://github.com/altanmehmet/mcpdeck.git
cd mcpdeck
go build -o mcpdeck .
sh scripts/install.sh
mcpdeck
```

**MCP ayarlarını tek yerden yöneten terminal aracı.**
Fare ve klavye destekli `mcpdeck` terminal paneli macOS, Linux ve Windows’ta çalışır.
Web servisi veya veritabanı kurulumu yoktur. İlk kullanım için
[başlangıç kılavuzuna](CLI-BASLANGIC.md) bakın.

**Genel talimatlar:** Panelde **Instructions** veya `g` ile kişisel talimatlarınızı
tek yerde düzenleyip desteklenen ajanlara dağıtabilirsiniz. Mevcut genel metin
korunur; proje kuralları düzenlenmez. Terminal karşılığı `mcpdeck instructions`.
[Hedefler, kullanım ve sınırlar](INSTRUCTIONS.md).

**Sync recovery:** `mcpdeck sync status` shows saved outcomes;
`mcpdeck sync --retry-failed` retries failed targets. The panel offers **Retry**
or **Ctrl+R**. [Backup restoration and concurrency limits](RECOVERY.md).

Cursor, Claude Desktop, Claude Code, Codex, GitHub Copilot (VS Code ve CLI),
Windsurf, Antigravity, Gemini CLI, OpenCode, Zed, Cline, Roo Code, Continue,
Amazon Q Developer, Kiro, Qwen Code ve TraeCode için MCP sunucu yönetimi. Desteklenen ek ajanlar yüklüyse ilk
okuma sırasında bulunup yapılandırmaya eklenir; Roo Code ve Cline her kurulu VS Code,
Cursor, Windsurf veya VSCodium kopyası için ayrı algılanır.
Direct Sync IDE ayarlarını üretir; Bridge, stdio araç sunucularını ihtiyaç olduğunda
başlatır ve boştayken kapatır.

## Derleme ve kullanım

### Dokümantasyondan kurulum

Panelde **Install MCP** düğmesi veya **i** tuşu, dokümantasyon/GitHub bağlantısını alır.
Seçtiğiniz Codex / Claude Code / Gemini CLI veya doğrudan API bir kurulum tarifi üretir. Tarifteki adımları
inceleyip `y` + Enter ile onayladıktan sonra eksik bilgiler gizli olarak sorulur;
kurulum yapılır ve gerçek MCP `initialize` + `tools/list` bağlantısı denenir.
Doğrulama başarılıysa tespit edilen destekli ajanların ayarları yedeklenip güncellenir.

Etkileşimli kurulumda plan gösterildikten sonra ajana soru sorabilir veya değişiklik
isteyebilirsiniz. Örneğin `Java zaten kurulu, onu kullan` ya da `Bu adım neden gerekli?`
yazın. `/refresh` yerel ortamı yeniden kontrol ederek tarifi günceller; Enter veya
`/install` komut onayına geçer, `/cancel` iptal eder. Manuel koşul sorusunda da
`DONE` yerine ajana soru yazabilirsiniz. Her güncellenmiş tarif yeniden onay ister.

Planlayıcı kurulu araçların yollarını ve sürümlerini, Java kurulum dizinlerini ve
Oracle bağlantı deposu dizininin bulunup bulunmadığını alır. MCPDeck bunun için
sabit sürüm sorguları kullanır; ajan ayarlarının veya bağlantı/parola dosyalarının
içeriğini okumaz. Algılanan yazılımları görmek için `mcpdeck environment` kullanın.
Standart olmayan kurulum konumlarını sohbet içinde belirtebilirsiniz.

```sh
./mcpdeck install
# Önce tarif üretip incelemek için:
./mcpdeck install 'MCP resmi dokümantasyon bağlantısı' --plan-only --out plan.json
./mcpdeck install --plan plan.json
# Varsayılan planlayıcıyı değiştir:
./mcpdeck planner set --provider claude
# Tek seferlik sağlayıcı seçimi:
./mcpdeck install 'MCP dokümantasyonu' --provider gemini --plan-only
# Kurulu bir MCP hata veriyorsa sağlık kontrolü + chat ile onarım:
./mcpdeck repair oracle-mcp "tools/list bağlantısı zaman aşımına uğruyor"
# `update` aynı komutun kısa adıdır.
./mcpdeck update oracle-mcp
```

API seçenekleri: `openai-api`, `anthropic-api`, `gemini-api`, `openai-compatible`.
API kullanımında `--model` zorunludur; anahtar ortam değişkeninden veya gizli
terminal girişinden alınır, ayar dosyasına kaydedilmez. `planner list` tüm
seçenekleri gösterir. OpenAI uyumlu servislerde web aracı yoktur; dokümantasyon
metni `--docs README.md` ile verilebilir.

Özel derleme/indirme adımları tarifte ayrı program ve argümanlarla desteklenir.
Tarifler yeniden kullanılabilir; hazır tarif uygulamak için Codex gerekmez.
Bu, her MCP için çalışma garantisi değildir: OAuth/lisans, eksik çalışma ortamı
ve dokümantasyondan çözülemeyen koşullar kurulum engeli olarak gösterilir.
[Kurulum asistanı, sınırlar ve test kapsamı](AKILLI-KURULUM.md).

## Sürüm durumu

MCPDeck şu anda terminal odaklı alpha sürümüdür. İmzalı macOS/Linux paketleri ve
Homebrew dağıtımı yayınlanmıştır; Windows için native paketleme ve kurulum vardır.
Production dağıtımından önce tüm desteklenen gerçek ajan oturumları ve bağımsız
güvenlik incelemesi tamamlanmalıdır. Qwen Code ve TraeCode adapter’ları yerel yapılandırma ve
senkronizasyon testleriyle desteklenir; gerçek hesap/OAuth bağlantıları test
ortamının dışındadır.

### Terminal paneli

Bu projedeki derlenmiş binary ile tek komutta kurun:

```sh
sh scripts/install.sh
~/.local/bin/mcpdeck
```

Dağıtım arşivini açtıysanız içindeki `sh install.sh` aynı kurulumu yapar.
Go ve veritabanı kullanıcı tarafında gerekmez; ~/.local/bin PATH’inizdeyse
`mcpdeck` yeterlidir. Yerel MCP sunucuları kendi çalışma ortamlarını gerektirebilir.

Herdr’ın fare ve klavyeyi birlikte kullanan terminal yaklaşımından esinlenen panelde
sol sütun ajanları, orta sütun MCP’leri gösterir. Geniş terminalde ayrıntı sütunu açılır.
Kutular ve üstteki düğmeler tıklanabilir; tekerlekle gezinilir. **+ New MCP** içinde
isim ve doğal dil isteğiyle planlayıcıyı başlatabilirsiniz. Hazır JSON/TOML
ayarları için `mcpdeck import` kullanın.
`--no-mouse` normal terminal metin seçimini korur. En az 64×18 terminal önerilir.

Geliştirici için CLI paketleme: `sh scripts/package-cli.sh`.
`dist/mcpdeck-<sistem>-<işlemci>.tar.gz`, kurulum betiği ve kılavuz içerir;
yanında SHA-256 dosyası üretilir. Yayınlanmış alpha paketleri ve Homebrew tap’i
[public dağıtım deposundadır](https://github.com/altanmehmet/homebrew-mcpdeck).
Windows paketleme: `./scripts/package-windows.ps1 -Version 0.1.0-dev`.

### Terminal

Go **1.24+** gerekir: istenen `@latest` bağımlılıklar çözümlendiğinde Bubble Tea
bu alt sürüm sınırını gerektiriyor. Bağımlılık sürümleri `go.mod` ve `go.sum` ile sabitlenmiştir.

```sh
go build -o mcpdeck .
./mcpdeck                         # TUI
./mcpdeck doctor
./mcpdeck sync --profile cursor --mode direct
./mcpdeck sync --profile cursor --mode bridge
./mcpdeck bridge --profile cursor --idle-timeout 3m
```

TUI: Tab profil, ↑/↓ sunucu seçimi, Space seçili profilde etkinleştir/kapat,
e tüm profillerde etkinleştir, d tüm profillerde kapat, b mod değiştir,
s tüm profilleri senkronize et, q çıkış. Space ve b değişikliği kaydeder ve ilgili
IDE'ye uygular. Senkronizasyon hatasında ayar saklanır; eksikleri giderip s kullanın.
Bağımsız IDE bridge oturumlarının çalışma durumu TUI'ye aktarılmaz.

Varsayılan dosya `~/.config/mcpdeck/deck.json`; her komutta `--config /tam/yol/deck.json`
ile değiştirilebilir. Eksik dosyada altı hazır sunucu ve sekiz temel profil bellekte
oluşturulur; desteklenen diğer ajanlar kuruluysa otomatik eklenir. İlk ayar değişikliğinde
kaydedilir. Başlangıçta bütün sunucular kapalıdır.

```sh
./mcpdeck add
./mcpdeck add --name custom --command /path/to/server \
  --arg stdio --env 'API_TOKEN=${CUSTOM_API_TOKEN}'
./mcpdeck disable custom
./mcpdeck enable custom
./mcpdeck cache-clear custom
```

`add`, yeni sunucuyu varsayılan olarak bütün kayıtlı profillerde etkinleştirir ve
ajan ayarlarına hemen yazar. Sihirbaz komut, JSON argüman listesi, isteğe bağlı
profil ve ortam değişkenlerini bir kez alır; boş profil bütün profiller demektir.
Varsayılan deck temel ajan profillerini içerir. Desteklenen kurulu Gemini CLI,
OpenCode ve Zed ajanları otomatik algılanır. Eski deck dosyanızda eksik temel profiller varsa önce
`profiles add-defaults` çalıştırın; özel profil yolları korunur.

`enable <sunucu>` ve `disable <sunucu>` tüm kayıtlı profilleri kaydedip senkronize
eder. Aynı komutu tekrar çalıştırmak seçimi tersine çevirmez. Tek ajan için
`--profile codex` kullanın; panelde Space tek profilde, e/d tüm profillerde çalışır.
`add --disabled` yalnızca kayıt yapar; `add --no-sync` seçimleri kaydedip IDE'ye
yazmayı erteler. Bunlar eski kayıt/senkronizasyon ayrımını açıkça seçmek içindir.

Senkronizasyon kısmen başarısız olursa tanım ve seçimler saklanır, başarısız
profiller hata mesajında listelenir; sorunu giderip `sync` ile tekrar deneyin.
Çalışan ajanların MCP bağlantısını yeniden başlatın. Özellikle açık Bridge
oturumları eski profil seçimini kullanmayı yeniden bağlantıya kadar sürdürür.

`add` mevcut sunucunun üzerine yazmaz. Komut ve argümanlar shell kullanmadan çalışır.
Sunucu adı `mcpdeck` ayrılmıştır; isimlerde `__` kullanılamaz. Modellerde `mode`
alanı yoksa Direct seçilir; `direct` ve `bridge` değerleri desteklenir.

## Ortam ve hazır tanımlar

| Sunucu | Çalıştırıcı | Gereken ortam değişkeni |
|---|---|---|
| filesystem | npx | MCP_FILESYSTEM_ROOT |
| github | docker | GITHUB_PERSONAL_ACCESS_TOKEN |
| postgres | npx | POSTGRES_URL |
| sqlite | uvx | SQLITE_DB_PATH |
| brave-search | npx | BRAVE_API_KEY |
| fetch | uvx | — |

`${VARIABLE}` referansları argümanlarda ve env değerlerinde çözülür. Eksik veya boş
referanslar işlem başlamadan hata verir. IDE tarafından açılan Bridge, IDE sürecinin
ortamını devralır; terminaldeki export değerleri IDE'ye otomatik ulaşmayabilir.
Direct Sync çözümlenen env/arg değerlerini IDE dosyasına yazar; dosyalar 0600 izinlidir.
Doctor token değerlerini yazdırmaz, API yetkisini veya token geçerliliğini test etmez.
Bütün hazır tanımların ortamını kontrol ettiği için kullanılmayan tanımlarda da eksik
ortam bildirimi verir. Doctor ayrıca npx, uvx, docker ve go araçlarını kontrol eder.

GitHub tanımı [resmi GitHub sunucusunu](https://github.com/github/github-mcp-server)
kullanır. İstenen postgres/sqlite/brave-search tanımları
[arşivlenmiş MCP referans sunucularıdır](https://github.com/modelcontextprotocol/servers-archived);
kullanımdan önce bakımı yapılan bir sunucuyla değiştirin. Sunucu paketleri otomatik
çalıştırılmaz; etkinleştirilip IDE bağlandığında indirilebilir. Üretimde sunucu paket
sürümlerini ve Docker imaj özetlerini kendi doğruladığınız sürümlere sabitleyin.

## Bridge davranışı ve sınırlar

- [MCP 2025-11-25 stdio yaşam döngüsü](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle)
  uygulanır; 2024-11-05, 2025-03-26 ve 2025-06-18 sürümleriyle de el sıkışılır.
  Bridge’in ajan tarafı stdio’dur; uzak HTTP/SSE sunucularına aşağıdaki adaptörle bağlanır.
- `initialize`, `ping`, `tools/list` ve `tools/call` desteklenir. Araç adları
  `sunucu__araç` biçimindedir; yalnızca profilde etkin sunuculara çağrı yapılır.
- Dolu araç önbelleği alt süreç açmadan kullanılır. Eksik/bozuk önbellekte alt süreç
  başlatılıp initialize → initialized → tools/list sırası izlenir; bütün sayfalar alınır.
- Önbellek `cached_tools` alanında Go `[]byte` biçimiyle (base64) saklanır. Sunucu
  komutu/env/paket sürümü veya araç listesi değiştiğinde `cache-clear` kullanın ve
  IDE bağlantısını yeniden başlatın. Çalışan bridge profilin açılış anındaki halini kullanır.
- İstekler sıralı işlenir; varsayılan üst süre 2 dakikadır (`--request-timeout`).
  Aktif çağrı reaper tarafından öldürülmez. Zaman aşımında bağlantı ve süreç kapatılır.
  İstemciden gelen cancellation/progress iletimi ve sunucunun dinamik listChanged
  bildirimleri uygulanmaz. Sampling, roots, elicitation, resources ve prompts
  yetenekleri ilan edilmez; alt sunucunun desteklenmeyen istekleri hata alır.
- Reaper 30 saniyede bir çalışır; varsayılan 3 dakika boşluk sonrası alt süreçleri
  kapatır. macOS/Linux'ta süreç grubu SIGTERM, ardından gerekirse SIGKILL ile temizlenir.
  Windows'ta yalnızca doğrudan child kapanışı uygulanır; descendant garantisi yoktur.
- Bridge kendisi IDE bağlı olduğu sürece çalışır: toplam RAM tüketimi **sıfır değildir**.
  Her IDE bağlantısı kendi süreç havuzunu oluşturur; havuzlar paylaşılmaz.
- stdout yalnızca JSON-RPC içerir. Harici sunucuların stderr çıktısı hassas veri
  sızıntısını önlemek için atılır. Hatalı taşıma yanıtları genel hata olarak döner.
  Mesaj sınırı 16 MiB'dır.

## Uzak MCP: Sentry, Figma ve özel sunucular

### Servise özel kod olmadan otomatik tanıma

**MCP Ekle → Otomatik tanı** ekranına MCP’nin yayınladığı kurulum JSON’unu,
Codex MCP TOML bloğunu veya doğrudan MCP URL’sini yapıştırın. **Bağlantıları Tanı**
ayarları değiştirmeden sunucuları ve eksik bilgileri gösterir. Eksik değerleri
doldurup **Tanınan MCP’leri Ekle ve Dağıt** ile bütün kayıtlı ajanlara uygulayın.
Tek belgede birden fazla MCP desteklenir; aynı isim zaten varsa tüm ekleme reddedilir.

Tanınan biçimler: `mcpServers`, VS Code `servers`, JSON/TOML `mcp_servers`,
isim–sunucu eşlemesi ve tek bağlantı nesnesi. `url`, `serverUrl`, `httpUrl`,
`headers`/`http_headers`, `command`, `args`, `env` ve HTTP/SSE/stdio türleri
ortak modele dönüştürülür. Kaynakta kapalı olan sunucu kapalı kalır.
URL’de ad belirtilmezse alan adından üretilir; adsız bağlantı nesnesinde ad gerekir.

`${VARIABLE}`, `${env:VARIABLE}` ve `${input:identifier}` eksik değerleri formda
istenir. Girilen değerler `variables` alanında 0600 izinli deck dosyasında tutulur;
önizleme çıktısına yazılmaz, `$` içeren değerler tekrar yorumlanmaz.
Yerel programın bağımlılıkları yine kurulu olmalıdır. OAuth ilgili ajan üzerinden
tamamlanır. Özel `cwd`, OAuth istemci ayarları veya araç kısıtları gibi ortak modelde
henüz karşılığı olmayan alanlar sessizce atılmaz; açıklayıcı hata ile durulur.
Bu özellik bir web sayfasını okuyup kurulum komutları çalıştırmaz; MCP bağlantı
ayarlarını tanır. Sağlayıcının özel gereksinimleri otomatik kaldırılmaz.

```sh
./mcpdeck import --preview < server-config.json
./mcpdeck import < server-config.json
./mcpdeck import --disabled < server-config.json
```

### URL ile terminalden ekleme

```sh
./mcpdeck add --name sentry --url https://mcp.sentry.dev/mcp
./mcpdeck add --name figma --url https://mcp.figma.com/mcp
./mcpdeck add --name internal --url https://example.com/mcp \
  --header 'Authorization=Bearer ${MCP_TOKEN}'
./mcpdeck disable sentry
./mcpdeck enable sentry
```

`--transport sse` eski SSE sunucuları içindir; varsayılan HTTP’dir. Yerel `command`
ve uzak `url` birbirini dışlar. URL sunucularının kimlik bilgileri `headers`, yerel
sunucularınki `env` ile verilir. Eksik referanslar ve geçersiz başlıklar reddedilir.
Değerler şifreli bir kasada tutulmaz; deck ve ajan ayarları 0600 izinli dosyalardır.
MCPDeck daha geniş izinli deck dosyasını açılışta 0600'a düzeltir; düzeltemezse
potansiyel sır sızıntısını önlemek için işlemi durdurur.

Yeni profillerde Cursor `url`, Claude Code `type/http`, Windsurf/Antigravity
`serverUrl`, VS Code/Copilot CLI kendi HTTP şemaları, Codex `url/http_headers`
kullanır. Claude Desktop, eski/özel `mcpServers` profilleri, Codex’te SSE ve
Bridge bağlantıları sabit `mcp-remote@0.1.38` stdio adaptörünü kullanır.
Adaptör ilk kullanımda npm üzerinden indirilebilir; Node/npx ve internet gerekir.
Özel profillerde `remote_format`: `url`, `serverUrl`, `http` veya `stdio` seçilebilir.
Mevcut profiller otomatik yeniden yorumlanmaz; eski genel JSON profilleri adaptöre düşer.

URL’yi kaydetmek OAuth izni vermez. Her ajanda servis hesabınızı yetkilendirin;
sağlayıcının istemci/kurum kısıtları geçerlidir. Figma yalnızca desteklediği istemcileri
kabul edebilir; adaptörle her istemcide erişim garantisi yoktur. Yerleşik HTTP desteği
olan ajanlarda Direct mod tercih edilir; Bridge yalnızca araç çağrılarını aktarır.

Kaynaklar: [Sentry MCP](https://mcp.sentry.dev/),
[Figma uzak MCP](https://developers.figma.com/docs/figma-mcp-server/remote-server-installation/),
[mcp-remote](https://github.com/geelen/mcp-remote).

## Dosya güvenliği

Deck ve IDE dosyaları geçici dosya + fsync + rename yöntemiyle atomik yazılır.
Deck güncellemeleri süreçler arası kilitle yeniden yüklenir; önbellek yazımı diğer
profil değişikliklerini ezmez. Zorla sonlanan bir yazıcı `.lock` dizini bırakabilir;
hata mesajındaki yolu yalnızca hiçbir yazıcı çalışmıyorken kaldırın.

IDE dosyasındaki diğer üst düzey ayarlar ve deck'te tanımlı olmayan MCP girdileri
korunur. Deck'teki sunucu isimleri ve `mcpdeck` girdisi uygulama tarafından yönetilir;
aynı isimli mevcut IDE girdileri değiştirilir. Bridge modunda yönetilen girdiler tek
`mcpdeck` girdisiyle değiştirilir; harici girdiler korunur. Deck'ten elle silinen bir
sunucunun eski IDE girdisini ayrıca kaldırın. Bozuk JSON/TOML üzerine yazılmaz. VS Code profili JSONC yorumlarını ve trailing comma
sözdizimini okuyabilir; çıktı standart JSON olur. TOML ve JSONC yorumları/biçimlendirmesi
yeniden yazımda korunmaz; diğer ayar değerleri korunur. Eşzamanlı harici IDE dosya düzenlemeleriyle
senkronizasyonu çakıştırmayın. Bridge config'i mutlak executable ve deck yolunu içerir;
uygulamayı taşırsanız tekrar sync çalıştırın.

## Doğrulama

```sh
go test ./...
go test -race ./...
go vet ./...
```

Testler gerçek token veya ağ kullanmadan sahte MCP alt süreci ile lazy spawn,
yeniden kullanım, el sıkışması, sayfalama, önbellek, namespace yönlendirmesi,
idle reaping, timeout, eşzamanlı spawn ve atomik yapılandırma güncellemelerini sınar.
Gerçek IDE entegrasyonu için bir test profili/yolu kullanarak filesystem root veya
fetch sunucusunu etkinleştirin; tools/list ve tools/call deneyin, boşluk süresi sonrası
child'ın kapandığını gözleyin. Gerçek sağlayıcı bağlantıları otomatik test edilmez.

## Ajan entegrasyonları

Mevcut deck dosyanıza yeni profilleri eklemek için `./mcpdeck profiles add-defaults`
çalıştırın. Bu komut yalnızca eksik profilleri ekler; mevcut profil yolları, modları ve
sunucu seçimlerini değiştirmez, IDE dosyalarına yazmaz. Yeni sunucular kapalı gelir.
Ardından `./mcpdeck` içinde ilgili sekmeden sunucuları seçin.
`./mcpdeck profiles` profil, mod, biçim ve hedef yollarını listeler.

| Profil | Uygulama | macOS varsayılan hedefi | Biçim |
|---|---|---|---|
| cursor | Cursor | `~/.cursor/mcp.json` | mcpServers JSON |
| claude | Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` | mcpServers JSON |
| claude-code | Claude Code (user scope) | `~/.claude.json` | mcpServers JSON |
| codex | Codex | `~/.codex/config.toml` | mcp_servers TOML |
| copilot | GitHub Copilot / VS Code | `~/Library/Application Support/Code/User/mcp.json` | servers JSONC → JSON |
| copilot-cli | GitHub Copilot CLI | `$COPILOT_HOME/mcp-config.json` veya `~/.copilot/mcp-config.json` | mcpServers JSON; type=local, tools=["*"] |
| windsurf | Windsurf | `~/.codeium/windsurf/mcp_config.json` | mcpServers JSON |
| antigravity | Antigravity | `~/.gemini/config/mcp_config.json` | mcpServers JSON |
| qwen-code | Qwen Code | `~/.qwen/settings.json` | mcpServers JSON; HTTP için httpUrl |
| trae | TraeCode | proje içindeki `.trae/mcp.json` | mcpServers JSON |

Bütün bu profiller Direct ve Bridge üretimini destekler. `sync --profile` sunucuları
etkinleştirmez; `enable` veya panelden seçim yapın. Örneğin `./mcpdeck sync --profile codex --mode bridge`
Codex'e Bridge girdisini yazar. Değişiklikten sonra ajan bağlantısını yeniden başlatın.

Codex varsayılanı, varsa `CODEX_HOME` dizinini kullanır. VS Code Insiders, portable,
remote veya özel kullanıcı profili için uygulamanın “MCP: Open User Configuration”
komutuyla gerçek yolu bulun ve `target_path` değerini değiştirin. Copilot profili
VS Code içindir; JetBrains veya Visual Studio yapılandırmalarını yönetmez.
Eski Antigravity sürümünüz başka bir dosya kullanıyorsa “View raw config” ile yolu
kontrol edip `target_path` alanını değiştirin. Mevcut hedefler otomatik taşınmaz.
Claude Code'un `projects` alanı ve Codex'in model/güvenlik ayarları korunur;
proje bazındaki MCP tanımları kullanıcı kapsamındaki tanımları geçersiz kılabilir.
Copilot CLI `COPILOT_HOME` tanımlıysa bu dizindeki `mcp-config.json` dosyasını kullanır;
aksi halde `~/.copilot/mcp-config.json` kullanılır. `tools=["*"]` araçların kullanılabilirliğini tanımlar; uygulamanın kullanıcı
onaylarını kapatan bir ayar eklenmez.

Yeni isteğe bağlı `format` alanı: `mcpServers`, `vscode`, `copilot-cli`, `codex`.
Alan yoksa eski `mcpServers` JSON davranışı devam eder; profil ismi üzerinden
biçim tahmin edilmez. Eski bir özel `codex` profili otomatik dönüştürülmez; TOML
üretmesi için `format` alanını `codex` olarak ayarlayın.

Kaynaklar: [Codex](https://developers.openai.com/codex/mcp),
[Claude Code](https://code.claude.com/docs/en/mcp),
[VS Code](https://code.visualstudio.com/docs/agents/reference/mcp-configuration),
[Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers),
[Antigravity](https://antigravity.google/docs/ide/mcp).
Qwen Code için [MCP belgeleri](https://qwenlm.github.io/qwen-code-docs/en/users/features/mcp/),
TraeCode için [MCP belgeleri](https://docs.trae.cn/ide_add-mcp-servers) kullanılmıştır.

### MCP kaldırma

### Kurulu MCP'yi onarma

Listede seçili MCP için **Repair** düğmesine veya `u` tuşuna basın. MCPDeck önce mevcut
bağlantıyı `initialize` ve `tools/list` ile dener; hata metnini parolaları göndermeden
planlayıcı chat'ine taşır. Sohbette sorunu açıklayın, `/refresh` ile yerel araçları yeniden
kontrol edin veya Enter ile tarifi onaya gönderin. Onaydan sonra tarif çalışır, MCP yeniden
probelenir ve başarılı olursa yalnızca o MCP'nin açık olduğu tüm ajan profilleri yedeklenip
güncellenir. Yeni probe başarısız olursa eski ajan ayarları korunur.

Terminalden aynı akış:

```bash
mcpdeck repair <mcp-adı> [sorun açıklaması]
mcpdeck update <mcp-adı>
```

MCPDeck dışında eklenmiş `external` kayıtlar doğrudan onarılamaz; önce Import ile MCPDeck
kaydına alınmaları gerekir. Özel değerler mevcut kayıttan yerel olarak yeniden kullanılır veya
eksikse gizli giriş olarak sorulur; planlayıcıya değerlerin kendisi gönderilmez.

Listeden MCP'yi seçin ve **Remove** düğmesine tıklayın veya `x` tuşuna basın.
Onay ekranında Enter ya da `y` kaldırır; Esc iptal eder.
Terminalden: `mcpdeck remove oracle-mcp` (otomasyon için `--yes`).
MCPDeck kaydı ve tüm tanımlı ajanlardaki MCP ayarları kaldırılır.
İndirilen dosyalar, veritabanı bağlantıları ve paylaşılan çalışma ortamları korunur.
Kısmi eşitleme hatasında devre dışı kayıt korunur; kaldırma komutunu yeniden deneyin.

### Ajanlarda önceden bulunan MCP'ler

MCPDeck, tanımlı ajanların MCP ayarlarını açılışta ve Refresh ile salt okunur tarar.
MCPDeck dışında eklenen bağlantılar listede `external` olarak ve bulundukları
ajanlarla görünür. Otomatik içe aktarılmaz veya başka ajanlara dağıtılmaz.
Bunları da **Remove** / `x` ile kaldırabilirsiniz; yalnızca o adın bulunduğu
ajan ayarları değiştirilir, önce `.mcpdeck-remove-backup` yedeği alınır.
Kimlik bilgileri listede gösterilmez. Bozuk/okunamayan ayarlar açıkça bildirilir.
Standart konum dışındaki ayarlar için ilgili ajan profilinin yolu tanımlanmalıdır.

```bash
mcpdeck discover
mcpdeck remove existing-mcp-name
```
