# MCPDeck — terminalden MCP yönetimi

## Hızlı başlangıç

Günlük kullanım için aşağıdaki komutlar yeterlidir:

| Yapmak istediğim | Komut |
|---|---|
| Paneli açmak | `mcpdeck` |
| MCP sunucularını görmek | `mcpdeck list` veya `mcpdeck ls` |
| Ajanları ve modlarını görmek | `mcpdeck agents` |
| Yeni MCP bağlamak | `mcpdeck connect "Add the official Git MCP"` |
| Tek ajanda etkinleştirmek | `mcpdeck enable SUNUCU --profile cursor` |
| Tek ajanda kapatmak | `mcpdeck disable SUNUCU --profile cursor` |
| Sorunu incelemek | `mcpdeck doctor` |

`connect`, mevcut `install` komutunun kısa adıdır; tarif inceleme/onay akışı aynıdır.
`enable` ve `disable` komutlarında `--profile` verilmezse bütün kayıtlı ajanlar etkilenir.
`list --profile cursor` her sunucunun Cursor için açık/kapalı olduğunu gösterir.
Scriptler için `mcpdeck status` aynı JSON çıktısını üretmeye devam eder;
`mcpdeck list --json` da bu biçimi kullanır. İnsan için olan listede ayrıca ajan
ayarlarında bulunan dış sunucular yer alır; JSON biçimi yönetilen sunucularla sınırlıdır.

Panelde ana düğmeler New MCP, Instructions, Sync, Help ve Exit'tir.
**More** veya `m` ile onarım, dokümantasyondan kurulum, mod değiştirme, yenileme,
yeniden deneme, kısayol ayarlama, metin seçimi ve kaldırmaya ulaşılır.
Menüde ↑/↓ ile seçip Enter'a basın; Esc geri döner. Önceki klavye kısayolları korunur.

`mcpdeck --help` komutları günlük kullanım, bağlantılar, bakım ve ileri kullanım
başlıklarında gösterir. Ayrıntılar için `mcpdeck list --help` gibi alt komut yardımını açın.


## Genel talimatlar

**Instructions** düğmesine veya `g` tuşuna basın. Bütün projelerde geçerli kişisel
talimatlarınızı yazın/yapıştırın. `Ctrl+S` hedefleri gösterir; tekrar `Ctrl+S`
kaydeder ve algılanan destekli ajanlara dağıtır. `Ctrl+Y` metni kopyalar. Proje
talimatları değiştirilmez. Sonuçta `manual` görünen ajanlara metni kişisel
ayarlarından ekleyin. Terminalden `mcpdeck instructions` aynı editörü açar;
`mcpdeck instructions set --file rules.md` dosyadan toplu dağıtır.
Hedefleri kontrol etmek için `mcpdeck instructions status`, yeniden denemek için
`mcpdeck instructions sync` kullanın. Ayrıntılar: `INSTRUCTIONS.md`.

**Existing** / `Ctrl+E` mevcut ajan talimatlarını açar; düzenlemeyi iki kez
`Ctrl+S` ile yedekleyip kaydedin. **Use for all** / `Ctrl+F`, o dosyanın metnini
ortak editöre taşır. **Add one** / `Ctrl+N`, tek talimatı önceki ortak metni
koruyarak tüm algılanan destekli ajanlara ekler. Terminal karşılığı:

    mcpdeck instructions append 'Never log credentials.'
    mcpdeck instructions show --agent codex

## Dokümantasyondan kurulum

Panelde **More → Install from documentation** veya **i**, MCP dokümantasyonundan kurulum planı üretir.
Planı onayladıktan sonra kurulum ve gerçek bağlantı testi yapılır; başarılıysa
tespit edilen destekli ajanlara dağıtılır. Codex, Claude Code veya Gemini CLI;
alternatif olarak OpenAI, Anthropic, Gemini veya OpenAI uyumlu API seçilebilir.
`mcpdeck install` aynı akışı açar. `mcpdeck planner set --provider claude`
varsayılanı değiştirir. Ayrıntılar: `AKILLI-KURULUM.md`.

Kurulu bir MCP hata veriyorsa panelde MCP'yi seçip **More → Repair / update selected MCP** işlemine veya `u`
tuşuna basın. Açılan chat mevcut bağlantıyı probeler, sorunu ve kurulu araçları
güvenli özetle ajana verir. Terminal karşılığı:

    mcpdeck repair oracle-mcp "tools/list zaman aşımına uğruyor"

Onaydan sonra yeni tarif çalıştırılır, tekrar `initialize` + `tools/list` yapılır ve
başarılı sonuç tüm açık ajan profillerine dağıtılır. Başarısız probe eski ayarları
korur. `mcpdeck update <ad>` aynı komutun kısa adıdır. MCPDeck dışındaki `external`
kayıtlar önce içe aktarılmadan onarılamaz.

Plan gösterilince `You:` alanından ajana soru sorun veya değişiklik isteyin.
`/refresh` kurulu yazılımları yeniden kontrol eder; Enter veya `/install` onaya
geçer, `/cancel` iptal eder. Gizli değerleri sohbet mesajına yazmayın.
`mcpdeck environment` algılanan çalıştırıcıları ve Java dizinlerini gösterir.

## Kurulum

Windows'ta native ZIP içindeki `install.ps1` ile kurun; PowerShell/Windows
Terminal'de `mcpdeck` çalışır. WSL veya Go gerekmez.
[Windows komutları ve test adımları](WINDOWS.md).

Mac/Linux için işlemcinize uygun CLI arşivini açın. Açılan klasörde:

    sh install.sh
    ~/.local/bin/mcpdeck

Go, Node veya veritabanı MCPDeck’in kendisi için gerekmez. Seçtiğiniz MCP
sunucusunun ihtiyaç duyduğu Node/uvx/Docker gibi programlar ayrıca gerekir.
~/.local/bin PATH’inizdeyse sonraki açılışlarda yalnızca `mcpdeck` yazın.
Kaynak proje klasöründeyseniz önce Go 1.24+ ile `go build -o mcpdeck .`,
sonra `sh scripts/install.sh` kullanın.
Yönetici yetkisi istenmez; araç ~/.local/bin içine kurulur.

Qwen Code kullanıyorsanız `~/.qwen/settings.json` otomatik algılanır. TraeCode
proje kapsamlı `.trae/mcp.json` dosyasını kullanır; proje klasöründen çalışırken
`TRAE_MCP_CONFIG` ile farklı bir Trae dosyası belirtebilirsiniz.

## Fareyle kullanım

- Soldaki ajan adına tıklayın.
- Ajan adının yanındaki `●` simgesi desteklenen ajanın algılandığını gösterir;
  `○` hedef dosyası bu bilgisayarda bulunamadı demektir. MCP ekleme tüm kayıtlı
  ajanlara uygulanır; algılanan Gemini CLI, OpenCode, Zed, Cline, Roo Code, Continue
  Amazon Q ve Kiro profilleri otomatik eklenir.
- Ortadaki MCP adına tıklayarak seçin.
- Satırdaki kutuya tıklayarak yalnızca seçili ajanda açın/kapatın.
- Üstteki Hepsini aç/kapat düğmeleri bütün kayıtlı ajanları etkiler.
- Fare tekerleğiyle bulunduğunuz sütunda gezinin.
- **+ New MCP**: MCP adını ve doğal dil isteğinizi girin; Ctrl+S planlayıcıyı
  başlatır. URL zorunlu değildir. Hazır JSON/TOML için `mcpdeck import` kullanın.
- Yerel ve uzak MCP’ler desteklenir. Özel, desteklenmeyen alanlar hata verir.
- ? yardım ekranında kılavuz ve ilk kurulum adımları görünür. Sağ ok/Tuşlar sekmesi
  atanmış tüm tuşları gösterir.
- Ana ekranda `a` tuşuna basın veya yardımdaki **Tuşları ata** düğmesine tıklayın. Bir işlemi
  seçip Enter’a basın ya da satıra tıklayın; ardından kullanmak istediğiniz tuşa
  basın. Atamalar ~/.config/mcpdeck/ui.json dosyasında saklanır. Aynı tuş iki
  işleme atanamaz; Ctrl+C çıkış için saklıdır. Ctrl+R varsayılanları yükler.

Terminalde en az 64 sütun ve 18 satır gerekir. Geniş ekranda ayrıntı sütunu görünür.
Terminaliniz fare olaylarını desteklemiyorsa klavye kısayollarını kullanabilirsiniz.
`mcpdeck --no-mouse` normal terminal metin seçimini korur.

## Klavye

Tab: ajan değiştir; ↑/↓: MCP seç; Space: seçili ajanda aç/kapat.
e/d: tüm ajanlarda aç/kapat; b: Direct/Bridge modu; s: eşitle.
n: MCP ekle; r: yeniden oku; ?: yardım; q: çıkış.
m: diğer işlemler; i: kurulum asistanı; a: yardım içinden tuş ata.
Yeni MCP formunda Tab: alan değiştir; Ctrl+U: alanı temizle;
Ctrl+S: planlayıcıyı başlat; Esc: vazgeç. Ctrl+R: başarısız sync hedeflerini tekrar dene.
`mcpdeck sync status` sonuçları, `mcpdeck sync --retry-failed` tekrar denemeyi sağlar.
Geri yükleme için `mcpdeck sync restore --profile cursor` ile önce önizleyin;
onay için `--yes` ekleyin. Ayrıntılar: `RECOVERY.md`.

Etkin göstergesi ayar seçimini belirtir, canlı bağlantı durumunu değil.
Açık ajanların MCP bağlantısını yenileyin; gerekiyorsa hesabınızı yetkilendirin.

## Güncelleme ve kaldırma

Çalışan panelden q ile çıkın, yeni pakette tekrar `sh install.sh` çalıştırın.
Ayarlar ~/.config/mcpdeck/deck.json dosyasında kalır. Bridge modu kurulum yolunu
kullanır; binary’yi taşıdıysanız yeniden eşitleyip ajan bağlantılarını yenileyin.
Kaldırmadan önce yönetilen bağlantıları panelden kapatıp ajanları yenileyin;
ardından ~/.local/bin/mcpdeck dosyasını kaldırabilirsiniz. Ayarlar ayrıca korunur.
