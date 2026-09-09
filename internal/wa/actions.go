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
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// === Chat flags ===

func (m *Manager) PinChat(ctx context.Context, accountID, jid string, pinned bool) error {
	if pinned {
		return m.store.SetChatFlag(ctx, accountID, jid, "pinned", 1)
	}
	return m.store.SetChatFlag(ctx, accountID, jid, "pinned", 0)
}

func (m *Manager) ArchiveChat(ctx context.Context, accountID, jid string, archived bool) error {
	v := int64(0)
	if archived {
		v = 1
	}
	return m.store.SetChatFlag(ctx, accountID, jid, "archived", v)
}

func (m *Manager) MuteChat(ctx context.Context, accountID, jid string, until int64) error {
	return m.store.SetChatFlag(ctx, accountID, jid, "muted_until", until)
}

func (m *Manager) BlockChat(ctx context.Context, sessionID, jidStr string, blocked bool) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	action := events.BlocklistChangeActionUnblock
	if blocked {
		action = events.BlocklistChangeActionBlock
	}
	if _, err := s.Client.UpdateBlocklist(ctx, jid, action); err != nil {
		return err
	}
	v := int64(0)
	if blocked {
		v = 1
	}
	return m.store.SetChatFlag(context.Background(), sessionID, jidStr, "blocked", v)
}

// === Forward ===

func (m *Manager) ForwardMessage(ctx context.Context, sessionID, fromChatJID, msgID string, toJIDs []string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	if !s.Connected {
		return errors.New("session not connected")
	}
	msgs, err := m.store.ListMessages(ctx, sessionID, fromChatJID, 500, 0)
	if err != nil {
		return err
	}
	var src *MessageInfo
	for i := range msgs {
		if msgs[i].ID == msgID {
			src = &msgs[i]
			break
		}
	}
	if src == nil {
		return errors.New("source message not found")
	}

	for _, jidStr := range toJIDs {
		jid, err := types.ParseJID(jidStr)
		if err != nil {
			continue
		}
		if src.MediaType == "" {
			_, _ = s.Client.SendMessage(ctx, jid, &waE2E.Message{
				Conversation: proto.String(src.Text),
			})
			continue
		}
		if src.MediaURL == "" {
			continue
		}
		localPath := filepath.Join(m.mediaDir, filepath.Base(src.MediaURL))
		data, err := os.ReadFile(localPath)
		if err != nil {
			continue
		}
		mt := whatsmeow.MediaImage
		switch src.MediaType {
		case "video":
			mt = whatsmeow.MediaVideo
		case "audio":
			mt = whatsmeow.MediaAudio
		case "document":
			mt = whatsmeow.MediaDocument
		}
		uploaded, err := s.Client.Upload(ctx, data, mt)
		if err != nil {
			continue
		}
		var msg *waE2E.Message
		switch src.MediaType {
		case "image":
			msg = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String(src.Caption), Mimetype: proto.String(src.MimeType),
				URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
				MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			}}
		case "video":
			msg = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
				Caption: proto.String(src.Caption), Mimetype: proto.String(src.MimeType),
				URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
				MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			}}
		case "audio":
			msg = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
				PTT: proto.Bool(src.IsPTT), Mimetype: proto.String(src.MimeType),
				URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
				MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			}}
		case "document":
			msg = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
				FileName: proto.String(src.FileName), Mimetype: proto.String(src.MimeType),
				URL: proto.String(uploaded.URL), DirectPath: proto.String(uploaded.DirectPath),
				MediaKey: uploaded.MediaKey, FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256: uploaded.FileSHA256, FileLength: proto.Uint64(uploaded.FileLength),
			}}
		}
		if msg != nil {
			_, _ = s.Client.SendMessage(ctx, jid, msg)
		}
	}
	return nil
}

// === Contact validate ===

type ContactCheck struct {
	Phone string `json:"phone"`
	JID   string `json:"jid"`
	OnWA  bool   `json:"onWhatsApp"`
}

func (m *Manager) IsOnWhatsApp(ctx context.Context, sessionID string, phones []string) ([]ContactCheck, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	resp, err := s.Client.IsOnWhatsApp(ctx, phones)
	if err != nil {
		return nil, err
	}
	out := make([]ContactCheck, 0, len(resp))
	for _, r := range resp {
		out = append(out, ContactCheck{
			Phone: r.Query,
			JID:   r.JID.String(),
			OnWA:  r.IsIn,
		})
	}
	return out, nil
}

// === Group ===

type GroupInfo struct {
	JID         string         `json:"jid"`
	Name        string         `json:"name"`
	Topic       string         `json:"topic"`
	OwnerJID    string         `json:"ownerJid"`
	Created     int64          `json:"created"`
	IsAnnounce  bool           `json:"isAnnounce"`
	IsLocked    bool           `json:"isLocked"`
	Participants []GroupMember `json:"participants"`
}

type GroupMember struct {
	JID        string `json:"jid"`
	IsAdmin    bool   `json:"isAdmin"`
	IsSuper    bool   `json:"isSuperAdmin"`
	PushName   string `json:"pushName"`
}

func (m *Manager) GetGroupInfo(ctx context.Context, sessionID, jidStr string) (*GroupInfo, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, err
	}
	info, err := s.Client.GetGroupInfo(ctx, jid)
	if err != nil {
		return nil, err
	}
	gi := &GroupInfo{
		JID:        info.JID.String(),
		Name:       info.Name,
		Topic:      info.Topic,
		OwnerJID:   info.OwnerJID.String(),
		Created:    info.GroupCreated.Unix(),
		IsAnnounce: info.IsAnnounce,
		IsLocked:   info.IsLocked,
	}
	for _, p := range info.Participants {
		gi.Participants = append(gi.Participants, GroupMember{
			JID:      p.JID.String(),
			IsAdmin:  p.IsAdmin,
			IsSuper:  p.IsSuperAdmin,
			PushName: p.DisplayName,
		})
	}
	return gi, nil
}

func (m *Manager) LeaveGroup(ctx context.Context, sessionID, jidStr string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	return s.Client.LeaveGroup(ctx, jid)
}

func (m *Manager) UpdateGroupParticipants(ctx context.Context, sessionID, jidStr string, participantJIDs []string, action string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	jids := make([]types.JID, 0, len(participantJIDs))
	for _, p := range participantJIDs {
		pj, err := types.ParseJID(p)
		if err != nil {
			continue
		}
		jids = append(jids, pj)
	}
	var change whatsmeow.ParticipantChange
	switch action {
	case "add":
		change = whatsmeow.ParticipantChangeAdd
	case "remove":
		change = whatsmeow.ParticipantChangeRemove
	case "promote":
		change = whatsmeow.ParticipantChangePromote
	case "demote":
		change = whatsmeow.ParticipantChangeDemote
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
	_, err = s.Client.UpdateGroupParticipants(ctx, jid, jids, change)
	return err
}

func (m *Manager) SetGroupName(ctx context.Context, sessionID, jidStr, name string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return err
	}
	return s.Client.SetGroupName(ctx, jid, name)
}

// === Profile (own) ===

func (m *Manager) SetSelfStatus(ctx context.Context, sessionID, status string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	return s.Client.SetStatusMessage(ctx, types.SetStatusInput{Text: proto.String(status)})
}

func (m *Manager) SetSelfProfilePicture(ctx context.Context, sessionID, filePath string) error {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return errors.New("session not found")
	}
	if s.Client.Store.ID == nil {
		return errors.New("not logged in")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	_, err = s.Client.SetGroupPhoto(ctx, types.EmptyJID, data)
	return err
}
