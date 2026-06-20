package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"wakupi/internal/ai"
	"wakupi/internal/cs"
	"wakupi/internal/market"
	"wakupi/internal/wa"
)

type App struct {
	ctx context.Context
	wa  *wa.Manager
	ai  *ai.Service
	cs  *cs.Bot
	imageGen *ai.Service

	aiStreamMu     sync.Mutex
	aiStreamCancel context.CancelFunc
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	mgr, err := wa.NewManager("./data", func(name string, data ...interface{}) {
		runtime.EventsEmit(a.ctx, name, data...)
	})
	if err != nil {
		runtime.LogErrorf(ctx, "wa manager init failed: %v", err)
		return
	}
	a.wa = mgr

	a.ai = ai.New(a.loadAIConfig())
	a.imageGen = ai.New(a.loadImageGenConfig())
	a.cs = cs.New(a.loadCSBotConfig())

	// Wire CS Bot hook into the WhatsApp manager so every incoming
	// text message is forwarded to the bot for auto-reply.
	a.wa.SetCSBotHook(func(s *wa.Session, text string, chatJID types.JID, pushName string) {
		a.cs.HandleMessage(a.ctx, cs.MessageContext{
			ChatJID:  chatJID,
			Text:     text,
			PushName: pushName,
		}, func(ctx context.Context, jid types.JID, reply string) error {
			_, err := s.Client.SendMessage(ctx, jid, &waE2E.Message{
				Conversation: proto.String(reply),
			})
			return err
		})
	})

	if err := mgr.LoadExisting(ctx); err != nil {
		runtime.LogErrorf(ctx, "load existing sessions: %v", err)
	}
}

func (a *App) loadAIConfig() ai.Config {
	if a.wa == nil {
		return ai.Config{}
	}
	raw, _ := a.wa.GetAppSetting(a.ctx, "ai_config")
	var cfg ai.Config
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	return cfg
}

func (a *App) shutdown(ctx context.Context) {
	if a.wa != nil {
		a.wa.Shutdown()
	}
}

// MediaHTTPHandler returns an http.Handler that serves /media files.
// Used by main.go to mount on the asset server.
func (a *App) MediaHTTPHandler() http.Handler {
	if a.wa == nil {
		return http.NotFoundHandler()
	}
	return a.wa.MediaHandler()
}

func (a *App) ListSessions() []wa.SessionInfo {
	if a.wa == nil {
		return []wa.SessionInfo{}
	}
	return a.wa.Sessions()
}

func (a *App) LoadChats(sessionID string) ([]wa.ChatInfo, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.LoadChats(a.ctx, sessionID)
}

func (a *App) LoadMessages(sessionID, jid string, limit int, beforeTS int64) ([]wa.MessageInfo, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.LoadMessages(a.ctx, sessionID, jid, limit, beforeTS)
}

func (a *App) RefreshAvatar(sessionID, jid string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.RefreshAvatar(a.ctx, sessionID, jid)
}

// === Chat flags ===

func (a *App) PinChat(sessionID, jid string, pinned bool) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.PinChat(a.ctx, sessionID, jid, pinned)
}

func (a *App) ArchiveChat(sessionID, jid string, archived bool) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.ArchiveChat(a.ctx, sessionID, jid, archived)
}

func (a *App) MuteChat(sessionID, jid string, until int64) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.MuteChat(a.ctx, sessionID, jid, until)
}

func (a *App) BlockChat(sessionID, jid string, blocked bool) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.BlockChat(a.ctx, sessionID, jid, blocked)
}

// === Star + Search ===

func (a *App) StarMessage(sessionID, jid, messageID string, starred bool) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.StarMessage(a.ctx, sessionID, jid, messageID, starred)
}

func (a *App) ListStarred(sessionID string, limit int) ([]wa.MessageInfo, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.ListStarred(a.ctx, sessionID, limit)
}

func (a *App) SearchMessages(sessionID, query string, limit int) ([]wa.MessageInfo, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.SearchMessages(a.ctx, sessionID, query, limit)
}

// === Forward ===

func (a *App) ForwardMessage(sessionID, fromChatJID, msgID string, toJIDs []string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.ForwardMessage(a.ctx, sessionID, fromChatJID, msgID, toJIDs)
}

// === Contact ===

func (a *App) IsOnWhatsApp(sessionID string, phones []string) ([]wa.ContactCheck, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.IsOnWhatsApp(a.ctx, sessionID, phones)
}

// === Group ===

func (a *App) GetGroupInfo(sessionID, jid string) (*wa.GroupInfo, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.GetGroupInfo(a.ctx, sessionID, jid)
}

func (a *App) LeaveGroup(sessionID, jid string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.LeaveGroup(a.ctx, sessionID, jid)
}

func (a *App) UpdateGroupParticipants(sessionID, jid string, participants []string, action string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.UpdateGroupParticipants(a.ctx, sessionID, jid, participants, action)
}

func (a *App) SetGroupName(sessionID, jid, name string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.SetGroupName(a.ctx, sessionID, jid, name)
}

// === Profile ===

func (a *App) SetSelfStatus(sessionID, status string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.SetSelfStatus(a.ctx, sessionID, status)
}

func (a *App) SetSelfProfilePicture(sessionID, filePath string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.SetSelfProfilePicture(a.ctx, sessionID, filePath)
}

// === AI ===

func (a *App) GetAIConfig() ai.Config {
	if a.ai == nil {
		return ai.Config{}
	}
	return a.ai.Config()
}

func (a *App) SetAIConfig(cfg ai.Config) error {
	if a.wa == nil || a.ai == nil {
		return fmt.Errorf("not ready")
	}
	cfg = a.resolveAIKey(cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	a.ai.Update(cfg)
	return a.wa.SetAppSetting(a.ctx, "ai_config", string(data))
}

// resolveAIKey replaces a masked (or empty) incoming key with the stored one,
// so test/save calls don't probe the provider with "********".
func (a *App) resolveAIKey(cfg ai.Config) ai.Config {
	if cfg.APIKey == "" || strings.HasPrefix(cfg.APIKey, "*") {
		cfg.APIKey = a.loadAIConfig().APIKey
	}
	return cfg
}

// AITestConnection probes the given (possibly unsaved) config and returns the
// provider error if the key/model/endpoint is invalid.
func (a *App) AITestConnection(cfg ai.Config) error {
	cfg = a.resolveAIKey(cfg)
	return ai.New(cfg).Ping(a.ctx)
}

// AIListModels returns the provider's available model IDs for the given config.
func (a *App) AIListModels(cfg ai.Config) ([]string, error) {
	cfg = a.resolveAIKey(cfg)
	return ai.New(cfg).ListModels(a.ctx)
}

func (a *App) AISuggestReplies(contactName, lastMessages string) ([]string, error) {
	if a.ai == nil || !a.ai.Enabled() {
		return nil, nil
	}
	return a.ai.SuggestReplies(a.ctx, contactName, lastMessages)
}

func (a *App) AISummarize(text string) (string, error) {
	if a.ai == nil || !a.ai.Enabled() {
		return "", fmt.Errorf("AI tidak aktif")
	}
	return a.ai.Summarize(a.ctx, text)
}

func (a *App) AICompose(prompt, tone string) (string, error) {
	if a.ai == nil || !a.ai.Enabled() {
		return "", fmt.Errorf("AI tidak aktif")
	}
	sys := "You are a WhatsApp assistant. Write a single message reply in the user's language. Tone: " + tone + ". Output the message only, no quotes, no preamble."
	return a.ai.Chat(a.ctx, sys, prompt)
}

// === AI Image Generation (standalone config) ===

func (a *App) loadImageGenConfig() ai.Config {
	if a.wa == nil {
		return ai.Config{}
	}
	raw, _ := a.wa.GetAppSetting(a.ctx, "imagegen_config")
	var cfg ai.Config
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	return cfg
}

func (a *App) resolveImageGenKey(cfg ai.Config) ai.Config {
	if cfg.APIKey == "" || strings.HasPrefix(cfg.APIKey, "*") {
		cfg.APIKey = a.loadImageGenConfig().APIKey
	}
	return cfg
}

func (a *App) GetImageGenConfig() ai.Config {
	if a.imageGen == nil {
		return ai.Config{}
	}
	return a.imageGen.Config()
}

func (a *App) SetImageGenConfig(cfg ai.Config) error {
	if a.wa == nil || a.imageGen == nil {
		return fmt.Errorf("not ready")
	}
	cfg = a.resolveImageGenKey(cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	a.imageGen.Update(cfg)
	return a.wa.SetAppSetting(a.ctx, "imagegen_config", string(data))
}

func (a *App) ImageGenTestConnection(cfg ai.Config) error {
	cfg = a.resolveImageGenKey(cfg)
	return ai.New(cfg).Ping(a.ctx)
}

func (a *App) AIGenerateImage(prompt string, opts ai.ImageOptions) ([]ai.ImageResult, error) {
	if a.imageGen == nil || !a.imageGen.Enabled() {
		return nil, fmt.Errorf("Image Gen tidak aktif — atur provider di pengaturan Image Generator")
	}
	return a.imageGen.GenerateImage(a.ctx, prompt, opts)
}

func (a *App) AIGetGamAPIModels() ([]string, error) {
	if a.imageGen == nil {
		return nil, fmt.Errorf("Image Gen not ready")
	}
	return a.imageGen.ListGamAPIModels(a.ctx)
}

func (a *App) AIGetGamAPIStyles() (map[string]string, error) {
	if a.imageGen == nil {
		return nil, fmt.Errorf("Image Gen not ready")
	}
	return a.imageGen.ListGamAPIStyles(a.ctx)
}

func (a *App) AIGetGamAPIRatios() (map[string]string, error) {
	if a.imageGen == nil {
		return nil, fmt.Errorf("Image Gen not ready")
	}
	return a.imageGen.ListGamAPIAspectRatios(a.ctx)
}

// === CS Bot ===

func (a *App) loadCSBotConfig() cs.CSConfig {
	if a.wa == nil {
		return cs.CSConfig{}
	}
	raw, _ := a.wa.GetAppSetting(a.ctx, "csbot_config")
	var cfg cs.CSConfig
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = cs.DefaultSystemPrompt
	}
	return cfg
}

func (a *App) resolveCSBotKey(cfg cs.CSConfig) cs.CSConfig {
	if cfg.APIKey == "" || strings.HasPrefix(cfg.APIKey, "*") {
		cfg.APIKey = a.loadCSBotConfig().APIKey
	}
	return cfg
}

func (a *App) GetCSBotConfig() cs.CSConfig {
	if a.cs == nil {
		return cs.CSConfig{}
	}
	return a.cs.Config()
}

func (a *App) SetCSBotConfig(cfg cs.CSConfig) error {
	if a.wa == nil || a.cs == nil {
		return fmt.Errorf("not ready")
	}
	cfg = a.resolveCSBotKey(cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	a.cs.Update(cfg)
	return a.wa.SetAppSetting(a.ctx, "csbot_config", string(data))
}

func (a *App) CSBotTestConnection(cfg cs.CSConfig) error {
	cfg = a.resolveCSBotKey(cfg)
	return cs.New(cfg).Ping(a.ctx)
}

// === Market Data ===

func (a *App) MarketGetQuote(symbol string) (*market.Quote, error) {
	return market.FetchQuote(symbol)
}

func (a *App) MarketGetQuotes(symbols []string) ([]market.Quote, error) {
	return market.FetchMultiQuotes(symbols)
}

func (a *App) MarketGetChart(symbol, rng string) ([]market.OHLC, error) {
	return market.FetchChart(symbol, rng)
}

// PlaygroundMessage mirrors ai.ChatMessage for Wails binding generation.
type PlaygroundMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PlaygroundOptions carries per-session overrides from the playground UI.
type PlaygroundOptions struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	System      string  `json:"system"`
}

// AIChat runs a streaming completion for the playground. Deltas are emitted as
// "ai:chat:delta" events tagged with streamID; completion/errors arrive via
// "ai:chat:done". Returns immediately after kicking off the stream goroutine.
func (a *App) AIChat(streamID string, messages []PlaygroundMessage, opts PlaygroundOptions) error {
	if a.ai == nil || !a.ai.Enabled() {
		return fmt.Errorf("AI tidak aktif")
	}

	msgs := make([]ai.ChatMessage, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, ai.ChatMessage{Role: m.Role, Content: m.Content})
	}

	// Cancel any in-flight stream before starting a new one.
	a.aiStreamMu.Lock()
	if a.aiStreamCancel != nil {
		a.aiStreamCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.aiStreamCancel = cancel
	a.aiStreamMu.Unlock()

	go func() {
		defer func() {
			a.aiStreamMu.Lock()
			if a.aiStreamCancel != nil {
				a.aiStreamCancel()
				a.aiStreamCancel = nil
			}
			a.aiStreamMu.Unlock()
		}()

		err := a.ai.ChatStream(ctx, msgs, ai.ChatOptions{
			Model:       opts.Model,
			Temperature: opts.Temperature,
			System:      opts.System,
		}, func(delta string) {
			runtime.EventsEmit(a.ctx, "ai:chat:delta", map[string]string{
				"id":    streamID,
				"delta": delta,
			})
		})

		done := map[string]interface{}{"id": streamID}
		if err != nil && ctx.Err() == nil {
			done["error"] = err.Error()
		} else if ctx.Err() != nil {
			done["cancelled"] = true
		}
		runtime.EventsEmit(a.ctx, "ai:chat:done", done)
	}()

	return nil
}

// AIChatCancel stops the currently streaming completion, if any.
func (a *App) AIChatCancel() {
	a.aiStreamMu.Lock()
	defer a.aiStreamMu.Unlock()
	if a.aiStreamCancel != nil {
		a.aiStreamCancel()
		a.aiStreamCancel = nil
	}
}

func (a *App) StartLogin(name string) (string, error) {
	if a.wa == nil {
		return "", fmt.Errorf("wa manager not ready")
	}
	return a.wa.StartLogin(a.ctx, name)
}

func (a *App) Logout(sessionID string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.Logout(a.ctx, sessionID)
}

func (a *App) Disconnect(sessionID string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.Disconnect(sessionID)
}

type QuotedArg struct {
	ID          string `json:"id"`
	Participant string `json:"participant"`
	Text        string `json:"text"`
}

func toQuoted(q *QuotedArg) *wa.QuotedRef {
	if q == nil || q.ID == "" {
		return nil
	}
	return &wa.QuotedRef{ID: q.ID, Participant: q.Participant, Text: q.Text}
}

func (a *App) SendText(sessionID, jid, text string, quoted *QuotedArg) (string, error) {
	if a.wa == nil {
		return "", fmt.Errorf("wa manager not ready")
	}
	return a.wa.SendText(a.ctx, sessionID, jid, text, toQuoted(quoted))
}

func (a *App) SendImage(sessionID, jid, filePath, caption string, quoted *QuotedArg) (*wa.SendMediaResult, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.SendImage(a.ctx, sessionID, jid, filePath, caption, toQuoted(quoted))
}

func (a *App) SendVideo(sessionID, jid, filePath, caption string, quoted *QuotedArg) (*wa.SendMediaResult, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.SendVideo(a.ctx, sessionID, jid, filePath, caption, toQuoted(quoted))
}

func (a *App) SendDocument(sessionID, jid, filePath string, quoted *QuotedArg) (*wa.SendMediaResult, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.SendDocument(a.ctx, sessionID, jid, filePath, toQuoted(quoted))
}

func (a *App) SendAudio(sessionID, jid, filePath string, ptt bool, quoted *QuotedArg) (*wa.SendMediaResult, error) {
	if a.wa == nil {
		return nil, fmt.Errorf("wa manager not ready")
	}
	return a.wa.SendAudio(a.ctx, sessionID, jid, filePath, ptt, toQuoted(quoted))
}

func (a *App) DeleteMessage(sessionID, jid, messageID string, forEveryone bool) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.DeleteMessage(a.ctx, sessionID, jid, messageID, forEveryone)
}

func (a *App) ReactMessage(sessionID, jid, messageID, sender, emoji string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.ReactMessage(a.ctx, sessionID, jid, messageID, sender, emoji)
}

func (a *App) PostStatusText(sessionID, text string) (string, error) {
	if a.wa == nil {
		return "", fmt.Errorf("wa manager not ready")
	}
	return a.wa.PostStatusText(a.ctx, sessionID, text)
}

func (a *App) PostStatusImage(sessionID, filePath, caption string) (string, error) {
	if a.wa == nil {
		return "", fmt.Errorf("wa manager not ready")
	}
	return a.wa.PostStatusImage(a.ctx, sessionID, filePath, caption)
}

func (a *App) Notify(title, body string) {
	runtime.EventsEmit(a.ctx, "ui:notify", map[string]string{"title": title, "body": body})
}

func (a *App) WindowMinimize() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) WindowToggleMaximize() {
	if runtime.WindowIsMaximised(a.ctx) {
		runtime.WindowUnmaximise(a.ctx)
	} else {
		runtime.WindowMaximise(a.ctx)
	}
}

func (a *App) WindowHide() {
	runtime.WindowHide(a.ctx)
}

func (a *App) WindowShow() {
	runtime.WindowShow(a.ctx)
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

func (a *App) MarkRead(sessionID, jid, sender string, messageIDs []string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.MarkRead(a.ctx, sessionID, jid, sender, messageIDs)
}

func (a *App) SubscribePresence(sessionID, jid string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.SubscribePresence(a.ctx, jid, sessionID)
}

func (a *App) SendChatPresence(sessionID, jid, state string) error {
	if a.wa == nil {
		return fmt.Errorf("wa manager not ready")
	}
	return a.wa.SendChatPresence(a.ctx, sessionID, jid, state)
}

// PickFile opens an OS file dialog. accept controls filters: "image", "video", "audio", "any".
func (a *App) PickFile(accept string) (string, error) {
	opts := runtime.OpenDialogOptions{Title: "Pilih file"}
	switch accept {
	case "image":
		opts.Filters = []runtime.FileFilter{{DisplayName: "Gambar", Pattern: "*.jpg;*.jpeg;*.png;*.webp;*.gif"}}
	case "video":
		opts.Filters = []runtime.FileFilter{{DisplayName: "Video", Pattern: "*.mp4;*.webm;*.mov"}}
	case "audio":
		opts.Filters = []runtime.FileFilter{{DisplayName: "Audio", Pattern: "*.mp3;*.ogg;*.m4a;*.wav;*.opus"}}
	}
	return runtime.OpenFileDialog(a.ctx, opts)
}

// SaveTempBlob writes a base64-encoded blob (e.g. recorded audio from MediaRecorder)
// to a temp file and returns its path. Used by voice note sending.
// SaveBase64Image writes a base64-encoded image (e.g. from Gemini Imagen) to a
// temp PNG file and returns its path. Used to bridge inline image data into
// WhatsApp media uploads.
func (a *App) SaveBase64Image(b64 string) (string, error) {
	return a.SaveTempBlob(b64, ".png")
}

func (a *App) SaveTempBlob(b64 string, ext string) (string, error) {
	clean := strings.TrimSpace(b64)
	if idx := strings.Index(clean, ","); idx >= 0 {
		clean = clean[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return "", err
	}
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	f, err := os.CreateTemp("", "wakupi-blob-*"+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// CopyToClipboard writes text to the system clipboard, bypassing webview
// restrictions that block navigator.clipboard in webkit2gtk.
func (a *App) CopyToClipboard(text string) error {
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command("wl-copy", "--foreground", "--type", "text/plain")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command("xsel", "--clipboard", "--input")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("tidak ada clipboard tool (install xclip/xsel/wl-clipboard)")
}
