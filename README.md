# Wakupi

<h4 align="center">
  WhatsApp Desktop Client — AI Playground, AI Image Gen, Stock Market & Desktop Control
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
- Status/Story viewer & poster (text + image)
- Group management (create, add/remove, promote/demote)
- Chat actions: pin, archive, mute, block, star, search, forward
- Markdown support in messages
- Message reactions & delete

### 🤖 AI Playground
- **Multi-provider chat**: OpenAI, Anthropic (Claude), Google (Gemini), Ollama (local)
- **AI Image Generation**: 3 providers — GamAPI (14 free models!), OpenAI DALL-E, Gemini Imagen
- **Multi-tab**: Chat, Image, and Market tabs in one workspace
- **Session management**: Multiple conversations, persistent history
- **Streaming responses**: Real-time typing with Markdown rendering
- **Send to WhatsApp**: Push AI responses or generated images directly to any chat
- Smart reply suggestions & message summarization
- Per-session parameter overrides (model, temperature, system prompt)
- Code blocks with syntax highlighting (via highlight.js)

### 📈 Stock Market Dashboard
- **Real-time IHSG stocks**: BBCA, TLKM, BMRI, ASII, and all IDX stocks with `.JK` suffix
- **Cryptocurrency**: BTC, ETH, and global crypto pairs
- **Interactive candlestick chart**: Powered by TradingView's `lightweight-charts`
- **Custom watchlist**: Add/remove your own symbols
- **Auto-refresh**: Quotes update every 30 seconds during market hours
- **Multiple timeframes**: 1 day (5-min candles), 5 days, 1 month, 3 months, 6 months, 1 year
- Powered by Yahoo Finance — no API key required

### 💳 Universal QRIS Payments
- Supports **all Indonesian QRIS providers** — ShopeePay, DANA, OVO, GoPay, LinkAja, BSI, etc.
- Convert any static QRIS into dynamic with custom amounts
- Perfect for merchants, cashiers, online sellers, and small businesses
- Send QRIS payment QR directly to customers via WhatsApp
- Product catalog with prices for quick check-out
- Transaction history and daily sales dashboard
- Real-time invoice generator in any chat

### 🖥️ Desktop Controller
- View running apps & quick launch
- Media controls (Play/Pause, Next, Previous, Now Playing)
- Volume control slider
- Screenshot capture & screen lock
- **WhatsApp remote control** — send `!open terminal`, `!play`, `!volume 80`, etc. from another phone

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

## ⌨️ Desktop Commands (WhatsApp Remote)

| Command | Action |
|---------|--------|
| `!open terminal` | Launch application |
| `!close firefox` | Close application |
| `!apps` | List running apps |
| `!play` / `!pause` | Play/Pause media |
| `!next` / `!prev` | Skip track |
| `!now` | Show current track |
| `!volume 80` | Set volume (0-100) |
| `!screenshot` | Capture screenshot |
| `!lock` | Lock screen |
| `!help` | Show all commands |

---

## 🎨 AI Image Generation

| Provider | Models | Notes |
|----------|--------|-------|
| **GamAPI** | 14 (Imagen, Ideogram, Flux, Playground) | Free, unlimited quota |
| **OpenAI** | DALL-E 3 | Requires API key |
| **Gemini** | Imagen | Requires API key |

GamAPI styles: illustration, photo, abstract, 3d, line-art, custom
GamAPI ratios: square (1:1), portrait (9:16), landscape (16:9), 4:3, 3:4, 4:5

---

## 📈 Supported Market Symbols

| Type | Pattern | Examples |
|------|---------|----------|
| IDX Stocks | `SYMBOL.JK` | BBCA.JK, TLKM.JK, GOTO.JK, ASII.JK |
| US Stocks | `SYMBOL` | AAPL, TSLA, MSFT, GOOGL |
| Crypto | `SYMBOL-USD` | BTC-USD, ETH-USD, SOL-USD |

---

## 🌍 Platform Support

| Feature | Linux | Windows |
|---------|-------|---------|
| WhatsApp Multi-Account | ✅ | ✅ |
| AI Chat (multi-provider) | ✅ | ✅ |
| AI Image Generation | ✅ | ✅ |
| Stock Market Dashboard | ✅ | ✅ |
| QRIS Generator | ✅ | ✅ |
| Desktop Controller | ✅ D-Bus | ✅ PowerShell |
| Media Controls | ✅ MPRIS2 | ✅ SMTC |

> macOS support planned for future releases.

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| **Desktop Shell** | [Wails v2](https://wails.io) |
| **Backend** | Go 1.25 |
| **Frontend** | Vue 3 + TypeScript + Pinia |
| **Styling** | Tailwind CSS |
| **Charts** | [lightweight-charts](https://github.com/tradingview/lightweight-charts) (TradingView) |
| **WhatsApp** | [whatsmeow](https://github.com/tulir/whatsmeow) |
| **AI Chat** | OpenAI / Anthropic / Gemini / Ollama |
| **AI Images** | GamAPI / OpenAI DALL-E / Gemini Imagen |
| **Market Data** | Yahoo Finance (unofficial) |
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
│   │   └── image.go        #   Image gen: GamAPI, DALL-E, Imagen
│   ├── desktop/            # Desktop controller (Linux+Windows)
│   ├── market/             # Stock market service (Yahoo Finance)
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
│       │   └── playground/ #   AI Playground (Chat, Image, Market)
│       ├── stores/         # Pinia stores (chat, ai, market, playground, etc.)
│       ├── lib/            # Shared utilities
│       └── style.css       # Tailwind + Markdown styles
├── scripts/                # Test & exploration scripts
│   ├── test-gamapi.sh      #   Bash: full GamAPI exploration
│   ├── test-gamapi.go      #   Go: GamAPI API test
│   └── test-image-gen.go   #   Go: multi-provider image gen test
└── data/                   # Local WhatsApp data & media
```

---

## 🧪 Test Scripts

```bash
# Test GamAPI (all endpoints + generate an image)
go run scripts/test-gamapi.go

# Test image generation through the Go service layer
go run scripts/test-image-gen.go gamapi    # GamAPI
go run scripts/test-image-gen.go openai    # DALL-E (needs OPENAI_API_KEY)
go run scripts/test-image-gen.go gemini    # Imagen (needs GEMINI_API_KEY)

# Bash alternative
bash scripts/test-gamapi.sh
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
