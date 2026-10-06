# Genel talimatları tek yerden yönetme

MCPDeck, kişisel ve bütün projelerde geçerli olmasını istediğiniz talimatları tek
bir Markdown metni olarak yönetir. Panelde **Instructions** düğmesini veya `g`
tuşunu kullanın. Metni yazın/yapıştırın, **Review targets** ile ajanları inceleyin,
ardından **Save & distribute** ile kaydedip dağıtın. Klavye karşılığı iki kez
`Ctrl+S`'dir: ilk basış inceleme, ikinci basış dağıtım. `Enter` yeni satır, oklar
imleci hareket ettirir; `Esc` incelemeden düzenlemeye, düzenlemeden panele döner.
Kaydetmeden çıkmak düzenlemeleri iptal eder.

**Existing** düğmesi veya `Ctrl+E`, ajanların mevcut genel talimat dosyalarını
listeler. MCPDeck dışında oluşturulmuş dosyalar ve Cursor/Kiro/Cline kural
dizinlerindeki dosyalar da gösterilir. Claude Code'un `~/.claude/rules` ve Copilot'un
global `instructions` dizinindeki modüler talimatlar da bu listeye dahildir.
Bir satıra tıklayın veya Enter'a basın: dosyanın güncel metni açılır.

Mevcut metni düzenledikten sonra iki kez `Ctrl+S`, yalnızca seçili genel dosyayı
yedekleyip günceller. Gemini/Antigravity gibi aynı dosyayı okuyan ajanlar değişikliği
birlikte görür. Dosya başka oturumda değişirse eski düzenleme üzerine yazılmaz.
MCPDeck'e ait ortak bölüm bu görünümde korunur; ortak talimatları **Shared editor**
içinden değiştirin.

**Use for all** veya `Ctrl+F`, açtığınız dosyadaki talimat metnini ortak editöre
taşır. Dosyaya özgü üst metadata ve MCPDeck işaretleri ortak metne aktarılmaz.
Bu adımda henüz dosya değişmez: hedefleri inceleyip dağıtımı onaylamanız gerekir.
Kaynak dosyanın mevcut kişisel metni korunur; ortak metne aldığınız kurallar bu
dosyada hem kişisel hem ortak bölümde yer alabilir.

**Add one** veya `Ctrl+N`, tek bir yeni talimatı ortak metnin sonuna ekler; önceki
ortak talimatları silmez. İki kez `Ctrl+S` ile inceleyip bütün algılanan destekli
ajanlara dağıtın. Aynı talimatın tekrar eklenmesi yinelenen kopya oluşturmaz.

Örnek genel talimatlar:

```markdown
- Follow existing project conventions.
- Never log credentials or connection strings.
- Do not claim tests passed unless they actually ran.
- Ask before making a breaking change.
```

Terminal komutları:

```sh
mcpdeck instructions
mcpdeck instructions set --file personal-instructions.md
mcpdeck instructions set 'Never log credentials. Follow existing project conventions.'
mcpdeck instructions show
mcpdeck instructions files
mcpdeck instructions show --agent codex
mcpdeck instructions --agent codex
mcpdeck instructions append 'Never log credentials.'
mcpdeck instructions status
mcpdeck instructions sync
mcpdeck instructions clear --yes
```

Bir ajanın mevcut dosyasını dışarıda düzenleyip geri yazmak için:

```sh
mcpdeck instructions show --agent codex > current-instructions.md
# current-instructions.md içindeki kişisel metni düzenleyin; ortak bloğu koruyun.
mcpdeck instructions set --agent codex --file current-instructions.md
```

Bir ajanın birden fazla genel kural dosyası varsa `instructions files` listesindeki
yolu `--agent cursor --path /tam/genel/kural/dosyası.mdc` ile seçebilirsiniz.
Proje veya listede olmayan rastgele bir dosya bu yolla düzenlenemez.

`set` merkezdeki metni değiştirir ve algılanan ajanlara dağıtır. Dosyanın veya stdin
girdisinin içeriği düz UTF-8 metindir (`--file -` stdin okur). Üst sınır 24 KiB'dir.
Metin bir planlayıcıya/API'ye gönderilmez. Merkez kayıt, `deck.json` yanında
`instructions.md` dosyasıdır; MCP deck formatı değişmez.

## Global hedefler

| Ajan | Kullanıcı düzeyindeki hedef | Not |
| --- | --- | --- |
| Codex | `$CODEX_HOME/AGENTS.md`, varsayılan `~/.codex/AGENTS.md` | `AGENTS.override.md` mevcutsa etkin olan bu dosyaya bölüm eklenir |
| Claude Code | `~/.claude/CLAUDE.md` | Varsayılan kullanıcı dizini |
| Copilot CLI | `$COPILOT_HOME/copilot-instructions.md`, varsayılan `~/.copilot/copilot-instructions.md` | Özel Copilot dizini desteklenir |
| Copilot VS Code | `~/.copilot/copilot-instructions.md` | Copilot Agent Host içindir; eski Local ajan modunda kullanıcı talimatlarını uygulama içinden ekleyin |
| Gemini CLI, Antigravity | `~/.gemini/GEMINI.md` | Ortak hedef tek kez yazılır |
| Qwen Code | `~/.qwen/QWEN.md` | Kişisel global bellek |
| OpenCode | `~/.config/opencode/AGENTS.md` | Varsayılan global dizin |
| Cursor | `~/.cursor/rules/mcpdeck-global.mdc` | Yerel kullanıcı kuralı; `alwaysApply: true` |
| Windsurf | `~/.codeium/windsurf/memories/global_rules.md` | Mevcut içerikle toplam 6.000 karakter sınırı korunur |
| Kiro | `~/.kiro/steering/mcpdeck-global.md` | `inclusion: always` |
| Cline CLI | `~/.cline/rules/mcpdeck-global.md` | Global kural |
| Algılanan Cline eklentileri | `~/Documents/Cline/Rules/mcpdeck-global.md` | Ortak global kural dizini |

Claude Desktop, Zed, Roo Code, Continue, Amazon Q ve Trae için bu sürümde
doğrulanmış otomatik global adapter yoktur. Sonuçta `manual` gösterilir.
Editörde **Copy** veya `Ctrl+Y` ile merkezi metni kopyalayıp ajanın kişisel talimat
ayarına yapıştırabilirsiniz. Clipboard terminalin OSC 52 desteğine bağlıdır.
MCP desteği ile global talimat desteği ayrı özelliklerdir.

## Mevcut ve proje talimatları

Dağıtım, mevcut kişisel dosyanın içine `mcpdeck:global-instructions` başlangıç ve
bitiş işaretleriyle bir bölüm ekler. Sonraki güncellemeler yalnızca bu bölümü
değiştirir. `clear` yalnızca bu bölümü kaldırır; kullanıcı metni korunur. Her
değişiklikten önce mevcut dosya `.mcpdeck-backup` olarak yedeklenir. Yedek en son
değişiklik öncesi sürümdür.

Proje kökleri, proje `AGENTS.md`, `CLAUDE.md`, `.github` ve proje kural dizinleri
taranmaz veya değiştirilmez. Global dizin proje içine symlink ile yönlendirilmişse
dağıtım hata verir. Genel ve proje talimatlarının birleştirilmesi ve önceliği ajan
tarafından belirlenir; tüm ajanlarda aynı çatışma önceliği garanti edilmez.

`status` dosya dağıtımını doğrular; modelin talimatlara gerçekten uyduğunu test
etmez. Yeni bir ajan oturumu açın. Gemini için `/memory reload` ve `/memory show`,
Copilot CLI için `/instructions` ile yüklenen kaynakları kontrol edebilirsiniz.
Ajanların kendi bağlam ve dosya boyutu sınırları ayrıca geçerlidir; genel metni
kısa tutun. Proje talimatlarıyla çelişen genel talimatları kaynak metinde düzeltin.

## Hatalar ve tekrar deneme

Bir ajanın hedefi bozuk, yazılamıyor veya boyut sınırını aşıyorsa o ajanda `failed`
görünür; diğer ajanlar işlemeye devam eder. Merkezi metin tekrar deneme için
saklanır. Sorunu çözdükten sonra `mcpdeck instructions sync` kullanın. İçerik başka
oturumda değişmişse açık editör eski metni üzerine yazamaz; editörü yeniden açın.
Algılanmayan ajanlar `not detected` gösterilir. Bilinçli olarak tanımlı, kurulu
olmayan hedefleri de oluşturmak için `--all-profiles` kullanılabilir.

## Tekrarlanabilir testler

Kaynak koddan derlenen binary ile Mac/Linux üzerinde:

```sh
go test ./...
go test -race ./internal/instructions ./internal/tui
go vet ./...
go build -o mcpdeck .
python3 tests/instructions_smoke.py ./mcpdeck
```

Binary/terminal testi geçici HOME kullanır; gerçek kişisel veya proje talimatlarını
değiştirmez. Fare/klavye akışı, Unicode, dosya koruma, güncelleme, kaldırma ve
kısmi hata sonrası tekrar deneme kontrol edilir. Unix CI'de de çalışır.

Kurulu istemci ve mevcut hesapla gerçek modelin talimat yüklemesini doğrulamak için:

```sh
python3 tests/live_instructions_smoke.py ./mcpdeck codex
python3 tests/live_instructions_smoke.py ./mcpdeck copilot
```

Bu isteğe bağlı testler her ajan için üç yeni oturum açar ve sağlayıcı hesabı
kullanımı oluşturur. Codex dosya tabanlı girişini geçici özel dizine kopyalar;
Copilot aktif GitHub CLI girişini bellekte kullanır. Gerçek talimat/ayar dosyaları
değişmez. Yeni ve güncellenmiş rastgele değerin yüklendiği, kaldırılan değerin
artık yüklenmediği doğrulanır. Diğer ajanların davranışı ayrıca test edilmelidir.

## Kaynaklar ve doğrulama sınırı

Hedefler resmî belgelerden alınmıştır:
[Codex](https://learn.chatgpt.com/docs/agent-configuration/agents-md),
[Claude Code](https://code.claude.com/docs/en/memory),
[Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions),
[VS Code](https://code.visualstudio.com/docs/agent-customization/custom-instructions),
[Gemini CLI](https://geminicli.com/docs/cli/gemini-md/),
[Antigravity](https://www.antigravity.google/docs/rules),
[Qwen Code](https://qwenlm.github.io/qwen-code-docs/en/users/features/memory/),
[OpenCode](https://opencode.ai/docs/rules/),
[Cursor](https://prod.cursor.com/help/customization/rules),
[Windsurf](https://docs.devin.ai/desktop/cascade/memories),
[Kiro](https://kiro.dev/docs/steering/),
[Cline](https://docs.cline.bot/customization/cline-rules).

Dosya dağıtımı, mevcut metnin korunması, proje dosyalarına dokunulmaması,
güncelleme/kaldırma, paylaşılan hedefler, sınırlı boyut, symlink ve eşzamanlı editör
kontrolleri yerel fixture testleriyle doğrulanır. Gerçek ajan sürümlerinin bu
dosyaları yüklemesi ayrıca ajan içinde kontrol edilmelidir.
