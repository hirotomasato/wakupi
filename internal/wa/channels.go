package wa

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// channelInfoFromMetadata converts whatsmeow newsletter metadata into the UI
// channel struct. AvatarURL is left empty; callers that need it fetch the
// stored avatar path separately.
func channelInfoFromMetadata(meta *types.NewsletterMetadata) ChannelInfo {
	ci := ChannelInfo{
		JID:             meta.ID.String(),
		Name:            meta.ThreadMeta.Name.Text,
		Description:     meta.ThreadMeta.Description.Text,
		SubscriberCount: meta.ThreadMeta.SubscriberCount,
		InviteCode:      meta.ThreadMeta.InviteCode,
	}
	if meta.ViewerMeta != nil {
		ci.Role = string(meta.ViewerMeta.Role)
		ci.IsSubscribed = meta.ViewerMeta.Role != types.NewsletterRoleGuest
	}
	return ci
}

// seedChannels pulls the subscribed channels and registers them as chats so
// they appear in the list even before a new post arrives.
func (m *Manager) seedChannels(s *Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nls, err := s.Client.GetSubscribedNewsletters(ctx)
	if err != nil {
		// Log the error so we can diagnose — the UI has no way to surface
		// this otherwise since the method returns silently.
		fmt.Printf("[wakupi] seedChannels session=%s error=%v\n", s.ID, err)
		return
	}
	fmt.Printf("[wakupi] seedChannels session=%s found=%d channels\n", s.ID, len(nls))
	for _, nl := range nls {
		if nl.ID.IsEmpty() {
			continue
		}
		role := ""
		if nl.ViewerMeta != nil {
			role = string(nl.ViewerMeta.Role)
		}
		fmt.Printf("[wakupi] seedChannels channel=%s name=%q role=%q\n", nl.ID.String(), nl.ThreadMeta.Name.Text, role)
		ci := &ChatInfo{
			AccountID:   s.ID,
			JID:         nl.ID.String(),
			Name:        nl.ThreadMeta.Name.Text,
			IsChannel:   true,
			ChannelRole: role,
		}
		_ = m.store.UpsertChat(context.Background(), ci)
		ci.ID = ci.JID
		avatarPath, _ := m.store.GetAvatarPath(context.Background(), s.ID, ci.JID)
		ci.AvatarURL = avatarToURL(avatarPath)
		m.emit("wa:chat", ci)
		go m.saveChannelAvatar(s, nl)
	}
}

func (m *Manager) handleNewsletterJoin(s *Session, e *events.NewsletterJoin) {
	meta := &e.NewsletterMetadata
	if meta.ID.IsEmpty() {
		return
	}
	role := ""
	if meta.ViewerMeta != nil {
		role = string(meta.ViewerMeta.Role)
	}
	ci := &ChatInfo{
		AccountID:   s.ID,
		JID:         meta.ID.String(),
		Name:        meta.ThreadMeta.Name.Text,
		IsChannel:   true,
		ChannelRole: role,
		LastTime:    time.Now().Unix(),
	}
	_ = m.store.UpsertChat(context.Background(), ci)
	ci.ID = ci.JID
	avatarPath, _ := m.store.GetAvatarPath(context.Background(), s.ID, ci.JID)
	ci.AvatarURL = avatarToURL(avatarPath)
	m.emit("wa:chat", ci)
	go m.saveChannelAvatar(s, meta)
}

func (m *Manager) handleNewsletterLeave(s *Session, e *events.NewsletterLeave) {
	_ = m.store.DeleteChat(context.Background(), s.ID, e.ID.String())
	m.emit("wa:channel_left", map[string]interface{}{
		"accountId": s.ID,
		"jid":       e.ID.String(),
	})
}

func (m *Manager) handleNewsletterMuteChange(s *Session, e *events.NewsletterMuteChange) {
	v := int64(0)
	if e.Mute == types.NewsletterMuteOn {
		// A far-future timestamp marks the chat muted indefinitely, matching
		// how regular chat mutes are represented (muted_until > now).
		v = time.Now().Add(100 * 365 * 24 * time.Hour).Unix()
	}
	_ = m.store.SetChatFlag(context.Background(), s.ID, e.ID.String(), "muted_until", v)
	m.emit("wa:channel_mute", map[string]interface{}{
		"accountId": s.ID,
		"jid":       e.ID.String(),
		"muted":     e.Mute == types.NewsletterMuteOn,
	})
}

// saveChannelAvatar downloads the channel profile picture. The generic
// GetProfilePictureInfo path does not work for newsletter JIDs. The
// subscribed-newsletters response may omit the full picture (Picture is nil),
// so we fall back through Preview → GetNewsletterInfo.
func (m *Manager) saveChannelAvatar(s *Session, meta *types.NewsletterMetadata) {
	pic := meta.ThreadMeta.Picture
	if pic == nil || pic.URL == "" {
		// Try the thumbnail/preview which is always present (value type).
		if meta.ThreadMeta.Preview.URL != "" {
			pic = &meta.ThreadMeta.Preview
		}
	}
	if pic == nil || pic.URL == "" {
		// Last resort: fetch the full newsletter info (with fetch_full_image=true).
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		info, err := s.Client.GetNewsletterInfo(ctx, meta.ID)
		if err == nil && info != nil && info.ThreadMeta.Picture != nil && info.ThreadMeta.Picture.URL != "" {
			pic = info.ThreadMeta.Picture
		}
	}
	if pic == nil || pic.URL == "" {
		fmt.Printf("[wakupi] saveChannelAvatar jid=%s no picture URL available\n", meta.ID.String())
		return
	}
	existing, _ := m.store.GetAvatarPath(context.Background(), s.ID, meta.ID.String())
	if existing != "" {
		full := filepath.Join(m.mediaDir, existing)
		if _, err := os.Stat(full); err == nil {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	data, err := downloadURL(ctx, pic.URL)
	if err != nil || len(data) == 0 {
		return
	}

	hash := sha1.Sum([]byte(meta.ID.String() + ":" + pic.ID))
	name := "avatar_" + hex.EncodeToString(hash[:8]) + ".jpg"
	full := filepath.Join(m.mediaDir, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return
	}
	_ = m.store.UpdateChatAvatar(context.Background(), s.ID, meta.ID.String(), name)
	m.emit("wa:avatar", map[string]interface{}{
		"accountId": s.ID,
		"jid":       meta.ID.String(),
		"avatarUrl": "/media/" + name,
	})
}

// === Public channel operations ===

func (m *Manager) GetSubscribedChannels(ctx context.Context, sessionID string) ([]ChannelInfo, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	nls, err := s.Client.GetSubscribedNewsletters(ctx)
	if err != nil {
		fmt.Printf("[wakupi] GetSubscribedChannels session=%s error=%v\n", sessionID, err)
		return nil, err
	}
	out := make([]ChannelInfo, 0, len(nls))
	for _, nl := range nls {
		if nl.ID.IsEmpty() {
			continue
		}
		ci := channelInfoFromMetadata(nl)
		ci.IsSubscribed = true
		avatar, _ := m.store.GetAvatarPath(ctx, s.ID, nl.ID.String())
		ci.AvatarURL = avatarToURL(avatar)
		out = append(out, ci)
	}
	return out, nil
}

func (m *Manager) GetChannelInfo(ctx context.Context, sessionID, jidStr string) (*ChannelInfo, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, err
	}
	nl, err := s.Client.GetNewsletterInfo(ctx, jid)
	if err != nil {
		fmt.Printf("[wakupi] GetChannelInfo jid=%s error=%v\n", jidStr, err)
		return nil, err
	}
	ci := channelInfoFromMetadata(nl)
	fmt.Printf("[wakupi] GetChannelInfo jid=%s role=%q viewerMeta=%v\n", jidStr, ci.Role, nl.ViewerMeta != nil)
	avatar, _ := m.store.GetAvatarPath(ctx, s.ID, jidStr)
	ci.AvatarURL = avatarToURL(avatar)
	return &ci, nil
}

// GetChannelInfoByInvite resolves a channel from a follow/invite link. The key
// is the ... part after https://whatsapp.com/channel/.
func (m *Manager) GetChannelInfoByInvite(ctx context.Context, sessionID, key string) (*ChannelInfo, error) {
	s, ok := m.sessionByID(sessionID)
	if !ok {
		return nil, errors.New("session not found")
	}
	if !s.Connected {
		return nil, errors.New("session not connected")
	}
	nl, err := s.Client.GetNewsletterInfoWithInvite(ctx, key)
	if err != nil {
		return nil, err
	}
	ci := channelInfoFromMetadata(nl)
	// Invite lookups carry no viewer metadata, so a followed channel is not
	// distinguishable here. Treat as not-yet-followed until FollowChannel.
	ci.IsSubscribed = false
	return &ci, nil
}

func (m *Manager) FollowChannel(ctx context.Context, sessionID, jidStr string) error {
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
	return s.Client.FollowNewsletter(ctx, jid)
}

func (m *Manager) UnfollowChannel(ctx context.Context, sessionID, jidStr string) error {
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
	if err := s.Client.UnfollowNewsletter(ctx, jid); err != nil {
		return err
	}
	return m.store.DeleteChat(ctx, sessionID, jidStr)
}

// LoadChannelMessages fetches channel history straight from the network.
// Unlike regular chats, channels have no history sync, so the store only
// holds posts that arrived while the app was running.
func (m *Manager) LoadChannelMessages(ctx context.Context, sessionID, jidStr string, count int) ([]MessageInfo, error) {
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
	if count <= 0 || count > 100 {
		count = 50
	}
	nlms, err := s.Client.GetNewsletterMessages(ctx, jid, &whatsmeow.GetNewsletterMessagesParams{Count: count})
	if err != nil {
		return nil, err
	}
	out := make([]MessageInfo, 0, len(nlms))
	for _, nlm := range nlms {
		mi := MessageInfo{
			ID:        nlm.MessageID,
			AccountID: s.ID,
			ChatID:    jidStr,
			JID:       jidStr,
			Sender:    jidStr,
			Timestamp: nlm.Timestamp.Unix(),
			FromMe:    false,
			IsChannel: true,
		}
		if nlm.Message != nil {
			enrichFromMessage(&mi, nlm.Message)
		}
		if mi.Text == "" && mi.MediaType == "" {
			continue
		}
		out = append(out, mi)
	}
	return out, nil
}
