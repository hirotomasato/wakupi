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
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	mimeType := detectMime(filePath, "image/jpeg")

	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			ContextInfo:   buildQuoteContext(quoted),
		},
	}
	resp, err := s.Client.SendMessage(ctx, jid, msg)
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
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	mimeType := detectMime(filePath, "video/mp4")

	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaVideo)
	if err != nil {
		return nil, err
	}

	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			ContextInfo:   buildQuoteContext(quoted),
		},
	}
	resp, err := s.Client.SendMessage(ctx, jid, msg)
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
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	mimeType := detectMime(filePath, "application/octet-stream")
	fileName := filepath.Base(filePath)

	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, err
	}
	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			FileName:      proto.String(fileName),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			ContextInfo:   buildQuoteContext(quoted),
		},
	}
	resp, err := s.Client.SendMessage(ctx, jid, msg)
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
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	mimeType := detectMime(filePath, "audio/ogg; codecs=opus")

	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaAudio)
	if err != nil {
		return nil, err
	}
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			PTT:           proto.Bool(ptt),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			ContextInfo:   buildQuoteContext(quoted),
		},
	}
	resp, err := s.Client.SendMessage(ctx, jid, msg)
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

func (m *Manager) PostStatusText(ctx context.Context, sessionID, text string) (string, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return "", errors.New("session not found")
	}
	if !s.Connected {
		return "", errors.New("session not connected")
	}
	resp, err := s.Client.SendMessage(ctx, types.StatusBroadcastJID, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *Manager) PostStatusImage(ctx context.Context, sessionID, filePath, caption string) (string, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return "", errors.New("session not found")
	}
	if !s.Connected {
		return "", errors.New("session not connected")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	mimeType := detectMime(filePath, "image/jpeg")
	uploaded, err := s.Client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return "", err
	}
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
		},
	}
	resp, err := s.Client.SendMessage(ctx, types.StatusBroadcastJID, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
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
