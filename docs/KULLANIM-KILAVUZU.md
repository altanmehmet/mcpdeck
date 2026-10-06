# MCPDeck — Kullanım kılavuzu

MCPDeck, macOS ve Linux için terminal tabanlı bir MCP yöneticisidir.
Web servisi veya veritabanı gerektirmez. Geliştirme sürümüdür; kayıtlı ayar,
canlı bağlantı ve gerçek araç erişimi farklı şeylerdir.

## 1. Kurulum ve ilk açılış

Kaynak koddan, klonladığınız proje klasöründe Go 1.24+ ile:

```sh
go build -o mcpdeck .
sh scripts/install.sh
~/.local/bin/mcpdeck
```

`~/.local/bin` PATH içinde olmalıdır. Zsh kullanıyorsanız aşağıdaki satırı
`~/.zshrc` dosyanıza ekleyip yeni terminal açın:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Sonraki açılışlarda `mcpdeck` yeterlidir. Derlenmiş dağıtım paketini
kullanıyorsanız Go gerekmez; pakette `sh install.sh` çalıştırın.
MCP sunucularının Node, Java, Docker gibi kendi gereksinimleri olabilir.
Henüz yayınlanmış kararlı paket veya Homebrew tap'i yoktur.

## 2. Yeni MCP ekleme

1. **+ New MCP** düğmesine tıklayın veya `n` tuşuna basın.
2. MCP adını girin: örneğin `oracle-mcp`.
3. İstediğinizi doğal dille anlatın. Örneğin:
   “Oracle Database için resmi MCP'yi kur; mevcut uyumlu Java'yı kullan.
   Bağlantı bilgilerini gizli girişte sor.”
4. **Ctrl+S** ile planlayıcıyı başlatın; sağlayıcı seçimini kontrol edin.
5. Ajanın kaynaklarını, komutlarını, gereksinimlerini ve dağıtım hedeflerini inceleyin.
6. `You:` alanında soru sorun veya düzeltme isteyin. `/refresh` yazılım
   envanterini yeniler; Enter veya `/install` onaya, `/cancel` iptale gider.
7. Son komut planını `y` + Enter ile onaylayın. Şifre ve tokenları yalnızca
   gizli giriş alanına yazın; doğal dil isteğine veya sohbete koymayın.
8. MCPDeck kurulumdan sonra `initialize` ve `tools/list` çağrılarını yapar.
   Başarıdan sonra hedef ajanların ayarlarını günceller ve sonuçları gösterir.
9. Ajanın MCP bağlantısını yenileyip o ajanda gerçek bir araç çağrısı yapın.

URL zorunlu değildir. Belirli bir resmi MCP'yi tercih ediyorsanız istek içinde
belge bağlantısı verebilirsiniz. OAuth, veritabanı hesabı veya lisans gibi dış
koşullar tamamen otomatik çözülemez; gereken işlem açıklanır.

Aynı akışın terminal karşılığı:

```sh
mcpdeck install 'Oracle Database için resmi MCP kurulumunu hazırla'
mcpdeck planner list
mcpdeck planner set --provider codex
mcpdeck environment
```

Kurulum ve onarım için bağlı ajan sohbeti vardır. MCPDeck genel amaçlı bir
sohbet istemcisi değildir. Sağlayıcı kullanım ücretleri hesabınıza yansıyabilir.
[Sağlayıcılar ve kurulum sınırları](AKILLI-KURULUM.md).

## 3. Mevcut ayarı içe aktarma

Hazır MCP JSON/TOML ayarını doğal dil kurulumundan bağımsız içe aktarabilirsiniz:

```sh
mcpdeck import --preview
mcpdeck import --help
mcpdeck add --help
```

Önizlemeyi inceleyin. Ortak modelin desteklemediği özel alanlar reddedilir.
Mevcut agent ayarlarınızdaki MCP'ler panelde `external` olarak görünebilir;
bu, MCPDeck tarafından kuruldukları veya bağlantılarının çalıştığı anlamına gelmez.

## 4. Günlük kullanım

Soldaki ajan listesi hedef uygulamayı, ortadaki liste MCP'leri gösterir.
Fareyle satır seçebilir, kutudan açıp kapatabilir ve tekerlekle gezinebilirsiniz.
`●` algılanan ajanı, `○` algılanmayan temel profili belirtir. Desteklenen tüm
ajanlar her bilgisayarda kurulu kabul edilmez. Hedefleri kurulum planında kontrol edin.

| Varsayılan tuş | İşlem |
| --- | --- |
| Tab | Sonraki ajan |
| ↑ / ↓ | MCP seçimi |
| Space | Seçili ajanda aç/kapat |
| e / d | Tüm kayıtlı profillerde aç/kapat |
| b | Seçili profilin Direct/Bridge modu |
| s | Tüm profilleri senkronize et |
| Ctrl+R | Başarısız senkronizasyon hedeflerini tekrar dene |
| n / i | Yeni MCP / kurulum asistanı |
| u | Seçili MCP için onarım sohbeti |
| x | MCP'yi kaldır; ayrıca onay ister |
| g | Genel talimatları düzenle |
| r | Dosyalardan yeniden oku |
| v | Fareyle terminal metni seçimi aç/kapat |
| a / ? / q | Tuş atama / yardım / çıkış |

Tuşlar yeniden atanabilir. `a` veya **Help → Rebind** üzerinden işlem seçip
Enter'a basın ve tuşu kaydedin. Aynı tuş iki işleme atanamaz; Ctrl+C çıkışa ayrılır.
Atamalar `~/.config/mcpdeck/ui.json` dosyasındadır.
En az 64×18 terminal gerekir; dar terminalde bazı düğmeler görünmeyebilir.
`mcpdeck --no-mouse` terminalin normal metin seçimini korur.

## 5. Hata, onarım ve geri yükleme

```sh
mcpdeck repair oracle-mcp 'tools/list zaman aşımına uğruyor'
mcpdeck sync status
mcpdeck sync --retry-failed
mcpdeck sync restore --profile cursor
# Yedeği yerel olarak inceledikten sonra:
mcpdeck sync restore --profile cursor --yes
```

`update`, `repair` komutunun diğer adıdır. Harici kayıt önce içe aktarılmalıdır.
Başarısız kurulum doğrulaması yeni bağlantıyı ajanlara dağıtmaz.
Dağıtımda bazı hedefler başarısız olursa çalışan hedefler korunur; yalnızca
başarısız olanları yeniden deneyebilirsiniz.

Geri yükleme ajan dosyasının tamamını değiştirir. Deck seçimleri değişmez;
sonraki sync onları tekrar uygular. Yedekler gizli bilgiler içerebilir.
[Yedekleme ve eşzamanlı düzenleme sınırları](RECOVERY.md).

## 6. Genel talimatlar

**Instructions** veya `g` ortak editörü açar. İlk Ctrl+S dağıtımı önizler,
ikincisi uygular. **Existing / Ctrl+E** mevcut genel talimat dosyalarını açar.
**Use for all / Ctrl+F** seçili metni ortak taslağa taşır; önce inceleyin.
**Add one / Ctrl+N** önceki metni koruyarak tek talimat ekler.
Proje kuralları bu genel dağıtımın dışındadır; bazı ajanlar manuel işlem gerektirir.

```sh
mcpdeck instructions append 'Never log credentials.'
mcpdeck instructions show --agent codex
mcpdeck instructions status
```

[Desteklenen talimat hedefleri](INSTRUCTIONS.md).

## 7. Kendi testiniz

1. Önce hassas olmayan bir MCP veya geçici dosya klasörüyle başlayın.
2. Yeni MCP akışında hedef listesini ve kurulum komutlarını inceleyin.
3. Sonuçta hedef başına başarı veya hata gördüğünüzü doğrulayın.
4. Kullandığınız ajanı yenileyip gerçek araç çağrısı yaptırın; sonuç kaydını inceleyin.
5. MCPDeck'te kapatın, ajanı yenileyin ve aracın artık kullanılamadığını kontrol edin.
6. Tekrar açıp aynı araç çağrısını deneyin.
7. **Remove / x** ile kaldırıp ajan ayarında girdinin silindiğini doğrulayın.

`initialize` ve `tools/list` başarısı Oracle hesabının, sorgu yetkisinin veya
Sentry/Figma OAuth erişiminin çalıştığını kanıtlamaz. Bunlar gerçek araç çağrısıyla
ayrıca doğrulanmalıdır. [Test kayıtları ve denenmeyenler](TEST-SONUCLARI.md).

Varsayılan deck yolu `~/.config/mcpdeck/deck.json`.
Ayrı deneme ayarı için `mcpdeck --config /absolute/path/test-deck.json` kullanın;
ajan hedeflerini de geçici dosyalara ayarlamadan gerçek ajan dosyaları etkilenebilir.
