# Wakupi

<h4 align="center">
  WhatsApp Desktop Client — AI Playground & AI Image Gen
</h4>

<p align="center">
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20Windows-blue" alt="Platform">
  <img src="https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/vue-3.x-4FC08D?logo=vue.js" alt="Vue">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

---

## ✨ Features

### 💬 WhatsApp Multi-Account
- Multi-session WhatsApp Web (multi-device) with real-time sync
- Media support: images, video, audio, documents, stickers
- Voice notes recording & playback
- Group management (create, add/remove, promote/demote)
- Chat actions: pin, archive, mute, block, forward
- Markdown support in messages
- Message reactions & delete

### 🤖 AI Playground
- **Multi-provider chat**: OpenAI, Anthropic (Claude), Google (Gemini), Ollama (local)
- **AI Image Generation**: 2 providers — OpenAI DALL-E, Gemini Imagen
- **Multi-tab**: Chat and Image tabs in one workspace
- **Session management**: Multiple conversations, persistent history
- **Streaming responses**: Real-time typing with Markdown rendering
- **Send to WhatsApp**: Push AI responses or generated images directly to any chat
- Smart reply suggestions & message summarization
- Per-session parameter overrides (model, temperature, system prompt)
- Code blocks with syntax highlighting (via highlight.js)

### 💳 Universal QRIS Payments
- Supports **all Indonesian QRIS providers** — ShopeePay, DANA, OVO, GoPay, LinkAja, BSI, etc.
- **Shopee Merchant login** — OTP login (phone + password + OTP), auto session refresh, silent token renewal
- Convert any static QRIS into dynamic with custom amounts
- **Unique amount allocation** — every QR gets a distinct amount so payments can't cross-match
- **Real-time settlement detection** — polls Shopee transaction feed, auto-marks paid/expired
- **Auto-notification** — sends "Pembayaran Berhasil" to the WhatsApp chat when a payment settles
- Invoice shows exact amount to pay + expiry time
- Perfect for merchants, cashiers, online sellers, and small businesses
- Send QRIS payment QR directly to customers via WhatsApp
- Product catalog with prices for quick check-out
- Transaction history and daily sales dashboard
- Real-time invoice generator in any chat

### 🎨 Modern UI
- WhatsApp-inspired design with Light/Dark/System theme
- Tailwind CSS — fully responsive
- Emoji picker, context menus, rich message bubbles
- Playground with collapsible panels

---

## 🚀 Quick Start

### Linux

```bash
git clone https://github.com/hirotomasato/wakupi.git
cd wakupi

# Build
wails build -tags webkit2_41

# Run
./build/bin/wakupi
```

**Requirements:**
- Go 1.25+
- Node.js 18+
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- WebKit2GTK 4.1 (`sudo apt install libwebkit2gtk-4.1-dev`)

### Windows

```bash
git clone https://github.com/hirotomasato/wakupi.git
cd wakupi

# Build
wails build

# Run
./build/bin/wakupi.exe
```

---

## 🎨 AI Image Generation

| Provider | Models | Notes |
|----------|--------|-------|
| **OpenAI** | DALL-E 3 | Requires API key |
| **Gemini** | Imagen | Requires API key |

---

## 🌍 Platform Support

| Feature | Linux | Windows |
|---------|-------|---------|
| WhatsApp Multi-Account | ✅ | ✅ |
| AI Chat (multi-provider) | ✅ | ✅ |
| AI Image Generation | ✅ | ✅ |
| QRIS Generator | ✅ | ✅ |

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| **Desktop Shell** | [Wails v2](https://wails.io) |
| **Backend** | Go 1.26 |
| **Frontend** | Vue 3 + TypeScript + Pinia |
| **Styling** | Tailwind CSS |
| **WhatsApp** | [whatsmeow](https://github.com/tulir/whatsmeow) |
| **AI Chat** | OpenAI / Anthropic / Gemini / Ollama |
| **AI Images** | OpenAI DALL-E / Gemini Imagen |
| **QRIS** | [paygateme](https://github.com/hirotomasato/paygateme) — Shopee merchant SDK |
| **Database** | SQLite |
| **D-Bus** | godbus/v5 |
| **QR Code** | qrcode + jsQR |

---

## 📁 Project Structure

```
wakupi/
├── app.go                  # Wails bindings (Go→JS bridge)
├── main.go                 # App entry point
├── internal/
│   ├── ai/                 # AI: chat + image generation
│   │   ├── service.go      #   Provider dispatch & config
│   │   ├── stream.go       #   SSE streaming for all providers
│   │   └── image.go        #   Image gen: DALL-E, Imagen
│   ├── cs/                 # Customer service bot
│   ├── payment/            # QRIS: Shopee merchant payment
│   │   ├── store.go        #   SQLite payment store
│   │   └── manager.go      #   OTP login, settlement, QRIS
│   └── wa/                 # WhatsApp manager
│       ├── manager.go      #   Session & event routing
│       ├── messages.go     #   Inbound message handling
│       ├── send.go         #   Outbound message sending
│       ├── actions.go      #   Chat actions & groups
│       ├── avatar.go       #   Profile picture caching
│       └── store.go        #   SQLite persistence
├── frontend/
│   └── src/
│       ├── components/     # Vue components
│       │   └── playground/ #   AI Playground
│       ├── stores/         # Pinia stores (chat, ai, playground, etc.)
│       ├── lib/            # Shared utilities
│       └── style.css       # Tailwind + Markdown styles
└── data/                   # Local WhatsApp data & media
```

---

## 🤝 Contributing

Contributions are welcome! Feel free to open issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

---

## 📄 License

MIT © [Masanto](https://github.com/hirotomasato)

See [LICENSE](LICENSE) for full text.

---

## ⚠️ Disclaimer

This project is not affiliated with WhatsApp (Meta). Use at your own risk. WhatsApp may ban accounts that use unofficial clients — always use a secondary number for testing.
