# MCP kurulum asistanı

## Kullanım

`mcpdeck` panelindeki **Install MCP** düğmesine basın veya `i` tuşunu kullanın.
Resmi dokümantasyon ya da GitHub bağlantısını verin. Asistanın terminal akışı
planı gösterir, `y` + Enter ile onay alır, gerekli girdileri
gizli olarak sorar ve kurulumdan sonra MCP bağlantısını doğrular.

Planın ardından açılan `You:` alanı seçilen ajana bağlı kurulum sohbetidir.
Soru sorun, kurulu yazılımı kullanmasını isteyin veya bir adımı değiştirin.
Enter ya da `/install` son tarif için ayrı komut onayına geçer; `/cancel` iptal
eder. `/refresh`, yazılım ortamını yeniden tarayıp tarifi günceller. Manuel koşul
ekranında `DONE` yerine soru yazmak da aynı sohbete döner. Konuşma bu kurulumun
belleğinde tutulur; yeni tarif eski plan kimliğiyle uygulanamaz.

`mcpdeck environment` kurulu çalıştırıcıları ve Java dizinlerini gösterir.
Tarama bilinen araçların sabit sürüm komutlarını ve standart kurulum dizinlerini
kullanır; tüm diski taramaz. Her planlama turunda güncel yollar/sürümler sağlayıcıya
iletilir. `.dbtools` dizininin bulunması, kayıtlı bağlantı veya başarılı girişin
doğrulandığı anlamına gelmez; bağlantı dosyalarının içeriği okunmaz.

Alternatif:

```sh
mcpdeck install 'dokümantasyon URL veya kurulum isteği' --plan-only --out plan.json
mcpdeck install --plan plan.json
```

## Sağlayıcı seçimi

| Seçenek | Bağlantı | API anahtarı ortam değişkeni |
| --- | --- | --- |
| `codex` | Kurulu Codex CLI oturumu (mevcut varsayılan) | CLI oturumu |
| `claude` | Kurulu Claude Code oturumu | CLI oturumu veya `ANTHROPIC_API_KEY` |
| `gemini` | Kurulu Gemini CLI oturumu | CLI oturumu veya `GEMINI_API_KEY` / `GOOGLE_API_KEY` |
| `openai-api` | OpenAI Responses API | `OPENAI_API_KEY` |
| `anthropic-api` | Anthropic Messages API | `ANTHROPIC_API_KEY` |
| `gemini-api` | Gemini generateContent API | `GEMINI_API_KEY` |
| `openai-compatible` | Özel Chat Completions API adresi | `MCPDECK_API_KEY` (yerel serviste isteğe bağlı) |

```sh
mcpdeck planner list
mcpdeck planner set --provider claude
mcpdeck planner                 # Kayıtlı seçimi göster
mcpdeck install 'resmi dokümantasyon URL' --provider gemini --plan-only
# MODEL_ID yerine hesabınızda erişebildiğiniz gerçek model adını yazın:
mcpdeck planner set --provider anthropic-api --model MODEL_ID
mcpdeck install 'resmi dokümantasyon URL'
```

Panelden **Install MCP** seçildiğinde sağlayıcı adı sorulur; Enter kayıtlı seçimi kullanır.
API model adı eksikse sorulur. Anahtar ortamda yoksa gizli giriş alanından alınır;
bu anahtar yalnızca o işlem için kullanılır. `planner set` seçimi `deck.json`
yanındaki `planner.json` dosyasına kaydeder; anahtarın kendisini kaydetmez.
Her API'nin model ve web aracı desteği sağlayıcının hesabınıza açtığı modellere bağlıdır.
CLI modelini belirtmezseniz ilgili CLI'nin varsayılanı kullanılır. API model adı
zorunludur; uygulama model adını tahmin etmez. Kullanım seçtiğiniz hesaba yansır.

`--planner /tam/yol/program` CLI yolunu değiştirir (eski Codex kullanımını korur).
`--key-env KENDI_ANAHTAR_DEGISKENIM` farklı ortam değişkeni seçer; anahtar değerini
komut satırına yazmayın. API modlarında yerel CLI kurulumu gerekmez.

OpenAI, Anthropic ve Gemini API adaptörleri sağlayıcının web araçlarını ister.
OpenAI uyumlu Chat Completions servislerinde standart web aracı yoktur: README veya
kurulum metnini `--docs README.md` ile verin. Yalnızca URL verilirse modelden
eksik dokümantasyonu `manual` engeli olarak raporlaması istenir. Web erişimi varmış
gibi kabul edilmez. Yerel modeller de bu servis üzerinden kullanılabilir:

```sh
mcpdeck planner set --provider openai-compatible --model MODEL_ID --base-url http://127.0.0.1:11434/v1
mcpdeck install 'Bu dokümantasyondaki MCP için plan oluştur' --docs README.md --plan-only
```

Özel temel adres yalnızca `openai-compatible` seçeneğinde kullanılır. HTTPS
zorunludur; HTTP yalnızca localhost/loopback için kabul edilir. Anahtarın başka
adrese taşınmaması için API yönlendirmeleri izlenmez. Ham sağlayıcı hata gövdeleri
ve CLI hata çıktıları sır sızdırmamak için gösterilmez.

CLI planlayıcıları geçici dizinde çalışır. Codex salt okunur sandbox ve geçici
oturum kullanır. Claude'a yalnız WebSearch/WebFetch araçları verilir; MCP config
yüklenmez. Gemini yerel araç allowlist'i ve policy ile web araçlarına sınırlandırılır;
headless plan modunun otomatik uygulama geçişi kullanılmaz. CLI'nin kurumsal
ayarları, sürümü ve oturum saklama davranışı yine geçerlidir; bu işletim sistemi
düzeyinde tam izolasyon garantisi değildir.

İstek, varsa `--docs` dosyası, sohbet yanıtları ve yerel yazılım envanteri seçilen sağlayıcıya
gönderilir. MCP kurulum girdileri, deck ve ajan config içerikleri prompt'a eklenmez.
API anahtarı yalnız API kimlik doğrulama başlığında kullanılır. CLI'lar kendi oturum
açma bilgilerine erişmeye devam eder. Belgeye/isteğe sır koymayın.

Referanslar: [Codex](https://learn.chatgpt.com/docs/non-interactive-mode),
[Claude Code](https://code.claude.com/docs/en/cli-reference),
[Gemini CLI](https://geminicli.com/docs/cli/headless/),
[OpenAI web araçları](https://developers.openai.com/api/docs/guides/tools-web-search),
[Anthropic web aracı](https://platform.claude.com/docs/en/agents-and-tools/tool-use/web-search-tool),
[Gemini API](https://ai.google.dev/api/generate-content).

## Kurulum tarifleri

Tarif sürümü 1: `name`, `summary`, `sources`, `requirements`, `steps`, `connection`,
`manual` ve isteğe bağlı `follow_up`. Eski tarif dosyaları ve onay kimlikleri
korunur. `follow_up`, yazılım kurulduktan ve ajanlara dağıtıldıktan sonra gereken
hesap/veritabanı adımlarını gösterir; MCP başlamasına engel olmayan bu adımlar
kurulumu durdurmaz. Connection tek MCP bağlantısının JSON metnidir. Her step `command`,
ayrı `args`, kurulum dizini içinde göreli `directory`, `description` içerir.
`${INSTALL_DIR}` kalıcı `deck.json` yanındaki `installations/<name>` dizinidir.
Örnek kullanım alanları: npm prefix kurulumu, Python venv, Git checkout + build,
Docker image hazırlama ve özel kurulum programları. Yeni kütüphane eklenmedi.

Etkileşimli kurulumda `Install this plan? [y/N]` sorusuna `y` + Enter ile onay verilir; boş giriş iptal eder. Etkileşimsiz kullanımda plan tam SHA-256 kimliğiyle onaylanır. Değişmiş plan eski onayla uygulanamaz.
Komutlar shell üzerinden geçirilmez; shell/sudo adımları reddedilir. Ancak
çalıştırılan programlar kullanıcının yetkilerine sahiptir: bu executor bir sandbox
değildir. Kaynakları ve komutları onaylamadan önce inceleyin. Programların yaptığı
harici değişiklikler veya indirilen paketler otomatik geri alınmaz.
Adımlara şifre koymayın; sırlar yalnızca connection içindeki `${VARIABLE}`
girdileriyle kullanılır. Kurulum adımları bu değerleri almaz. Çocuk program çıktısı
gizli değer sızdırmamak için gösterilmez; hata hangi adımın başarısız olduğunu söyler.

## Doğrulama ve dağıtım

- Gereksinimler ve mevcut ajan dosyalarındaki ad çakışmaları önceden kontrol edilir.
- `manual` boş değilse kullanıcı koşulları tamamlayıp `DONE` ile doğrular veya
  ajana soru sorarak tarifi düzeltir. Eksik desteklenen macOS çalıştırıcıları
  Homebrew ile kurmak için ayrı izin istenir. Otomatik OAuth
  yöneticisi veya başarısız komutları modelle tekrar tekrar çalıştırma döngüsü yoktur.
- Onaydan sonra adımlar yürütülür. Ardından gerçek MCP `initialize`,
  `notifications/initialized`, sayfalanmış `tools/list` protokolü denenir.
- İş verisi okuyan/yazan araç çağrısı yapılmaz. Bu test araçların tüm işlevlerinin
  doğru çalıştığını veya her ajanın mevcut oturumunun yenilendiğini kanıtlamaz.
- Uzak bağlantı testi mevcut `mcp-remote` adaptörünü kullanır; Node/npx gerektirir.
  OAuth veya desteklenmeyen istemci koşulu varsa doğrulama başarısız olabilir.
- Doğrulamadan önce deck ve ajan dosyaları değiştirilmez. Başarısız kurulumun
  kendi indirdiği dosyalar sonraki inceleme için kalabilir; tekrar çalışan adımlar
  otomatik olarak atlanmaz, tarife uygun davranır.
- Varsayılan hedefler: config dosyası bulunan destekli profiller; varsayılan
  profillerde PATH veya standart Mac uygulama dizininden tespit edilen ajanlar.
  Bu tespit her kurulum konumunu bulamaz. `--all-profiles` kayıtlı tüm profilleri
  dahil eder. Özel config yolları mevcut profil sistemiyle tanımlanabilir.
- Şu anki hedef ajanlar: Cursor, Claude Desktop, Claude Code, Codex, VS Code Copilot,
  Copilot CLI, Windsurf, Antigravity, Gemini CLI, OpenCode, Zed, Cline, Roo Code,
  Continue, Amazon Q Developer ve Kiro. Gemini CLI, OpenCode, Zed, Continue, Amazon Q
  ve Kiro ayarları/uygulamaları bulunduğunda profilleri otomatik eklenir. Roo Code ve Cline
  her desteklenen VS Code uyumlu editörün uzantı deposunda ayrı algılanır. Bir MCP eklendiğinde
  kayıtlı ajan profillerinin tümüne dağıtılır; tek ajana özel seçim de korunur.
  [Gemini CLI MCP ayarları](https://geminicli.com/docs/tools/mcp-server/) ve
  [Zed MCP ayarları](https://zed.dev/docs/ai/mcp) kendi config biçimlerini kullanır;
  MCPDeck bunları ayrı adaptörlerle yazar. [OpenCode MCP ayarları](https://opencode.ai/v2/docs/mcp-servers)
  yerel ve uzak sunucuları destekler.
- Goose MCP sunucularını YAML `config.yaml` içinde saklar. Mevcut MCPDeck JSON/TOML
  yazıcısı YAML dosyalarını güvenle koruyamadığı için Goose henüz otomatik dağıtım
  hedefine dahil değildir.
- Ajanlar MCP ayarlarını farklı dosyalarda ve farklı alan adlarıyla saklayabilir.
  MCP standardı kurulum veya ajan keşfi için ortak bir işletim sistemi kaydı tanımlamaz.
  Bu nedenle listelenmeyen her ajan için o istemcinin config biçimini destekleyen bir
  adaptör gerekir; algılanmayan bir istemciye yanlış dosyaya yazarak kurulum yapılmaz.
- Var olan dosyanın son kurulum öncesi kopyası `.mcpdeck-backup` olarak 0600
  izinlerle saklanır; sonraki kurulum bu tek yedeği yeniler. İlgisiz ayarlar korunur.
- Kısmi yazma hatasında başarılı hedefler kalır, deck kaydı korunur; eksik hedefler
  raporlanır ve `mcpdeck sync` ile yeniden denenir. Dağıtım çok dosyalı atomik işlem değildir.
- `mcpdeck disable <ad>` / `enable <ad>` mevcut yönetim akışını kullanır.
  Açık ajanlarda MCP bağlantısını yenilemek/ajanı yeniden başlatmak gerekebilir.
- Copilot CLI kullanıcı ayarı `COPILOT_HOME/mcp-config.json` konumundadır;
  `COPILOT_HOME` boşsa varsayılan `~/.copilot/mcp-config.json` kullanılır.
  Copilot CLI zaten açıkken yapılan değişiklik için oturumu yeniden başlatın.

## Otomasyon

`--approve <tam-plan-kimliği>` etkileşimsiz onaydır. Değerleri ortam değişkeniyle
veya `--values-stdin` ile JSON nesnesi olarak verin; komut argümanına sır koymayın.
Girdiler kurulumdan sonra deck içinde 0600 dosya izinleriyle tutulur, şifrelenmez.
MCPDeck açılışta deck dosyası daha geniş izinliyse 0600'a düşürür; izin değişikliği
başarısızsa sırları kullanmaya devam etmek yerine işlemi durdurur. Ajan yapılandırmaları
da yazılırken 0600 olarak oluşturulur.

## Test sınırları

Testler gerçek yerel MCP alt süreciyle bağlantı ve iki sayfalı araç listesini,
sekiz config biçimine dağıtımı, aç/kapatmayı, yedekleri, başarısız bağlantıda
değişiklik yapılmamasını, onay kimliğini ve dizin kaçışlarını kontrol eder.
CLI planlayıcı testleri üç CLI'nin protokolünü taklit eden yerel fixture kullanır.
Dört API adaptöründe istek/yanıt biçimi, kimlik doğrulama başlıkları, sırların
prompt'a/ayar dosyasına girmemesi, hata yanıtları ve yönlendirme engeli test edilir.
Canlı model hesaplarıyla dokümantasyon keşfi veya Oracle/Figma/Sentry hesaplarına
uçtan uca kurulum bu testlerin kapsamı değildir.
