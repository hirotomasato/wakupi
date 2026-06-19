package wa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"wakupi/internal/desktop"
)

func (m *Manager) handleMessage(s *Session, e *events.Message) {
	chatJID := e.Info.Chat
	isGroup := chatJID.Server == types.GroupServer
	isStatus := chatJID == types.StatusBroadcastJID
	displayName := m.resolveName(s, chatJID, e.Info.PushName)

	if rev := getProtocolRevoke(e.Message); rev != "" {
		_ = m.store.MarkDeleted(context.Background(), s.ID, chatJID.String(), rev)
		m.emit("wa:deleted", DeletedInfo{
			AccountID: s.ID,
			JID:       chatJID.String(),
			MessageID: rev,
			Sender:    e.Info.Sender.String(),
		})
		return
	}

	if reaction := e.Message.GetReactionMessage(); reaction != nil {
		m.emit("wa:reaction", ReactionInfo{
			AccountID: s.ID,
			JID:       chatJID.String(),
			MessageID: reaction.GetKey().GetID(),
			Sender:    e.Info.Sender.String(),
			FromMe:    e.Info.IsFromMe,
			Emoji:     reaction.GetText(),
			Timestamp: e.Info.Timestamp.Unix(),
		})
		return
	}

	mi := MessageInfo{
		ID:        e.Info.ID,
		AccountID: s.ID,
		ChatID:    chatJID.String(),
		JID:       chatJID.String(),
		Sender:    e.Info.Sender.String(),
		Timestamp: e.Info.Timestamp.Unix(),
		FromMe:    e.Info.IsFromMe,
		IsGroup:   isGroup,
		PushName:  e.Info.PushName,
	}

	enrichFromMessage(&mi, e.Message)

	// Desktop command detection: !command
	if strings.HasPrefix(mi.Text, "!") && m.dc != nil {
		go func(session *Session, jid types.JID, text string) {
			response := desktop.HandleCommand(m.dc, text)
			if response != "" {
				_, _ = session.Client.SendMessage(context.Background(), jid, &waE2E.Message{
					Conversation: proto.String(response),
				})
			}
		}(s, chatJID, mi.Text)
		return
	}

	// CS Bot auto-reply hook — runs async, does not block the pipeline.
	if m.csHook != nil && !e.Info.IsFromMe && !isStatus && mi.Text != "" {
		go m.csHook(s, mi.Text, chatJID, displayName)
	}

	if mi.Text == "" && mi.MediaType == "" {
		return
	}

	if isStatus {
		if mi.MediaType != "" {
			go func(info *events.Message, base MessageInfo) {
				path, err := m.downloadAndSave(s, info)
				if err == nil && path != "" {
					base.MediaURL = "/media/" + filepath.Base(path)
				}
				m.emit("wa:status", base)
			}(e, mi)
			return
		}
		m.emit("wa:status", mi)
		return
	}

	if mi.MediaType != "" {
		go func(info *events.Message, base MessageInfo) {
			path, err := m.downloadAndSave(s, info)
			if err == nil && path != "" {
				base.MediaURL = "/media/" + filepath.Base(path)
				_ = m.store.UpdateMessageMediaURL(context.Background(), s.ID, chatJID.String(), base.ID, base.MediaURL)
				m.emit("wa:message", base)
				m.persistChat(s, chatJID, displayName, isGroup, base.Text, base.Timestamp)
				m.emitChatPreview(s, chatJID, displayName, isGroup, base.Text, base.Timestamp)
				return
			}
			_ = m.store.UpsertMessage(context.Background(), &base)
			m.emit("wa:message", base)
			m.persistChat(s, chatJID, displayName, isGroup, base.Text, base.Timestamp)
			m.emitChatPreview(s, chatJID, displayName, isGroup, base.Text, base.Timestamp)
		}(e, mi)
		_ = m.store.UpsertMessage(context.Background(), &mi)
		return
	}

	_ = m.store.UpsertMessage(context.Background(), &mi)
	m.emit("wa:message", mi)
	m.persistChat(s, chatJID, displayName, isGroup, mi.Text, mi.Timestamp)
	m.emitChatPreview(s, chatJID, displayName, isGroup, mi.Text, mi.Timestamp)

	go m.ensureAvatar(s, chatJID, isGroup)
}

func (m *Manager) persistChat(s *Session, chatJID types.JID, name string, isGroup bool, lastText string, ts int64) {
	ci := &ChatInfo{
		AccountID:   s.ID,
		JID:         chatJID.String(),
		Name:        name,
		IsGroup:     isGroup,
		LastMessage: lastText,
		LastTime:    ts,
	}
	_ = m.store.UpsertChat(context.Background(), ci)
}

func (m *Manager) emitChatPreview(s *Session, chatJID types.JID, name string, isGroup bool, lastText string, ts int64) {
	avatar, _ := m.store.GetAvatarPath(context.Background(), s.ID, chatJID.String())
	ci := ChatInfo{
		ID:          chatJID.String(),
		AccountID:   s.ID,
		JID:         chatJID.String(),
		Name:        name,
		IsGroup:     isGroup,
		LastMessage: lastText,
		LastTime:    ts,
		AvatarURL:   avatarToURL(avatar),
	}
	m.emit("wa:chat", ci)
}

func getProtocolRevoke(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	pm := msg.GetProtocolMessage()
	if pm == nil {
		return ""
	}
	if pm.GetType() == waE2E.ProtocolMessage_REVOKE {
		return pm.GetKey().GetID()
	}
	return ""
}

func enrichFromMessage(mi *MessageInfo, msg *waE2E.Message) {
	if msg == nil {
		return
	}

	var ctxInfo *waE2E.ContextInfo

	switch {
	case msg.Conversation != nil:
		mi.Text = msg.GetConversation()
	case msg.ExtendedTextMessage != nil:
		etm := msg.GetExtendedTextMessage()
		mi.Text = etm.GetText()
		ctxInfo = etm.GetContextInfo()
	case msg.ImageMessage != nil:
		im := msg.GetImageMessage()
		mi.MediaType = "image"
		mi.MimeType = im.GetMimetype()
		mi.FileSize = im.GetFileLength()
		mi.Width = im.GetWidth()
		mi.Height = im.GetHeight()
		mi.Caption = im.GetCaption()
		mi.Text = "[Foto]"
		if mi.Caption != "" {
			mi.Text = mi.Caption
		}
		ctxInfo = im.GetContextInfo()
	case msg.VideoMessage != nil:
		vm := msg.GetVideoMessage()
		mi.MediaType = "video"
		mi.MimeType = vm.GetMimetype()
		mi.FileSize = vm.GetFileLength()
		mi.Width = vm.GetWidth()
		mi.Height = vm.GetHeight()
		mi.Duration = vm.GetSeconds()
		mi.Caption = vm.GetCaption()
		mi.Text = "[Video]"
		if mi.Caption != "" {
			mi.Text = mi.Caption
		}
		ctxInfo = vm.GetContextInfo()
	case msg.AudioMessage != nil:
		am := msg.GetAudioMessage()
		mi.MediaType = "audio"
		mi.MimeType = am.GetMimetype()
		mi.FileSize = am.GetFileLength()
		mi.Duration = am.GetSeconds()
		mi.IsPTT = am.GetPTT()
		if mi.IsPTT {
			mi.Text = "[Voice note]"
		} else {
			mi.Text = "[Audio]"
		}
		ctxInfo = am.GetContextInfo()
	case msg.DocumentMessage != nil:
		dm := msg.GetDocumentMessage()
		mi.MediaType = "document"
		mi.MimeType = dm.GetMimetype()
		mi.FileSize = dm.GetFileLength()
		mi.FileName = dm.GetFileName()
		mi.Caption = dm.GetCaption()
		mi.Text = "[Dokumen] " + mi.FileName
		ctxInfo = dm.GetContextInfo()
	case msg.StickerMessage != nil:
		sm := msg.GetStickerMessage()
		mi.MediaType = "sticker"
		mi.MimeType = sm.GetMimetype()
		mi.FileSize = sm.GetFileLength()
		mi.Text = "[Stiker]"
		ctxInfo = sm.GetContextInfo()
	}

	if ctxInfo != nil && ctxInfo.GetStanzaID() != "" {
		mi.QuotedID = ctxInfo.GetStanzaID()
		mi.QuotedFrom = ctxInfo.GetParticipant()
		if quoted := ctxInfo.GetQuotedMessage(); quoted != nil {
			mi.QuotedText = previewText(quoted)
		}
	}
}

func previewText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return msg.GetConversation()
	}
	if msg.ExtendedTextMessage != nil {
		return msg.GetExtendedTextMessage().GetText()
	}
	if msg.ImageMessage != nil {
		c := msg.GetImageMessage().GetCaption()
		if c != "" {
			return "[Foto] " + c
		}
		return "[Foto]"
	}
	if msg.VideoMessage != nil {
		c := msg.GetVideoMessage().GetCaption()
		if c != "" {
			return "[Video] " + c
		}
		return "[Video]"
	}
	if msg.AudioMessage != nil {
		if msg.GetAudioMessage().GetPTT() {
			return "[Voice note]"
		}
		return "[Audio]"
	}
	if msg.DocumentMessage != nil {
		return "[Dokumen] " + msg.GetDocumentMessage().GetFileName()
	}
	if msg.StickerMessage != nil {
		return "[Stiker]"
	}
	return ""
}

// downloadAndSave fetches the encrypted media and stores it locally.
// Returns the absolute path of the saved file.
func (m *Manager) downloadAndSave(s *Session, e *events.Message) (string, error) {
	mediaMsg := getDownloadable(e.Message)
	if mediaMsg == nil {
		return "", errors.New("no downloadable media")
	}
	data, err := s.Client.Download(context.Background(), mediaMsg)
	if err != nil {
		return "", err
	}

	ext := guessExt(getMime(e.Message), getFileName(e.Message))
	hash := sha256.Sum256([]byte(e.Info.ID))
	name := hex.EncodeToString(hash[:8]) + "_" + e.Info.ID + ext
	name = sanitizeFileName(name)
	full := filepath.Join(m.mediaDir, name)
	if _, err := os.Stat(full); err == nil {
		return full, nil
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return full, nil
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}

func getDownloadable(msg *waE2E.Message) whatsmeow.DownloadableMessage {
	if msg == nil {
		return nil
	}
	switch {
	case msg.ImageMessage != nil:
		return msg.GetImageMessage()
	case msg.VideoMessage != nil:
		return msg.GetVideoMessage()
	case msg.AudioMessage != nil:
		return msg.GetAudioMessage()
	case msg.DocumentMessage != nil:
		return msg.GetDocumentMessage()
	case msg.StickerMessage != nil:
		return msg.GetStickerMessage()
	}
	return nil
}

func getMime(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	switch {
	case msg.ImageMessage != nil:
		return msg.GetImageMessage().GetMimetype()
	case msg.VideoMessage != nil:
		return msg.GetVideoMessage().GetMimetype()
	case msg.AudioMessage != nil:
		return msg.GetAudioMessage().GetMimetype()
	case msg.DocumentMessage != nil:
		return msg.GetDocumentMessage().GetMimetype()
	case msg.StickerMessage != nil:
		return msg.GetStickerMessage().GetMimetype()
	}
	return ""
}

func getFileName(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if msg.DocumentMessage != nil {
		return msg.GetDocumentMessage().GetFileName()
	}
	return ""
}

func guessExt(mimeType, fileName string) string {
	if fileName != "" {
		if ext := filepath.Ext(fileName); ext != "" {
			return ext
		}
	}
	if mimeType == "" {
		return ".bin"
	}
	mt := strings.SplitN(mimeType, ";", 2)[0]
	switch mt {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "audio/ogg", "audio/ogg; codecs=opus":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4", "audio/aac":
		return ".m4a"
	case "application/pdf":
		return ".pdf"
	}
	exts, err := mime.ExtensionsByType(mt)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ".bin"
}

// helper used by some other callers
func (m *Manager) sessionByID(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.clients[id]
	return s, ok
}

func (m *Manager) handleHistorySync(s *Session, e *events.HistorySync) {
	if e.Data == nil {
		return
	}
	convs := e.Data.GetConversations()
	for _, conv := range convs {
		jidStr := conv.GetID()
		jid, err := types.ParseJID(jidStr)
		if err != nil {
			continue
		}
		// Skip status broadcast — it's not a real chat.
		if jid == types.StatusBroadcastJID {
			continue
		}
		isGroup := jid.Server == types.GroupServer
		name := m.resolveName(s, jid, conv.GetName())

		var lastText string
		var lastTime int64
		msgs := conv.GetMessages()

		for i := len(msgs) - 1; i >= 0; i-- {
			webMsg := msgs[i].GetMessage()
			if webMsg == nil || webMsg.Message == nil {
				continue
			}
			key := webMsg.GetKey()
			if key == nil {
				continue
			}
			ts := int64(webMsg.GetMessageTimestamp())
			fromMe := key.GetFromMe()
			senderJID := webMsg.GetParticipant()
			if senderJID == "" {
				senderJID = key.GetParticipant()
			}
			pushName := webMsg.GetPushName()

			mi := MessageInfo{
				ID:        key.GetID(),
				AccountID: s.ID,
				ChatID:    jidStr,
				JID:       jidStr,
				Sender:    senderJID,
				Timestamp: ts,
				FromMe:    fromMe,
				IsGroup:   isGroup,
				PushName:  pushName,
			}
			enrichFromMessage(&mi, webMsg.Message)
			if mi.Text == "" && mi.MediaType == "" {
				continue
			}
			_ = m.store.UpsertMessage(context.Background(), &mi)
			// Don't emit wa:message per history msg - frontend re-fetches after sync_complete

			if ts > lastTime {
				lastTime = ts
				lastText = mi.Text
			}
		}

		if lastTime == 0 {
			continue
		}

		ci := ChatInfo{
			ID:          jidStr,
			AccountID:   s.ID,
			JID:         jidStr,
			Name:        name,
			IsGroup:     isGroup,
			LastMessage: lastText,
			LastTime:    lastTime,
		}
		_ = m.store.UpsertChat(context.Background(), &ci)
		avatarPath, _ := m.store.GetAvatarPath(context.Background(), s.ID, jidStr)
		ci.AvatarURL = avatarToURL(avatarPath)
		// Don't emit wa:chat per history conv - frontend re-fetches after sync_complete

		go m.ensureAvatar(s, jid, isGroup)
	}
}

func fillAccount(s *Session, mi *MessageInfo) {
	if mi.AccountID == "" {
		mi.AccountID = s.ID
	}
}

var _ = fmt.Sprintf
