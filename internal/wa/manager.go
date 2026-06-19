package wa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "modernc.org/sqlite"
)

type EventEmitter func(name string, data ...interface{})

// CSBotHook is called for every incoming non-self, non-status message.
// It runs in a goroutine so the message pipeline is not blocked.
type CSBotHook func(s *Session, text string, chatJID types.JID, pushName string)

type Manager struct {
	mu        sync.RWMutex
	emit      EventEmitter
	dbDir     string
	mediaDir  string
	container *sqlstore.Container
	store     *Store
	clients   map[string]*Session
	avatarMu  sync.Mutex
	avatarReq map[string]bool
	csHook    CSBotHook
}

// SetCSBotHook registers a hook that fires for every incoming non-self
// text message. The hook runs in a goroutine and does not block the
// message pipeline.
func (m *Manager) SetCSBotHook(hook CSBotHook) {
	m.csHook = hook
}

type Session struct {
	ID        string
	Name      string
	Client    *whatsmeow.Client
	Connected bool
	JID       string
	Phone     string
}

type SessionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	JID       string `json:"jid"`
	Phone     string `json:"phone"`
}

type ChatInfo struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	JID         string `json:"jid"`
	Name        string `json:"name"`
	IsGroup     bool   `json:"isGroup"`
	LastMessage string `json:"lastMessage"`
	LastTime    int64  `json:"lastTime"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Pinned      bool   `json:"pinned"`
	Archived    bool   `json:"archived"`
	MutedUntil  int64  `json:"mutedUntil"`
	Blocked     bool   `json:"blocked"`
}

type MessageInfo struct {
	ID         string  `json:"id"`
	ChatID     string  `json:"chatId"`
	AccountID  string  `json:"accountId"`
	JID        string  `json:"jid"`
	Sender     string  `json:"sender"`
	Text       string  `json:"text"`
	Timestamp  int64   `json:"timestamp"`
	FromMe     bool    `json:"fromMe"`
	IsGroup    bool    `json:"isGroup"`
	PushName   string  `json:"pushName"`
	MediaType  string  `json:"mediaType,omitempty"`
	MediaURL   string  `json:"mediaUrl,omitempty"`
	MimeType   string  `json:"mimeType,omitempty"`
	FileName   string  `json:"fileName,omitempty"`
	FileSize   uint64  `json:"fileSize,omitempty"`
	Width      uint32  `json:"width,omitempty"`
	Height     uint32  `json:"height,omitempty"`
	Duration   uint32  `json:"duration,omitempty"`
	IsPTT      bool    `json:"isPtt,omitempty"`
	Caption    string  `json:"caption,omitempty"`
	QuotedID   string  `json:"quotedId,omitempty"`
	QuotedText string  `json:"quotedText,omitempty"`
	QuotedFrom string  `json:"quotedFrom,omitempty"`
}

type ReceiptInfo struct {
	AccountID  string   `json:"accountId"`
	JID        string   `json:"jid"`
	Sender     string   `json:"sender"`
	MessageIDs []string `json:"messageIds"`
	Type       string   `json:"type"`
	Timestamp  int64    `json:"timestamp"`
}

type PresenceInfo struct {
	AccountID string `json:"accountId"`
	JID       string `json:"jid"`
	Online    bool   `json:"online"`
	LastSeen  int64  `json:"lastSeen"`
}

type ChatPresenceInfo struct {
	AccountID string `json:"accountId"`
	JID       string `json:"jid"`
	State     string `json:"state"`
	Media     string `json:"media"`
}

type ReactionInfo struct {
	AccountID string `json:"accountId"`
	JID       string `json:"jid"`
	MessageID string `json:"messageId"`
	Sender    string `json:"sender"`
	FromMe    bool   `json:"fromMe"`
	Emoji     string `json:"emoji"`
	Timestamp int64  `json:"timestamp"`
}

type DeletedInfo struct {
	AccountID string `json:"accountId"`
	JID       string `json:"jid"`
	MessageID string `json:"messageId"`
	Sender    string `json:"sender"`
}

func NewManager(dataDir string, emit EventEmitter) (*Manager, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	mediaDir := filepath.Join(dataDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "wakupi.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-20000)", dbPath)
	logger := waLog.Stdout("DB", "WARN", true)

	container, err := sqlstore.New(context.Background(), "sqlite", dsn, logger)
	if err != nil {
		return nil, fmt.Errorf("open sqlstore: %w", err)
	}

	appStore, err := openStore(filepath.Join(dataDir, "wakupi-app.db"))
	if err != nil {
		return nil, fmt.Errorf("open app store: %w", err)
	}

	return &Manager{
		emit:      emit,
		dbDir:     dataDir,
		mediaDir:  mediaDir,
		container: container,
		store:     appStore,
		clients:   make(map[string]*Session),
		avatarReq: make(map[string]bool),
	}, nil
}

func (m *Manager) MediaDir() string { return m.mediaDir }

// MediaHandler serves files from the media directory at /media/...
func (m *Manager) MediaHandler() http.Handler {
	return http.StripPrefix("/media/", http.FileServer(http.Dir(m.mediaDir)))
}

func (m *Manager) Sessions() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]SessionInfo, 0, len(m.clients))
	for _, s := range m.clients {
		out = append(out, s.snapshot())
	}
	return out
}

func (m *Manager) LoadExisting(ctx context.Context) error {
	devices, err := m.container.GetAllDevices(ctx)
	if err != nil {
		return err
	}
	for _, dev := range devices {
		if dev.ID == nil {
			continue
		}
		id := dev.ID.User
		s := m.attach(id, dev)
		go m.connect(s)
	}
	return nil
}

func (m *Manager) attach(id string, dev *store.Device) *Session {
	clientLog := waLog.Stdout("Client", "WARN", true)
	cl := whatsmeow.NewClient(dev, clientLog)

	s := &Session{
		ID:     id,
		Name:   "Akun " + id,
		Client: cl,
	}
	if dev.ID != nil {
		s.JID = dev.ID.String()
		s.Phone = "+" + dev.ID.User
		s.Name = "+" + dev.ID.User
	}

	cl.AddEventHandler(m.handlerFor(s))

	m.mu.Lock()
	m.clients[id] = s
	m.mu.Unlock()
	return s
}

func (m *Manager) handlerFor(s *Session) func(interface{}) {
	return func(evt interface{}) {
		switch e := evt.(type) {
		case *events.Connected:
			s.Connected = true
			if s.Client.Store.ID != nil {
				s.JID = s.Client.Store.ID.String()
				s.Phone = "+" + s.Client.Store.ID.User
				if s.Name == "" || strings.HasPrefix(s.Name, "Akun ") {
					s.Name = s.Phone
				}
			}
			m.emit("wa:connected", s.snapshot())
			go m.seedChatsFromContacts(s)
			go m.markOnline(s)
		case *events.Disconnected:
			s.Connected = false
			m.emit("wa:disconnected", s.snapshot())
		case *events.LoggedOut:
			s.Connected = false
			m.emit("wa:logged_out", s.snapshot())
		case *events.PairSuccess:
			m.emit("wa:pair_success", s.snapshot())
		case *events.Message:
			m.handleMessage(s, e)
		case *events.Receipt:
			m.handleReceipt(s, e)
		case *events.Presence:
			m.handlePresence(s, e)
		case *events.ChatPresence:
			m.handleChatPresence(s, e)
		case *events.HistorySync:
			m.handleHistorySync(s, e)
		case *events.OfflineSyncCompleted:
			m.emit("wa:sync_complete", map[string]interface{}{"sessionId": s.ID})
		}
	}
}

func (s *Session) snapshot() SessionInfo {
	return SessionInfo{
		ID:        s.ID,
		Name:      s.Name,
		Connected: s.Connected,
		JID:       s.JID,
		Phone:     s.Phone,
	}
}

func (m *Manager) handleReceipt(s *Session, e *events.Receipt) {
	if len(e.MessageIDs) == 0 {
		return
	}
	var t string
	switch e.Type {
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		t = "read"
	case types.ReceiptTypeDelivered:
		t = "delivered"
	case types.ReceiptTypePlayed:
		t = "played"
	default:
		return
	}
	ri := ReceiptInfo{
		AccountID:  s.ID,
		JID:        e.Chat.String(),
		Sender:     e.Sender.String(),
		MessageIDs: e.MessageIDs,
		Type:       t,
		Timestamp:  e.Timestamp.Unix(),
	}
	m.emit("wa:receipt", ri)
}

func (m *Manager) markOnline(s *Session) {
	// WhatsApp servers only push other users' status updates and presence
	// once we mark ourselves available. PushName may not be populated the
	// instant we connect, so retry briefly until it succeeds.
	for i := 0; i < 10; i++ {
		err := s.Client.SendPresence(context.Background(), types.PresenceAvailable)
		if err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}

func (m *Manager) handlePresence(s *Session, e *events.Presence) {
	pi := PresenceInfo{
		AccountID: s.ID,
		JID:       e.From.String(),
		Online:    !e.Unavailable,
		LastSeen:  e.LastSeen.Unix(),
	}
	m.emit("wa:presence", pi)
}

func (m *Manager) handleChatPresence(s *Session, e *events.ChatPresence) {
	state := "paused"
	if e.State == types.ChatPresenceComposing {
		state = "composing"
	}
	media := ""
	if e.Media != "" {
		media = string(e.Media)
	}
	cp := ChatPresenceInfo{
		AccountID: s.ID,
		JID:       e.Chat.String(),
		State:     state,
		Media:     media,
	}
	m.emit("wa:chat_presence", cp)
}

func (m *Manager) resolveName(s *Session, jid types.JID, fallback string) string {
	ctx := context.Background()
	if jid.Server == types.GroupServer {
		if info, err := s.Client.GetGroupInfo(ctx, jid); err == nil && info != nil {
			return info.Name
		}
	} else {
		if contact, err := s.Client.Store.Contacts.GetContact(ctx, jid); err == nil {
			if contact.FullName != "" {
				return contact.FullName
			}
			if contact.PushName != "" {
				return contact.PushName
			}
			if contact.BusinessName != "" {
				return contact.BusinessName
			}
		}
	}
	if fallback != "" {
		return fallback
	}
	return jid.User
}

func (m *Manager) StartLogin(ctx context.Context, sessionName string) (string, error) {
	dev := m.container.NewDevice()
	tempID := fmt.Sprintf("new-%d", time.Now().UnixNano())
	s := m.attach(tempID, dev)
	if sessionName != "" {
		s.Name = sessionName
	}

	qrChan, err := s.Client.GetQRChannel(ctx)
	if err != nil {
		return "", err
	}
	if err := s.Client.Connect(); err != nil {
		return "", err
	}

	go func() {
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				m.emit("wa:qr", map[string]interface{}{
					"sessionId": s.ID,
					"code":      evt.Code,
					"timeout":   evt.Timeout.Seconds(),
				})
			case "success":
				if s.Client.Store.ID != nil {
					newID := s.Client.Store.ID.User
					m.mu.Lock()
					delete(m.clients, s.ID)
					s.ID = newID
					s.JID = s.Client.Store.ID.String()
					s.Phone = "+" + s.Client.Store.ID.User
					if s.Name == "" || strings.HasPrefix(s.Name, "Akun ") {
						s.Name = s.Phone
					}
					m.clients[newID] = s
					m.mu.Unlock()
				}
				m.emit("wa:login_success", s.snapshot())
			case "timeout":
				m.emit("wa:qr_timeout", map[string]interface{}{"sessionId": s.ID})
			default:
				m.emit("wa:qr_event", map[string]interface{}{"sessionId": s.ID, "event": evt.Event})
			}
		}
	}()

	return s.ID, nil
}

func (m *Manager) connect(s *Session) {
	if s.Client.Store.ID == nil {
		return
	}
	if err := s.Client.Connect(); err != nil {
		m.emit("wa:error", map[string]interface{}{"sessionId": s.ID, "error": err.Error()})
	}
}

func (m *Manager) MarkRead(ctx context.Context, sessionID, jidStr, senderStr string, messageIDs []string) error {
	m.mu.RLock()
	s, ok := m.clients[sessionID]
	m.mu.RUnlock()
	if !ok {
		return errors.New("session not found")
	}
	if len(messageIDs) == 0 {
		return nil
	}
	chatJID, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	var senderJID types.JID
	if senderStr != "" {
		if sj, err := types.ParseJID(senderStr); err == nil {
			senderJID = sj
		}
	}
	ids := make([]types.MessageID, 0, len(messageIDs))
	for _, id := range messageIDs {
		ids = append(ids, types.MessageID(id))
	}
	return s.Client.MarkRead(ctx, ids, time.Now(), chatJID, senderJID)
}

func (m *Manager) SubscribePresence(ctx context.Context, jidStr, sessionID string) error {
	m.mu.RLock()
	s, ok := m.clients[sessionID]
	m.mu.RUnlock()
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	return s.Client.SubscribePresence(ctx, jid)
}

func (m *Manager) SendChatPresence(ctx context.Context, sessionID, jidStr, state string) error {
	m.mu.RLock()
	s, ok := m.clients[sessionID]
	m.mu.RUnlock()
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	var st types.ChatPresence
	switch state {
	case "composing":
		st = types.ChatPresenceComposing
	default:
		st = types.ChatPresencePaused
	}
	return s.Client.SendChatPresence(ctx, jid, st, types.ChatPresenceMediaText)
}

func (m *Manager) Logout(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	s, ok := m.clients[sessionID]
	m.mu.Unlock()
	if !ok {
		return errors.New("session not found")
	}
	if err := s.Client.Logout(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.clients, sessionID)
	m.mu.Unlock()
	return nil
}

func (m *Manager) Disconnect(sessionID string) error {
	m.mu.Lock()
	s, ok := m.clients[sessionID]
	m.mu.Unlock()
	if !ok {
		return errors.New("session not found")
	}
	s.Client.Disconnect()
	return nil
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.clients {
		s.Client.Disconnect()
	}
	if m.store != nil {
		_ = m.store.Close()
	}
}

func (m *Manager) LoadChats(ctx context.Context, accountID string) ([]ChatInfo, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.ListChats(ctx, accountID)
}

func (m *Manager) LoadMessages(ctx context.Context, accountID, jid string, limit int, beforeTS int64) ([]MessageInfo, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.ListMessages(ctx, accountID, jid, limit, beforeTS)
}

func (m *Manager) RefreshAvatar(ctx context.Context, sessionID, jidStr string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	go m.ensureAvatar(s, jid, jid.Server == types.GroupServer)
	return nil
}

func (m *Manager) GetAppSetting(ctx context.Context, key string) (string, error) {
	if m.store == nil {
		return "", nil
	}
	return m.store.GetSetting(ctx, key)
}

func (m *Manager) SetAppSetting(ctx context.Context, key, value string) error {
	if m.store == nil {
		return nil
	}
	return m.store.SetSetting(ctx, key, value)
}
