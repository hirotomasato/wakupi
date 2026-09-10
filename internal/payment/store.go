package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hirotomasato/paygateme/core"
	_ "modernc.org/sqlite"
)

// SQLStore implements core.PaymentStore backed by SQLite.
type SQLStore struct {
	db *sql.DB
}

// OpenSQLStore opens or creates a SQLite-backed payment store.
func OpenSQLStore(path string) (*SQLStore, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &SQLStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS payment_sessions (
			provider   TEXT NOT NULL,
			id         TEXT NOT NULL,
			session    TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			PRIMARY KEY (provider, id)
		)`,
		`CREATE TABLE IF NOT EXISTS payments (
			id             TEXT NOT NULL PRIMARY KEY,
			provider       TEXT NOT NULL,
			account_id     TEXT NOT NULL,
			merchant_id    TEXT NOT NULL,
			base_amount    INTEGER NOT NULL,
			unique_offset  INTEGER NOT NULL,
			unique_amount  INTEGER NOT NULL,
			status         TEXT NOT NULL,
			created_at     INTEGER NOT NULL,
			expires_at     INTEGER NOT NULL,
			reference      TEXT NOT NULL,
			qr_string      TEXT NOT NULL,
			txn    TEXT NOT NULL,
			metadata       TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_scope ON payments(provider, account_id, merchant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// --- PaymentStore ---

func (s *SQLStore) Create(ctx context.Context, p core.Payment) error {
	txJSON, _ := json.Marshal(p.Transaction)
	metaJSON, _ := json.Marshal(p.Metadata)
	scope := p.Scope
	if scope == nil {
		scope = &core.PaymentScope{}
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO payments (id, provider, account_id, merchant_id, base_amount, unique_offset, unique_amount, status, created_at, expires_at, reference, qr_string, txn, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID,
		scope.Provider, scope.AccountID, scope.MerchantID,
		p.BaseAmount, p.UniqueOffset, p.UniqueAmount,
		string(p.Status),
		p.CreatedAt.UnixMilli(), p.ExpiresAt.UnixMilli(),
		p.Reference, p.QRString,
		string(txJSON), string(metaJSON),
	)
	return err
}

func (s *SQLStore) Update(ctx context.Context, p core.Payment) error {
	txJSON, _ := json.Marshal(p.Transaction)
	metaJSON, _ := json.Marshal(p.Metadata)
	scope := p.Scope
	if scope == nil {
		scope = &core.PaymentScope{}
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE payments SET
			status=?, expires_at=?, txn=?, metadata=?
		 WHERE id=?`,
		string(p.Status),
		p.ExpiresAt.UnixMilli(),
		string(txJSON), string(metaJSON),
		p.ID,
	)
	return err
}

func (s *SQLStore) Get(ctx context.Context, id string) (*core.Payment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, provider, account_id, merchant_id, base_amount, unique_offset, unique_amount, status, created_at, expires_at, reference, qr_string, txn, metadata
		 FROM payments WHERE id=?`, id)
	return s.scanPayment(row)
}

func (s *SQLStore) ListActive(ctx context.Context, scope *core.PaymentScope) ([]core.Payment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, provider, account_id, merchant_id, base_amount, unique_offset, unique_amount, status, created_at, expires_at, reference, qr_string, txn, metadata
		 FROM payments WHERE status='pending' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []core.Payment
	for rows.Next() {
		p, err := s.scanPaymentRow(rows)
		if err != nil {
			return nil, err
		}
		if scope != nil {
			if p.Scope == nil || !core.SameScope(*p.Scope, *scope) {
				continue
			}
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *SQLStore) scanPayment(row *sql.Row) (*core.Payment, error) {
	var (
		id, provider, accountID, merchantID string
		baseAmount, uniqueOffset, uniqueAmount int64
		status string
		createdAtMs, expiresAtMs int64
		reference, qrString, txJSON, metaJSON string
	)
	err := row.Scan(&id, &provider, &accountID, &merchantID,
		&baseAmount, &uniqueOffset, &uniqueAmount, &status,
		&createdAtMs, &expiresAtMs,
		&reference, &qrString, &txJSON, &metaJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p := &core.Payment{
		ID:           id,
		Scope:        &core.PaymentScope{Provider: provider, AccountID: accountID, MerchantID: merchantID},
		BaseAmount:   baseAmount,
		UniqueOffset: uniqueOffset,
		UniqueAmount: uniqueAmount,
		Status:       core.PaymentStatus(status),
		CreatedAt:    time.UnixMilli(createdAtMs),
		ExpiresAt:    time.UnixMilli(expiresAtMs),
		Reference:    reference,
		QRString:     qrString,
	}
	if txJSON != "" && txJSON != "null" {
		var tx core.MerchantTransaction
		if json.Unmarshal([]byte(txJSON), &tx) == nil {
			p.Transaction = &tx
		}
	}
	if metaJSON != "" && metaJSON != "null" {
		_ = json.Unmarshal([]byte(metaJSON), &p.Metadata)
	}
	return p, nil
}

func (s *SQLStore) scanPaymentRow(rows *sql.Rows) (*core.Payment, error) {
	var (
		id, provider, accountID, merchantID string
		baseAmount, uniqueOffset, uniqueAmount int64
		status string
		createdAtMs, expiresAtMs int64
		reference, qrString, txJSON, metaJSON string
	)
	err := rows.Scan(&id, &provider, &accountID, &merchantID,
		&baseAmount, &uniqueOffset, &uniqueAmount, &status,
		&createdAtMs, &expiresAtMs,
		&reference, &qrString, &txJSON, &metaJSON)
	if err != nil {
		return nil, err
	}
	p := &core.Payment{
		ID:           id,
		Scope:        &core.PaymentScope{Provider: provider, AccountID: accountID, MerchantID: merchantID},
		BaseAmount:   baseAmount,
		UniqueOffset: uniqueOffset,
		UniqueAmount: uniqueAmount,
		Status:       core.PaymentStatus(status),
		CreatedAt:    time.UnixMilli(createdAtMs),
		ExpiresAt:    time.UnixMilli(expiresAtMs),
		Reference:    reference,
		QRString:     qrString,
	}
	if txJSON != "" && txJSON != "null" {
		var tx core.MerchantTransaction
		if json.Unmarshal([]byte(txJSON), &tx) == nil {
			p.Transaction = &tx
		}
	}
	if metaJSON != "" && metaJSON != "null" {
		_ = json.Unmarshal([]byte(metaJSON), &p.Metadata)
	}
	return p, nil
}

// ListAll returns all payments, newest first.
func (s *SQLStore) ListAll(ctx context.Context, limit int) ([]core.Payment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, provider, account_id, merchant_id, base_amount, unique_offset, unique_amount, status, created_at, expires_at, reference, qr_string, txn, metadata
		 FROM payments ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []core.Payment
	for rows.Next() {
		p, err := s.scanPaymentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *SQLStore) SaveSession(ctx context.Context, provider, id string, sessionJSON []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO payment_sessions (provider, id, session, created_at)
		 VALUES (?, ?, ?, ?)`,
		provider, id, string(sessionJSON), time.Now().UnixMilli())
	return err
}

func (s *SQLStore) LoadSession(ctx context.Context, provider, id string) ([]byte, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT session FROM payment_sessions WHERE provider=? AND id=?`, provider, id)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return []byte(raw), nil
}

// LoadLatestSession returns the most recently saved session for a provider.
func (s *SQLStore) LoadLatestSession(ctx context.Context, provider string) ([]byte, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT session FROM payment_sessions WHERE provider=? ORDER BY created_at DESC LIMIT 1`, provider)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return []byte(raw), nil
}

func (s *SQLStore) DeleteSession(ctx context.Context, provider, id string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM payment_sessions WHERE provider=? AND id=?`, provider, id)
	return err
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}