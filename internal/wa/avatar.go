package wa

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func (m *Manager) ensureAvatar(s *Session, jid types.JID, isGroup bool) {
	if jid.Server == types.BroadcastServer {
		return
	}
	key := s.ID + "::" + jid.String()
	m.avatarMu.Lock()
	if m.avatarReq[key] {
		m.avatarMu.Unlock()
		return
	}
	m.avatarReq[key] = true
	m.avatarMu.Unlock()

	defer func() {
		m.avatarMu.Lock()
		delete(m.avatarReq, key)
		m.avatarMu.Unlock()
	}()

	existing, _ := m.store.GetAvatarPath(context.Background(), s.ID, jid.String())
	if existing != "" {
		full := filepath.Join(m.mediaDir, existing)
		if _, err := os.Stat(full); err == nil {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := s.Client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: true})
	if err != nil || info == nil || info.URL == "" {
		return
	}

	data, err := downloadURL(ctx, info.URL)
	if err != nil || len(data) == 0 {
		return
	}

	hash := sha1.Sum([]byte(jid.String() + ":" + info.ID))
	name := "avatar_" + hex.EncodeToString(hash[:8]) + ".jpg"
	full := filepath.Join(m.mediaDir, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return
	}
	_ = m.store.UpdateChatAvatar(context.Background(), s.ID, jid.String(), name)

	m.emit("wa:avatar", map[string]interface{}{
		"accountId": s.ID,
		"jid":       jid.String(),
		"avatarUrl": "/media/" + name,
	})
}

func downloadURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, &httpError{Status: resp.StatusCode}
	}
	return io.ReadAll(resp.Body)
}

type httpError struct{ Status int }

func (e *httpError) Error() string { return http.StatusText(e.Status) }

// seedChatsFromContacts pulls saved contacts and joined groups from whatsmeow's
// internal store and registers them as chats so the chat list isn't empty
// even before any new message arrives or HistorySync fires.
func (m *Manager) seedChatsFromContacts(s *Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	contacts, err := s.Client.Store.Contacts.GetAllContacts(ctx)
	if err == nil {
		for jid, c := range contacts {
			if jid.Server != types.DefaultUserServer {
				continue
			}
			name := c.FullName
			if name == "" {
				name = c.PushName
			}
			if name == "" {
				name = c.BusinessName
			}
			if name == "" {
				continue
			}
			ci := &ChatInfo{
				AccountID:   s.ID,
				JID:         jid.String(),
				Name:        name,
				IsGroup:     false,
				LastMessage: "",
				LastTime:    0,
			}
			_ = m.store.UpsertChat(context.Background(), ci)
			avatarPath, _ := m.store.GetAvatarPath(context.Background(), s.ID, jid.String())
			ci.AvatarURL = avatarToURL(avatarPath)
			ci.ID = ci.JID
			go m.ensureAvatar(s, jid, false)
		}
	}

	groups, err := s.Client.GetJoinedGroups(ctx)
	if err == nil {
		for _, g := range groups {
			ci := &ChatInfo{
				AccountID:   s.ID,
				JID:         g.JID.String(),
				Name:        g.Name,
				IsGroup:     true,
				LastMessage: "",
				LastTime:    0,
			}
			_ = m.store.UpsertChat(context.Background(), ci)
			avatarPath, _ := m.store.GetAvatarPath(context.Background(), s.ID, g.JID.String())
			ci.AvatarURL = avatarToURL(avatarPath)
			ci.ID = ci.JID
			go m.ensureAvatar(s, g.JID, true)
		}
	}
}
