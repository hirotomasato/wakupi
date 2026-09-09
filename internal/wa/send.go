package wa

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type QuotedRef struct {
	ID          string
	Participant string
	Text        string
}

func (m *Manager) SendText(ctx context.Context, sessionID, jidStr, text string, quoted *QuotedRef) (string, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return "", errors.New("session not found")
	}
	if !s.Connected {
		return "", errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return "", fmt.Errorf("invalid jid: %w", err)
	}

	msg := &waE2E.Message{}
	if quoted != nil && quoted.ID != "" {
		msg.ExtendedTextMessage = &waE2E.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: buildQuoteContext(quoted),
		}
	} else {
		msg.Conversation = proto.String(text)
	}

	resp, err := s.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func buildQuoteContext(q *QuotedRef) *waE2E.ContextInfo {
	if q == nil || q.ID == "" {
		return nil
	}
	quoted := &waE2E.Message{Conversation: proto.String(q.Text)}
	ci := &waE2E.ContextInfo{
		StanzaID:      proto.String(q.ID),
		QuotedMessage: quoted,
	}
	if q.Participant != "" {
		ci.Participant = proto.String(q.Participant)
	}
	return ci
}

type SendMediaResult struct {
	MessageID string `json:"messageId"`
	LocalURL  string `json:"localUrl"`
	MimeType  string `json:"mimeType"`
}

// sendMedia uploads and sends a media message. Newsletter JIDs use the
// unencrypted upload path (UploadNewsletter) and require the returned media
// handle in the send request. The build callback constructs the protobuf
// message; MediaKey/FileEncSHA256 are only populated for encrypted uploads,
// so the callback should set them only when len(uploaded.MediaKey) > 0.
func (m *Manager) sendMedia(ctx context.Context, s *Session, jid types.JID, filePath, mimeType string, mediaType whatsmeow.MediaType, build func(uploaded whatsmeow.UploadResponse) *waE2E.Message) (*SendMediaResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var uploaded whatsmeow.UploadResponse
	var extra []whatsmeow.SendRequestExtra
	if jid.Server == types.NewsletterServer {
		uploaded, err = s.Client.UploadNewsletter(ctx, data, mediaType)
		if err != nil {
			return nil, fmt.Errorf("upload: %w", err)
		}
		extra = []whatsmeow.SendRequestExtra{{MediaHandle: uploaded.Handle}}
	} else {
		uploaded, err = s.Client.Upload(ctx, data, mediaType)
		if err != nil {
			return nil, fmt.Errorf("upload: %w", err)
		}
	}

	resp, err := s.Client.SendMessage(ctx, jid, build(uploaded), extra...)
	if err != nil {
		return nil, err
	}

	localPath, _ := m.copyToMedia(filePath, resp.ID, mimeType)
	url := ""
	if localPath != "" {
		url = "/media/" + filepath.Base(localPath)
	}
	return &SendMediaResult{MessageID: resp.ID, LocalURL: url, MimeType: mimeType}, nil
}

func (m *Manager) SendImage(ctx context.Context, sessionID, jidStr, filePath, caption string, quoted *QuotedRef) (*SendMediaResult, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	mimeType := detectMime(filePath, "image/jpeg")
	return m.sendMedia(ctx, s, jid, filePath, mimeType, whatsmeow.MediaImage, func(uploaded whatsmeow.UploadResponse) *waE2E.Message {
		im := &waE2E.ImageMessage{
			Caption:     proto.String(caption),
			Mimetype:    proto.String(mimeType),
			URL:         proto.String(uploaded.URL),
			DirectPath:  proto.String(uploaded.DirectPath),
			FileSHA256:  uploaded.FileSHA256,
			FileLength:  proto.Uint64(uploaded.FileLength),
			ContextInfo: buildQuoteContext(quoted),
		}
		if len(uploaded.MediaKey) > 0 {
			im.MediaKey = uploaded.MediaKey
			im.FileEncSHA256 = uploaded.FileEncSHA256
		}
		return &waE2E.Message{ImageMessage: im}
	})
}

func (m *Manager) SendVideo(ctx context.Context, sessionID, jidStr, filePath, caption string, quoted *QuotedRef) (*SendMediaResult, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	mimeType := detectMime(filePath, "video/mp4")
	return m.sendMedia(ctx, s, jid, filePath, mimeType, whatsmeow.MediaVideo, func(uploaded whatsmeow.UploadResponse) *waE2E.Message {
		vm := &waE2E.VideoMessage{
			Caption:     proto.String(caption),
			Mimetype:    proto.String(mimeType),
			URL:         proto.String(uploaded.URL),
			DirectPath:  proto.String(uploaded.DirectPath),
			FileSHA256:  uploaded.FileSHA256,
			FileLength:  proto.Uint64(uploaded.FileLength),
			ContextInfo: buildQuoteContext(quoted),
		}
		if len(uploaded.MediaKey) > 0 {
			vm.MediaKey = uploaded.MediaKey
			vm.FileEncSHA256 = uploaded.FileEncSHA256
		}
		return &waE2E.Message{VideoMessage: vm}
	})
}

func (m *Manager) SendDocument(ctx context.Context, sessionID, jidStr, filePath string, quoted *QuotedRef) (*SendMediaResult, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	mimeType := detectMime(filePath, "application/octet-stream")
	fileName := filepath.Base(filePath)
	return m.sendMedia(ctx, s, jid, filePath, mimeType, whatsmeow.MediaDocument, func(uploaded whatsmeow.UploadResponse) *waE2E.Message {
		dm := &waE2E.DocumentMessage{
			FileName:    proto.String(fileName),
			Mimetype:    proto.String(mimeType),
			URL:         proto.String(uploaded.URL),
			DirectPath:  proto.String(uploaded.DirectPath),
			FileSHA256:  uploaded.FileSHA256,
			FileLength:  proto.Uint64(uploaded.FileLength),
			ContextInfo: buildQuoteContext(quoted),
		}
		if len(uploaded.MediaKey) > 0 {
			dm.MediaKey = uploaded.MediaKey
			dm.FileEncSHA256 = uploaded.FileEncSHA256
		}
		return &waE2E.Message{DocumentMessage: dm}
	})
}

func (m *Manager) SendAudio(ctx context.Context, sessionID, jidStr, filePath string, ptt bool, quoted *QuotedRef) (*SendMediaResult, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	mimeType := detectMime(filePath, "audio/ogg; codecs=opus")
	return m.sendMedia(ctx, s, jid, filePath, mimeType, whatsmeow.MediaAudio, func(uploaded whatsmeow.UploadResponse) *waE2E.Message {
		am := &waE2E.AudioMessage{
			PTT:         proto.Bool(ptt),
			Mimetype:    proto.String(mimeType),
			URL:         proto.String(uploaded.URL),
			DirectPath:  proto.String(uploaded.DirectPath),
			FileSHA256:  uploaded.FileSHA256,
			FileLength:  proto.Uint64(uploaded.FileLength),
			ContextInfo: buildQuoteContext(quoted),
		}
		if len(uploaded.MediaKey) > 0 {
			am.MediaKey = uploaded.MediaKey
			am.FileEncSHA256 = uploaded.FileEncSHA256
		}
		return &waE2E.Message{AudioMessage: am}
	})
}

func (m *Manager) DeleteMessage(ctx context.Context, sessionID, jidStr, messageID string, forEveryone bool) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	if !s.Connected {
		return errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	if forEveryone {
		// Server will echo a revoke protocol message; handleMessage picks it up
		// and calls MarkDeleted + emit("wa:deleted") for us.
		_, err := s.Client.SendMessage(ctx, jid, s.Client.BuildRevoke(jid, types.EmptyJID, types.MessageID(messageID)))
		return err
	}
	// Delete-for-me: server does NOT notify us, so handle locally.
	if err := m.store.MarkDeleted(ctx, sessionID, jidStr, messageID); err != nil {
		return err
	}
	m.emit("wa:deleted", DeletedInfo{
		AccountID: sessionID,
		JID:       jidStr,
		MessageID: messageID,
		Sender:    "",
	})
	return nil
}

func (m *Manager) ReactMessage(ctx context.Context, sessionID, jidStr, messageID, sender, emoji string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	if !s.Connected {
		return errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	var senderJID types.JID
	if sender != "" {
		if sj, err := types.ParseJID(sender); err == nil {
			senderJID = sj
		}
	}
	_, err = s.Client.SendMessage(ctx, jid, s.Client.BuildReaction(jid, senderJID, types.MessageID(messageID), emoji))
	return err
}

func (m *Manager) copyToMedia(srcPath, msgID, mimeType string) (string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(srcPath)
	if ext == "" {
		ext = guessExt(mimeType, "")
	}
	dst := filepath.Join(m.mediaDir, "out_"+msgID+ext)
	dst = sanitizePath(dst)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	return dst, nil
}

func sanitizePath(p string) string {
	dir, base := filepath.Split(p)
	return filepath.Join(dir, sanitizeFileName(base))
}

func detectMime(path, fallback string) string {
	switch ext := filepath.Ext(path); ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogg", ".opus":
		return "audio/ogg; codecs=opus"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".aac":
		return "audio/mp4"
	case ".pdf":
		return "application/pdf"
	}
	return fallback
}
