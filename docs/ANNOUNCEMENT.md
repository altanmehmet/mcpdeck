# Alpha announcement drafts

## LinkedIn / developer forums — Turkish

MCPDeck'i MIT lisansıyla open source olarak yayınladım.

Birden fazla kodlama ajanı kullanırken MCP ayarlarını ve ortak kişisel talimatları
ayrı ayrı yönetmek yorucuydu. MCPDeck bunları fare ve klavye destekli bir terminal
arayüzünde topluyor:

- MCP'leri keşfetme, incelenen planlarla kurma, açma/kapatma ve kaldırma
- Desteklenen ajanların ayarlarını yedekleyerek senkronize etme
- Mevcut genel talimatları görme ve tek ortak metni ajanlara dağıtma
- Hatalı senkronizasyonu yeniden deneme ve yedekten geri dönme

macOS, Linux ve Windows için alpha paketleri var. Henüz stabil sürüm değil:
OAuth, bazı ajanlar ve özel kurulumlar ek kullanıcı adımları gerektirebilir.

Repo ve kurulum: https://github.com/altanmehmet/mcpdeck

Deneyip hangi ajan/MCP kombinasyonunda sorun yaşadığınızı bir issue ile
paylaşırsanız bir sonraki sürümü buna göre geliştirebilirim. Lütfen parola,
token veya kişisel yapılandırma dosyası paylaşmayın.

## English

I released MCPDeck as an MIT-licensed open source project.

It brings MCP configuration and shared personal instructions for coding agents
into one terminal interface, with mouse and keyboard controls. You can discover
MCPs, review agent-generated installation plans, sync supported client settings,
enable/disable/remove servers, and manage global instructions with backups.

Alpha packages are available for macOS, Linux and Windows. This is an early
release: OAuth, client-specific behavior and specialized installations can
require additional setup. It does not promise unattended installation of every MCP.

Source and installation: https://github.com/altanmehmet/mcpdeck

Feedback and reproducible issues are welcome, especially from people using
several agents. Please exclude secrets and personal configuration from reports.
