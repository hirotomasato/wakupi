package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hirotomasato/paygateme/core"
	"github.com/hirotomasato/paygateme/shopee"
	"github.com/hirotomasato/paygateme/utils"
)

// Manager orchestrates the Shopee merchant payment lifecycle. It owns the
// provider, challenge/verification intermediate state, and the SQLite store.
// All exported methods are safe for concurrent use.
type Manager struct {
	mu       sync.Mutex
	store    *SQLStore
	provider *shopee.Provider
	emit     func(name string, data ...interface{})
	started  bool

	// Intermediate login state.
	challenge    *shopee.OtpChallenge
	verification *shopee.OtpVerification
}

// NewManager creates a payment manager. The SQLite store is opened at
// dataDir/payment.db. emit is called for real-time events (payment:paid, etc.).
func NewManager(dataDir string, emit func(name string, data ...interface{})) (*Manager, error) {
	store, err := OpenSQLStore(dataDir + "/payment.db")
	if err != nil {
		return nil, fmt.Errorf("payment store: %w", err)
	}
	return &Manager{store: store, emit: emit}, nil
}

// --- Session ---

// HasSession reports whether a Shopee session is loaded and authenticated.
func (m *Manager) HasSession() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.provider != nil && m.provider.Authenticated()
}

// Logout discards the current session and provider.
func (m *Manager) Logout(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopServiceLocked()

	if m.provider != nil {
		if scope := m.provider.GetPaymentScope(); scope != nil {
			_ = m.store.DeleteSession(ctx, scope.Provider, scope.AccountID)
		}
	}
	m.provider = nil
	m.challenge = nil
	m.verification = nil
	m.started = false
	return nil
}

// --- OTP Login ---

// OtpResponse is the result of requesting an OTP.
type OtpResponse struct {
	PhoneNumber string `json:"phoneNumber"`
	Channel     int    `json:"channel"`
	HasPassword bool   `json:"hasPassword"`
}

// newProvider builds a provider wired to the SQLite store and settlement
// events.
func (m *Manager) newProvider(session *shopee.Session) *shopee.Provider {
	return shopee.NewProvider(shopee.ProviderConfig{
		Session:      session,
		Store:        m.store,
		DeviceReport: shopee.DeviceRiskBlob,
		Logger:       utils.NoopLogger,
		OnSessionUpdated: func(s shopee.Session) error {
			data, _ := json.Marshal(s)
			return m.store.SaveSession(context.Background(), "shopee", s.AccountID, data)
		},
	})
}

// RequestOtp sends an OTP to the given phone number. Password is required for
// password-protected accounts; pass empty to discover whether the account
// requires one (the error message will indicate it).
func (m *Manager) RequestOtp(phone, password string) (*OtpResponse, error) {
	p := m.newProvider(nil)

	challenge, err := p.RequestOtp(context.Background(), phone, shopee.OtpRequestOptions{
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.provider = p
	m.challenge = challenge
	m.mu.Unlock()

	return &OtpResponse{
		PhoneNumber: challenge.PhoneNumber,
		Channel:     challenge.Channel,
		HasPassword: challenge.HasPassword,
	}, nil
}

// VerifyLoginOutcome is the result of verifying an OTP and attempting login.
type VerifyLoginOutcome struct {
	Status       string            `json:"status"` // "complete" or "merchant-selection-required"
	MerchantID   string            `json:"merchantId,omitempty"`
	MerchantName string            `json:"merchantName,omitempty"`
	Merchants    []MerchantSummary `json:"merchants,omitempty"`
}

// MerchantSummary for the frontend.
type MerchantSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

// VerifyOtp verifies the OTP and attempts login. If there are multiple
// merchants, the caller must pick one via CompleteLogin.
func (m *Manager) VerifyOtp(otp string) (*VerifyLoginOutcome, error) {
	m.mu.Lock()
	challenge := m.challenge
	provider := m.provider
	m.mu.Unlock()

	if challenge == nil || provider == nil {
		return nil, fmt.Errorf("no pending OTP challenge; call RequestOtp first")
	}

	outcome, err := provider.LoginWithOtp(context.Background(), shopee.LoginWithOtpInput{
		Challenge: *challenge,
		OTP:       otp,
	})
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.challenge = nil // consumed
	m.mu.Unlock()

	result := &VerifyLoginOutcome{Status: string(outcome.Status)}

	switch outcome.Status {
	case shopee.LoginComplete:
		result.MerchantID = outcome.Session.Merchant.ID
		result.MerchantName = outcome.Session.Merchant.Name
		sessionJSON, _ := json.Marshal(outcome.Session)
		_ = m.store.SaveSession(context.Background(), "shopee", outcome.Session.AccountID, sessionJSON)
		m.startService()

	case shopee.LoginMerchantSelectionNeeded:
		m.mu.Lock()
		m.verification = outcome.Verification
		m.mu.Unlock()

		merchants := make([]MerchantSummary, len(outcome.Merchants))
		for i, m := range outcome.Merchants {
			merchants[i] = MerchantSummary{ID: m.ID, Name: m.Name, Status: m.Status}
		}
		result.Merchants = merchants
	}

	return result, nil
}

// CompleteLoginResult holds the result of completing login with a specific merchant.
type CompleteLoginResult struct {
	Status       string `json:"status"`
	MerchantID   string `json:"merchantId"`
	MerchantName string `json:"merchantName"`
}

// CompleteLogin finishes login by selecting a specific merchant + store.
func (m *Manager) CompleteLogin(merchantID, storeID string) (*CompleteLoginResult, error) {
	m.mu.Lock()
	verification := m.verification
	provider := m.provider
	m.mu.Unlock()

	if verification == nil || provider == nil {
		return nil, fmt.Errorf("no pending verification; call VerifyOtp first")
	}

	session, err := provider.CompleteLogin(context.Background(), shopee.CompleteLoginInput{
		Verification: *verification,
		MerchantID:   merchantID,
		StoreID:      storeID,
	})
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.verification = nil
	m.mu.Unlock()

	sessionJSON, _ := json.Marshal(session)
	_ = m.store.SaveSession(context.Background(), "shopee", session.AccountID, sessionJSON)

	m.startService()

	return &CompleteLoginResult{
		Status:       "complete",
		MerchantID:   session.Merchant.ID,
		MerchantName: session.Merchant.Name,
	}, nil
}

// RestoreSession attempts to restore a previously persisted session.
func (m *Manager) RestoreSession(ctx context.Context) (bool, error) {
	raw, err := m.store.LoadLatestSession(ctx, "shopee")
	if err != nil || raw == nil {
		return false, nil
	}

	var session shopee.Session
	if err := json.Unmarshal(raw, &session); err != nil {
		return false, nil
	}

	m.mu.Lock()
	m.provider = m.newProvider(&session)
	m.mu.Unlock()

	if !m.provider.Authenticated() {
		return false, nil
	}

	m.startService()
	return true, nil
}

// --- Service lifecycle ---

// startService registers settlement handlers and begins background polling.
// Safe to call repeatedly; only the first call has an effect.
func (m *Manager) startService() {
	m.mu.Lock()
	provider := m.provider
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	if provider == nil {
		return
	}

	svc, err := provider.Payments()
	if err != nil {
		if m.emit != nil {
			m.emit("payment:error", map[string]interface{}{"error": err.Error()})
		}
		return
	}

	svc.OnPaid(func(p core.Payment) {
		if m.emit != nil {
			m.emit("payment:paid", paymentToInfo(&p))
		}
	})
	svc.OnExpired(func(p core.Payment) {
		if m.emit != nil {
			m.emit("payment:expired", paymentToInfo(&p))
		}
	})
	svc.OnError(func(err error) {
		if m.emit != nil {
			m.emit("payment:error", map[string]interface{}{"error": err.Error()})
		}
	})

	svc.Start()
}

// stopServiceLocked stops background polling if it was started.
func (m *Manager) stopServiceLocked() {
	if m.provider == nil {
		return
	}
	if svc, err := m.provider.Payments(); err == nil && svc != nil {
		svc.Stop()
	}
}

// --- QRIS ---

// SetStaticQris binds a static QRIS payload to the active provider and its
// payment service.
func (m *Manager) SetStaticQris(payload string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.provider == nil {
		return fmt.Errorf("no Shopee session; login first")
	}
	if err := m.provider.SetStaticQris(payload); err != nil {
		return err
	}
	// Propagate to the active payment service so it can generate dynamic QRIS.
	if svc, err := m.provider.Payments(); err == nil && svc != nil {
		svc.SetStaticQris(payload)
	}
	return nil
}

// --- Payments ---

// PaymentInfo is a frontend-safe payment summary.
type PaymentInfo struct {
	ID           string `json:"id"`
	BaseAmount   int64  `json:"baseAmount"`
	UniqueAmount int64  `json:"uniqueAmount"`
	Status       string `json:"status"`
	Reference    string `json:"reference"`
	QRString     string `json:"qrString"`
	CreatedAt    int64  `json:"createdAt"`
	ExpiresAt    int64  `json:"expiresAt"`
}

// CreatePayment creates a new payment intent with a unique amount and dynamic
// QRIS. The static QRIS must have been set first.
func (m *Manager) CreatePayment(amount int64, reference string) (*PaymentInfo, error) {
	m.mu.Lock()
	provider := m.provider
	m.mu.Unlock()

	if provider == nil {
		return nil, fmt.Errorf("no Shopee session; login first")
	}

	p, err := provider.CreatePayment(context.Background(), amount, reference)
	if err != nil {
		return nil, err
	}
	return paymentToInfo(p), nil
}

// CancelPayment cancels a pending payment.
func (m *Manager) CancelPayment(id string) error {
	m.mu.Lock()
	provider := m.provider
	m.mu.Unlock()

	if provider == nil {
		return fmt.Errorf("no Shopee session; login first")
	}
	_, err := provider.CancelPayment(context.Background(), id)
	return err
}

// GetPayment returns a payment by id.
func (m *Manager) GetPayment(ctx context.Context, id string) (*PaymentInfo, error) {
	p, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return paymentToInfo(p), nil
}

// ListPayments returns recent payments, newest first.
func (m *Manager) ListPayments(ctx context.Context, limit int) ([]PaymentInfo, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	list, err := m.store.ListAll(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentInfo, len(list))
	for i := range list {
		out[i] = *paymentToInfo(&list[i])
	}
	return out, nil
}

func paymentToInfo(p *core.Payment) *PaymentInfo {
	if p == nil {
		return nil
	}
	return &PaymentInfo{
		ID:           p.ID,
		BaseAmount:   p.BaseAmount,
		UniqueAmount: p.UniqueAmount,
		Status:       string(p.Status),
		Reference:    p.Reference,
		QRString:     p.QRString,
		CreatedAt:    p.CreatedAt.UnixMilli(),
		ExpiresAt:    p.ExpiresAt.UnixMilli(),
	}
}

// SessionInfo is the current session status for the frontend.
type SessionInfo struct {
	LoggedIn     bool   `json:"loggedIn"`
	MerchantName string `json:"merchantName,omitempty"`
	StoreID      string `json:"storeId,omitempty"`
	NeedsRelogin bool   `json:"needsRelogin"`
}

// GetSessionInfo returns the current payment session status. If the merchant
// token has expired, it attempts a silent refresh first (no OTP required).
// Only when the passport account session is also dead does it report
// NeedsRelogin=true.
func (m *Manager) GetSessionInfo() *SessionInfo {
	m.mu.Lock()
	provider := m.provider
	m.mu.Unlock()

	if provider == nil {
		return &SessionInfo{LoggedIn: false}
	}

	if !provider.Authenticated() {
		// Try silent token refresh — no OTP needed if the passport account
		// session is still alive.
		refreshed, err := provider.RefreshSession(context.Background())
		if err != nil || refreshed == nil {
			m.mu.Lock()
			m.provider = nil
			m.started = false
			m.mu.Unlock()
			return &SessionInfo{LoggedIn: false, NeedsRelogin: true}
		}

		// Persist the refreshed session.
		data, _ := json.Marshal(refreshed)
		_ = m.store.SaveSession(context.Background(), "shopee", refreshed.AccountID, data)

		// Restart polling with the new token.
		m.mu.Lock()
		m.stopServiceLocked()
		m.started = false
		m.mu.Unlock()
		m.startService()
	}

	m.mu.Lock()
	session := m.provider.ExportSession()
	m.mu.Unlock()

	if session == nil {
		return &SessionInfo{LoggedIn: false}
	}

	return &SessionInfo{
		LoggedIn:     true,
		MerchantName: session.Merchant.Name,
		StoreID:      session.StoreID,
	}
}

// Close shuts down the manager.
func (m *Manager) Close() error {
	m.mu.Lock()
	m.stopServiceLocked()
	m.mu.Unlock()
	return m.store.Close()
}
