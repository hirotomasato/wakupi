package wa

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func openStore(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id          TEXT NOT NULL,
			account_id  TEXT NOT NULL,
			chat_jid    TEXT NOT NULL,
			sender      TEXT,
			text        TEXT,
			timestamp   INTEGER NOT NULL,
			from_me     INTEGER NOT NULL,
			is_group    INTEGER NOT NULL,
			push_name   TEXT,
			media_type  TEXT,
			media_url   TEXT,
			mime_type   TEXT,
			file_name   TEXT,
			file_size   INTEGER,
			width       INTEGER,
			height      INTEGER,
			duration    INTEGER,
			is_ptt      INTEGER,
			caption     TEXT,
			quoted_id   TEXT,
			quoted_text TEXT,
			quoted_from TEXT,
			deleted     INTEGER DEFAULT 0,
			starred     INTEGER DEFAULT 0,
			status      TEXT,
			PRIMARY KEY (account_id, chat_jid, id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages(account_id, chat_jid, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_search ON messages(account_id, text)`,
		`CREATE TABLE IF NOT EXISTS chats (
			account_id   TEXT NOT NULL,
			jid          TEXT NOT NULL,
			name         TEXT,
			is_group     INTEGER NOT NULL,
			last_message TEXT,
			last_time    INTEGER,
			avatar_path  TEXT,
			pinned       INTEGER DEFAULT 0,
			archived     INTEGER DEFAULT 0,
			muted_until  INTEGER DEFAULT 0,
			blocked      INTEGER DEFAULT 0,
			PRIMARY KEY (account_id, jid)
		)`,
		`CREATE TABLE IF NOT EXISTS reactions (
			account_id TEXT NOT NULL,
			chat_jid   TEXT NOT NULL,
			message_id TEXT NOT NULL,
			sender     TEXT NOT NULL,
			emoji      TEXT NOT NULL,
			from_me    INTEGER NOT NULL,
			timestamp  INTEGER NOT NULL,
			PRIMARY KEY (account_id, chat_jid, message_id, sender)
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT
		)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migration error: %w", err)
		}
	}

	// Add columns that may not exist in older databases. Only ignore the
	// "duplicate column" error so we don't silently swallow real problems.
	alters := []struct{ table, column, def string }{
		{"chats", "pinned", "INTEGER DEFAULT 0"},
		{"chats", "archived", "INTEGER DEFAULT 0"},
		{"chats", "muted_until", "INTEGER DEFAULT 0"},
		{"chats", "blocked", "INTEGER DEFAULT 0"},
		{"messages", "starred", "INTEGER DEFAULT 0"},
	}
	for _, a := range alters {
		if err := s.addColumnIfNotExists(a.table, a.column, a.def); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) addColumnIfNotExists(table, column, def string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("pragma table_info %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil // already exists
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	q := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, def)
	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("alter table %s add column %s: %w", table, column, err)
	}
	return nil
}

func (s *Store) UpsertMessage(ctx context.Context, m *MessageInfo) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (
			id, account_id, chat_jid, sender, text, timestamp, from_me, is_group, push_name,
			media_type, media_url, mime_type, file_name, file_size, width, height, duration,
			is_ptt, caption, quoted_id, quoted_text, quoted_from
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(account_id, chat_jid, id) DO UPDATE SET
			text=excluded.text,
			media_url=COALESCE(NULLIF(excluded.media_url,''), media_url),
			caption=excluded.caption,
			deleted=0
	`,
		m.ID, m.AccountID, m.JID, m.Sender, m.Text, m.Timestamp, boolToInt(m.FromMe), boolToInt(m.IsGroup), m.PushName,
		m.MediaType, m.MediaURL, m.MimeType, m.FileName, m.FileSize, m.Width, m.Height, m.Duration,
		boolToInt(m.IsPTT), m.Caption, m.QuotedID, m.QuotedText, m.QuotedFrom,
	)
	return err
}

func (s *Store) UpdateMessageMediaURL(ctx context.Context, accountID, chatJID, msgID, url string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE messages SET media_url=? WHERE account_id=? AND chat_jid=? AND id=?`,
		url, accountID, chatJID, msgID)
	return err
}

func (s *Store) MarkDeleted(ctx context.Context, accountID, chatJID, msgID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE messages SET deleted=1, text='Pesan ini dihapus', media_url='' WHERE account_id=? AND chat_jid=? AND id=?`,
		accountID, chatJID, msgID)
	return err
}

func (s *Store) UpsertChat(ctx context.Context, c *ChatInfo) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chats (account_id, jid, name, is_group, last_message, last_time)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(account_id, jid) DO UPDATE SET
			name=CASE WHEN excluded.name='' THEN chats.name ELSE excluded.name END,
			is_group=excluded.is_group,
			last_message=CASE WHEN excluded.last_time >= COALESCE(chats.last_time,0) THEN excluded.last_message ELSE chats.last_message END,
			last_time=CASE WHEN excluded.last_time >= COALESCE(chats.last_time,0) THEN excluded.last_time ELSE chats.last_time END
	`, c.AccountID, c.JID, c.Name, boolToInt(c.IsGroup), c.LastMessage, c.LastTime)
	return err
}

func (s *Store) UpdateChatAvatar(ctx context.Context, accountID, jid, path string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chats (account_id, jid, name, is_group, avatar_path) VALUES (?,?,?,?,?)
		 ON CONFLICT(account_id, jid) DO UPDATE SET avatar_path=excluded.avatar_path`,
		accountID, jid, "", 0, path)
	return err
}

func (s *Store) GetAvatarPath(ctx context.Context, accountID, jid string) (string, error) {
	var path sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT avatar_path FROM chats WHERE account_id=? AND jid=?`,
		accountID, jid).Scan(&path)
	if err != nil {
		return "", err
	}
	return path.String, nil
}

func (s *Store) ListChats(ctx context.Context, accountID string) ([]ChatInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT jid, COALESCE(name,''), is_group, COALESCE(last_message,''), COALESCE(last_time,0), COALESCE(avatar_path,''),
			COALESCE(pinned,0), COALESCE(archived,0), COALESCE(muted_until,0), COALESCE(blocked,0)
		FROM chats WHERE account_id=? ORDER BY last_time DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatInfo
	for rows.Next() {
		var c ChatInfo
		var isGroup, pinned, archived, blocked int
		var avatarPath string
		var mutedUntil int64
		if err := rows.Scan(&c.JID, &c.Name, &isGroup, &c.LastMessage, &c.LastTime, &avatarPath, &pinned, &archived, &mutedUntil, &blocked); err != nil {
			return nil, err
		}
		c.AccountID = accountID
		c.ID = c.JID
		c.IsGroup = isGroup == 1
		c.AvatarURL = avatarToURL(avatarPath)
		c.Pinned = pinned == 1
		c.Archived = archived == 1
		c.MutedUntil = mutedUntil
		c.Blocked = blocked == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetChatFlag(ctx context.Context, accountID, jid, column string) (int64, error) {
	var v int64
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COALESCE(%s,0) FROM chats WHERE account_id=? AND jid=?`, column),
		accountID, jid).Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *Store) SetChatFlag(ctx context.Context, accountID, jid, column string, value int64) error {
	switch column {
	case "pinned", "archived", "blocked", "muted_until", "desktop_cmd":
		// allowed
	default:
		return fmt.Errorf("invalid column")
	}
	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE chats SET %s=? WHERE account_id=? AND jid=?`, column),
		value, accountID, jid)
	return err
}

func (s *Store) StarMessage(ctx context.Context, accountID, jid, msgID string, starred bool) error {
	v := 0
	if starred {
		v = 1
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE messages SET starred=? WHERE account_id=? AND chat_jid=? AND id=?`,
		v, accountID, jid, msgID)
	return err
}

func (s *Store) ListStarred(ctx context.Context, accountID string, limit int) ([]MessageInfo, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, chat_jid, COALESCE(sender,''), COALESCE(text,''), timestamp, from_me, is_group, COALESCE(push_name,''),
			COALESCE(media_type,''), COALESCE(media_url,''), COALESCE(mime_type,''), COALESCE(file_name,''),
			COALESCE(file_size,0), COALESCE(width,0), COALESCE(height,0), COALESCE(duration,0),
			COALESCE(is_ptt,0), COALESCE(caption,''), COALESCE(quoted_id,''), COALESCE(quoted_text,''),
			COALESCE(quoted_from,'')
		FROM messages WHERE account_id=? AND starred=1
		ORDER BY timestamp DESC LIMIT ?`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows, accountID)
}

func (s *Store) SearchMessages(ctx context.Context, accountID, query string, limit int) ([]MessageInfo, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, chat_jid, COALESCE(sender,''), COALESCE(text,''), timestamp, from_me, is_group, COALESCE(push_name,''),
			COALESCE(media_type,''), COALESCE(media_url,''), COALESCE(mime_type,''), COALESCE(file_name,''),
			COALESCE(file_size,0), COALESCE(width,0), COALESCE(height,0), COALESCE(duration,0),
			COALESCE(is_ptt,0), COALESCE(caption,''), COALESCE(quoted_id,''), COALESCE(quoted_text,''),
			COALESCE(quoted_from,'')
		FROM messages WHERE account_id=? AND deleted=0 AND (text LIKE ? OR caption LIKE ? OR file_name LIKE ?)
		ORDER BY timestamp DESC LIMIT ?`, accountID, q, q, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows, accountID)
}

func scanMessages(rows *sql.Rows, accountID string) ([]MessageInfo, error) {
	var out []MessageInfo
	for rows.Next() {
		var m MessageInfo
		var fromMe, isGroup, isPTT int
		var chatJID string
		if err := rows.Scan(
			&m.ID, &chatJID, &m.Sender, &m.Text, &m.Timestamp, &fromMe, &isGroup, &m.PushName,
			&m.MediaType, &m.MediaURL, &m.MimeType, &m.FileName,
			&m.FileSize, &m.Width, &m.Height, &m.Duration,
			&isPTT, &m.Caption, &m.QuotedID, &m.QuotedText, &m.QuotedFrom,
		); err != nil {
			return nil, err
		}
		m.AccountID = accountID
		m.JID = chatJID
		m.ChatID = chatJID
		m.FromMe = fromMe == 1
		m.IsGroup = isGroup == 1
		m.IsPTT = isPTT == 1
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v.String, nil
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value)
	return err
}

func (s *Store) ListMessages(ctx context.Context, accountID, jid string, limit int, beforeTS int64) ([]MessageInfo, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT id, COALESCE(sender,''), COALESCE(text,''), timestamp, from_me, is_group, COALESCE(push_name,''),
			COALESCE(media_type,''), COALESCE(media_url,''), COALESCE(mime_type,''), COALESCE(file_name,''),
			COALESCE(file_size,0), COALESCE(width,0), COALESCE(height,0), COALESCE(duration,0),
			COALESCE(is_ptt,0), COALESCE(caption,''), COALESCE(quoted_id,''), COALESCE(quoted_text,''),
			COALESCE(quoted_from,''), COALESCE(deleted,0)
		  FROM messages WHERE account_id=? AND chat_jid=?`
	args := []interface{}{accountID, jid}
	if beforeTS > 0 {
		q += ` AND timestamp < ?`
		args = append(args, beforeTS)
	}
	q += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MessageInfo
	for rows.Next() {
		var m MessageInfo
		var fromMe, isGroup, isPTT, deleted int
		if err := rows.Scan(
			&m.ID, &m.Sender, &m.Text, &m.Timestamp, &fromMe, &isGroup, &m.PushName,
			&m.MediaType, &m.MediaURL, &m.MimeType, &m.FileName,
			&m.FileSize, &m.Width, &m.Height, &m.Duration,
			&isPTT, &m.Caption, &m.QuotedID, &m.QuotedText, &m.QuotedFrom, &deleted,
		); err != nil {
			return nil, err
		}
		m.AccountID = accountID
		m.JID = jid
		m.ChatID = jid
		m.FromMe = fromMe == 1
		m.IsGroup = isGroup == 1
		m.IsPTT = isPTT == 1
		if deleted == 1 {
			m.Text = "Pesan ini dihapus"
			m.MediaURL = ""
			m.MediaType = ""
		}
		out = append(out, m)
	}
	// reverse to chronological order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

func avatarToURL(path string) string {
	if path == "" {
		return ""
	}
	return "/media/" + path
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
