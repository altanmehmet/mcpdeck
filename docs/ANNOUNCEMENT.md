# MCPDeck CLI alpha.8 — paylaşım metinleri

16 Ekim 2026. Aşağıdaki taslaklar kanalına göre ayrı kullanılabilir.
Repo: https://github.com/altanmehmet/mcpdeck
Paketler: https://github.com/altanmehmet/homebrew-mcpdeck/releases/tag/v0.1.0-alpha.8

## LinkedIn — Türkçe

MCPDeck’i açık kaynak olarak yayımladım.

Birden fazla kodlama ajanı kullanırken aynı MCP bağlantısını her ajanda ayrı ayrı yapılandırmak gerekiyor. Genel talimatları güncel tutmak da ayrı bir iş.

MCPDeck, bu ayarları terminalden yönetmek için yazdığım bir Go CLI. MCP’yi adıyla ekleyebiliyor, kurulum planını çalıştırmadan önce inceleyebiliyor ve desteklenen ajanlara dağıtabiliyorsunuz. Ortak talimatları güncellerken mevcut kişisel metin korunuyor; değişen dosyalar yedekleniyor.

macOS ve Linux’ta Homebrew ile kuruluyor. Windows için de kurucu var. Mevcut sürüm 0.1.0-alpha.8. Codex ve Copilot CLI ile gerçek talimat yükleme ve MCP araç çağrısı testleri geçti.

Denemek isteyenler için kod ve kurulum:
https://github.com/altanmehmet/mcpdeck

Özellikle birden fazla ajan kullananların kurulum ve günlük kullanım hakkındaki geri bildirimini merak ediyorum.

## X — English

I released MCPDeck: a Go CLI for managing MCP configs and shared instructions across coding agents. Review install plans, sync settings and keep backups. Available through Homebrew and a Windows installer. MIT, alpha.
https://github.com/altanmehmet/mcpdeck

## Show HN — English

Title:
Show HN: MCPDeck – a CLI for MCP configs and coding-agent instructions

URL:
https://github.com/altanmehmet/mcpdeck

First comment:

I built MCPDeck to manage MCP connections and shared personal instructions across coding agents from a terminal.

You can add an MCP by name, choose a detected planner, inspect its installation commands, and sync the configuration to supported agents. Instruction updates preserve existing personal text. Project rules stay outside the instruction manager. Changes have backups, and failed syncs can be retried.

The CLI is written in Go and available for macOS, Linux and Windows. Homebrew installation:

    brew install altanmehmet/mcpdeck/mcpdeck

This is alpha.8. Codex and Copilot CLI passed live instruction add/update/remove tests and audited MCP tool calls. Other agent adapters have configuration tests, but that does not establish their behavior in every live session. Installation planning uses your selected agent or API provider; it can consume provider usage.

I’d appreciate feedback on the setup flow, especially from people using more than one coding agent. Which parts would you expect the tool to handle, and which would you prefer to configure yourself?

## Reddit — English

Title:
I built a Go CLI to manage MCP configs and shared instructions across coding agents

Post:

MCPDeck is an open-source CLI I’m working on for managing MCP connections and general personal instructions across coding agents.

It has a terminal interface for adding, enabling and removing MCPs, reviewing installation plans, and syncing supported agent configurations. You can also distribute shared instructions while retaining existing personal text. Project instruction files are excluded, and configuration changes have backups.

The current release is alpha.8. It’s available through Homebrew on macOS/Linux and an installer on Windows. Codex and Copilot CLI have passed real instruction-loading and MCP tool-call tests. The remaining adapters have configuration coverage, with live client testing still needed.

Code and installation: https://github.com/altanmehmet/mcpdeck

If you use several agents, I’d like to hear whether this fits your workflow. Feedback on confusing setup steps or configurations it fails to preserve would be useful.

## DEV — English

Title:
Managing MCP configurations across coding agents with MCPDeck

Opening:

Using several coding agents means keeping several MCP configurations up to date. Shared personal instructions add another set of files to maintain.

MCPDeck is a Go CLI for managing those connections and instructions from a terminal. Its current alpha release supports reviewed installation plans, configuration synchronization, backups and recovery. This walkthrough shows how to install it, add an MCP and distribute a shared instruction.

Suggested article structure — complete with actual captures before publishing:

1. Install with Homebrew or the Windows installer; show `mcpdeck --version`.
2. Open `mcpdeck`, select New MCP, enter a name and choose a detected planner.
3. Show the reviewed commands and target agents before installation.
4. Open Instructions, add a short general rule, review targets and save.
5. Start a new agent session and check the loaded instructions.
6. Explain the current test coverage: live Codex/Copilot tests versus configuration-only coverage for other adapters.

Source and installation: https://github.com/altanmehmet/mcpdeck

## Product Hunt — English

Name:
MCPDeck

Tagline:
Manage MCP configs and agent instructions from your terminal

Description:
MCPDeck is an open-source Go CLI for managing MCP connections and shared personal instructions across coding agents. Review installation plans, sync supported configurations, and recover from backups. Available through Homebrew on macOS/Linux and a Windows installer. Current release: alpha.8.

Maker’s first comment:

Hi, I’m the maker of MCPDeck.

It puts MCP configuration and shared personal instructions in one terminal interface. You can add an MCP by name, review the installation plan, select its target agents and distribute shared guidance while retaining existing personal text.

It’s MIT licensed and currently alpha.8. Real instruction-loading and MCP tool-call tests passed with Codex and Copilot CLI; other clients still need live validation. Installation planning uses your selected agent or API provider.

I’m looking for feedback from people who use multiple coding agents. If you try it, I’d like to know where setup is unclear or where it needs more control.

Product URL:
https://github.com/altanmehmet/mcpdeck

## Demo çekimi — 30–45 saniye

Bu bölüm paylaşım metni değildir; gerçek terminal kaydı için çekim sırasıdır.
İzole örnek dosyalar kullanın; kişisel talimatları veya bağlantı değerlerini göstermeyin.

- 0–5 sn: `mcpdeck --version`, ardından terminal paneli.
- 5–15 sn: New MCP → sunucu adı → algılanan planlayıcı seçimi.
- 15–25 sn: Gerçek planın komutları ve hedef ajanları. Kurulum beklemelerini kesebilirsiniz; inceleme adımını atlamayın.
- 25–35 sn: Instructions → kısa bir genel talimat → hedef inceleme → kayıt.
- 35–45 sn: Kaydedilmiş talimatı yeniden açın. Son karede repo adresi ve Homebrew kurulum komutu.

İki Product Hunt görseli için terminal paneli ve talimat hedef incelemesi kullanılabilir.
Bir adım gerçek kayıtta tamamlanmadıysa başarılı olmuş gibi göstermeyin.
