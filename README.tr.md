# MCPDeck

[English](README.md) · [Dokümantasyon](docs/README.md) · [Sürümler](https://github.com/altanmehmet/homebrew-mcpdeck/releases)

MCP sunucularını ve ajanların genel kişisel talimatlarını tek terminalden yöneten açık kaynak Go CLI. **Public alpha**; macOS, Linux ve Windows'ta çalışır.

## Kurulum

macOS / Linux:

```sh
brew install altanmehmet/mcpdeck/mcpdeck
mcpdeck
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/altanmehmet/homebrew-mcpdeck/main/install.ps1 | iex
mcpdeck
```

Kurucu sabitlenmiş paketin SHA-256 özetini kontrol eder ve kullanıcı PATH'ini günceller. Go, WSL veya yönetici yetkisi gerekmez. [Kurucu kaynağı](https://github.com/altanmehmet/homebrew-mcpdeck/blob/main/install.ps1) · [Manuel Windows kurulumu](docs/WINDOWS.md).

## İlk kullanım

1. `mcpdeck` açın; **New MCP** (`n`) ile bir MCP adı veya ihtiyacınızı yazın.
2. Kurulu planlayıcıyı ve modeli seçin. Codex, Claude Code, Gemini CLI, Copilot CLI ve Antigravity CLI (`agy`) desteklenir; API sağlayıcısı da seçilebilir.
3. Kaynakları, çalışacak komutları ve hedef ajanları inceleyin.
4. Kasa için Yeni MCP ekranında `Ctrl+K` ile **OS VAULT** seçeneğini açın (alpha.8+).
5. Planı onaylayın; özel değerleri yalnızca ayrı, maskeli alana girin. MCPDeck kurulumu ve protokol testini yapar, başarılıysa ayarları dağıtır.
6. Ajanları yeniden yükleyin. Hesap/veritabanı erişimi ayrıca gerçek bir araç çağrısıyla doğrulanmalıdır.

```sh
mcpdeck install "Bağlanacağım servis için MCP kur" --vault
mcpdeck instructions append 'Never log credentials.'
mcpdeck instructions files
mcpdeck repair <mcp-adi>
mcpdeck sync --retry-failed
```

**Instructions** (`g`) mevcut genel dosyaları gösterir ve ortak talimatları dağıtır. Proje kuralları bu ortak metnin kapsamına dahil edilmez. [Talimat rehberi](docs/INSTRUCTIONS.md).

## Güvenlik ve sınırlar

Maskeli alanlardaki değerler kurulum ajanının promptuna veya sohbete eklenmez. `--vault`, değerleri OS kasasında saklar; ajan ayarlarına yerel çalıştırıcı yazılır. Normal sohbet alanına sır yazmayın. [Kasa rehberi](docs/VAULT.md).

MCPDeck OS sandbox değildir; onaylanan komutlar kullanıcı yetkinizle çalışır. Aynı kullanıcı yetkisine sahip sınırsız bir program veya kötü amaçlı MCP'ye karşı tam izolasyon sağlamaz. MCP'nin döndürdüğü araç yanıtları da özel veri içerebilir.

Bu sürüm alpha'dır. Bütün ajan sürümleri, hesaplar ve MCP'ler için canlı test garantisi yoktur. macOS notarization ve Windows Authenticode imzası henüz yoktur; arşiv yayıncı imzaları ve özetler bulunur. [Destek kapsamı](docs/COMPATIBILITY.md) · [Güvenlik politikası](SECURITY.md).

## Devamı

[Başlangıç rehberi](docs/CLI-BASLANGIC.md) · [Kurulum sohbeti](docs/AKILLI-KURULUM.md) · [Kurtarma](docs/RECOVERY.md) · [Sorular ve hatalar](SUPPORT.md) · [Katkıda bulunma](CONTRIBUTING.md)

[MIT lisansı](LICENSE).
