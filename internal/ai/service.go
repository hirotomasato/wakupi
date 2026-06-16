package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
	ProviderOllama    Provider = "ollama"
	ProviderGamAPI    Provider = "gamapi"
)

// ImageResult holds a single generated image.
type ImageResult struct {
	URL           string `json:"url"`
	RevisedPrompt string `json:"revisedPrompt,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Model         string `json:"model,omitempty"`
	B64JSON       string `json:"b64Json,omitempty"` // base64-encoded PNG for providers that return inline data
}

// ImageOptions carry per-request overrides for image generation.
type ImageOptions struct {
	Model       string `json:"model"`
	Size        string `json:"size"`
	Style       string `json:"style"`
	AspectRatio string `json:"aspectRatio"`
	Count       int    `json:"count"`
}

type Config struct {
	Provider Provider `json:"provider"`
	APIKey   string   `json:"apiKey"`
	BaseURL  string   `json:"baseUrl"`
	Model    string   `json:"model"`
	Enabled  bool     `json:"enabled"`
}

type Service struct {
	cfg          Config
	client       *http.Client
	streamClient *http.Client
}

func New(cfg Config) *Service {
	return &Service{
		cfg:          cfg,
		client:       &http.Client{Timeout: 60 * time.Second},
		streamClient: &http.Client{}, // no timeout: streams rely on ctx cancellation
	}
}

func (s *Service) Update(cfg Config) {
	s.cfg = cfg
}

func (s *Service) Config() Config {
	c := s.cfg
	if c.APIKey != "" {
		c.APIKey = strings.Repeat("*", 8)
	}
	return c
}

func (s *Service) Enabled() bool {
	return s.cfg.Enabled && (s.cfg.APIKey != "" || s.cfg.Provider == ProviderOllama)
}

// GenerateImage creates images from a text prompt using the configured provider.
// Not all providers support image generation (Anthropic, Ollama don't).
func (s *Service) GenerateImage(ctx context.Context, prompt string, opts ImageOptions) ([]ImageResult, error) {
	if !s.Enabled() {
		return nil, errors.New("AI not configured")
	}
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}
	switch s.cfg.Provider {
	case ProviderOpenAI:
		return s.generateOpenAIImage(ctx, prompt, opts)
	case ProviderGemini:
		return s.generateGeminiImage(ctx, prompt, opts)
	case ProviderGamAPI:
		return s.generateGamAPIImage(ctx, prompt, opts)
	}
	return nil, fmt.Errorf("provider %s does not support image generation", s.cfg.Provider)
}

// GamAPIImageSupport exposes GamAPI-specific endpoints for the frontend to query
// models, styles and aspect ratios before generating.

func (s *Service) ListGamAPIModels(ctx context.Context) ([]string, error) {
	return s.listGamAPIModels(ctx)
}

func (s *Service) ListGamAPIStyles(ctx context.Context) (map[string]string, error) {
	return s.listGamAPIStyles(ctx)
}

func (s *Service) ListGamAPIAspectRatios(ctx context.Context) (map[string]string, error) {
	return s.listGamAPIAspectRatios(ctx)
}

// Chat is the unified prompt entrypoint. Returns a single completion string.
func (s *Service) Chat(ctx context.Context, system, user string) (string, error) {
	if !s.Enabled() {
		return "", errors.New("AI not configured")
	}
	switch s.cfg.Provider {
	case ProviderOpenAI:
		return s.callOpenAI(ctx, system, user)
	case ProviderAnthropic:
		return s.callAnthropic(ctx, system, user)
	case ProviderGemini:
		return s.callGemini(ctx, system, user)
	case ProviderOllama:
		return s.callOllama(ctx, system, user)
	case ProviderGamAPI:
		return "", fmt.Errorf("GamAPI hanya mendukung image generation — gunakan tab Image di Playground")
	}
	return "", fmt.Errorf("unknown provider: %s", s.cfg.Provider)
}

// SuggestReplies returns 3 short reply suggestions for a given chat snippet.
func (s *Service) SuggestReplies(ctx context.Context, contactName, lastMessages string) ([]string, error) {
	if !s.Enabled() {
		return nil, nil
	}
	sys := "You are a WhatsApp assistant. Suggest 3 short, natural reply options (max 1 sentence each) in the same language as the conversation. Output only a JSON array of 3 strings, nothing else."
	user := fmt.Sprintf("Contact: %s\n\nLast messages:\n%s\n\nReply suggestions:", contactName, lastMessages)
	out, err := s.Chat(ctx, sys, user)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if i := strings.Index(out, "["); i >= 0 {
		out = out[i:]
	}
	if i := strings.LastIndex(out, "]"); i >= 0 {
		out = out[:i+1]
	}
	var arr []string
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		return nil, err
	}
	if len(arr) > 3 {
		arr = arr[:3]
	}
	return arr, nil
}

// Summarize returns a 1-paragraph summary of a chat snippet.
func (s *Service) Summarize(ctx context.Context, text string) (string, error) {
	sys := "Summarize the following WhatsApp conversation in 2-3 sentences in the same language as the input. Be concise and neutral."
	return s.Chat(ctx, sys, text)
}

// Ping sends a minimal request to validate key + model + endpoint in one shot.
// It bypasses the Enabled() gate so the user can test a config before saving it.
func (s *Service) Ping(ctx context.Context) error {
	if s.cfg.Provider != ProviderOllama && s.cfg.APIKey == "" {
		return errors.New("API key kosong")
	}
	switch s.cfg.Provider {
	case ProviderOpenAI:
		_, err := s.callOpenAI(ctx, "Reply with OK.", "ping")
		return err
	case ProviderAnthropic:
		_, err := s.callAnthropic(ctx, "Reply with OK.", "ping")
		return err
	case ProviderGemini:
		_, err := s.callGemini(ctx, "Reply with OK.", "ping")
		return err
	case ProviderOllama:
		_, err := s.callOllama(ctx, "Reply with OK.", "ping")
		return err
	case ProviderGamAPI:
		return fmt.Errorf("GamAPI tidak mendukung chat — hanya image generation")
	}
	return fmt.Errorf("unknown provider: %s", s.cfg.Provider)
}

// ListModels fetches the provider's available model IDs (best-effort).
func (s *Service) ListModels(ctx context.Context) ([]string, error) {
	switch s.cfg.Provider {
	case ProviderOpenAI:
		return s.listOpenAIModels(ctx)
	case ProviderAnthropic:
		return s.listAnthropicModels(ctx)
	case ProviderGemini:
		return s.listGeminiModels(ctx)
	case ProviderOllama:
		return s.listOllamaModels(ctx)
	case ProviderGamAPI:
		return s.ListGamAPIModels(ctx)
	}
	return nil, fmt.Errorf("unknown provider: %s", s.cfg.Provider)
}

func (s *Service) getJSON(ctx context.Context, url string, headers map[string]string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%d: %s", resp.StatusCode, string(raw))
	}
	return json.Unmarshal(raw, out)
}

func (s *Service) listOpenAIModels(ctx context.Context) ([]string, error) {
	url := "https://api.openai.com/v1/models"
	if s.cfg.BaseURL != "" {
		base := strings.TrimSuffix(s.cfg.BaseURL, "/chat/completions")
		base = strings.TrimSuffix(base, "/")
		url = base + "/models"
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := s.getJSON(ctx, url, map[string]string{"Authorization": "Bearer " + s.cfg.APIKey}, &out); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func (s *Service) listAnthropicModels(ctx context.Context) ([]string, error) {
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	headers := map[string]string{
		"x-api-key":         s.cfg.APIKey,
		"anthropic-version": "2023-06-01",
	}
	if err := s.getJSON(ctx, "https://api.anthropic.com/v1/models", headers, &out); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func (s *Service) listGeminiModels(ctx context.Context) ([]string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + s.cfg.APIKey
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := s.getJSON(ctx, url, nil, &out); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		models = append(models, strings.TrimPrefix(m.Name, "models/"))
	}
	return models, nil
}

func (s *Service) listOllamaModels(ctx context.Context) ([]string, error) {
	root := "http://localhost:11434"
	if s.cfg.BaseURL != "" {
		if i := strings.Index(s.cfg.BaseURL, "/api/"); i >= 0 {
			root = s.cfg.BaseURL[:i]
		} else {
			root = strings.TrimSuffix(s.cfg.BaseURL, "/")
		}
	}
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := s.getJSON(ctx, root+"/api/tags", nil, &out); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

// === OpenAI ===

func (s *Service) callOpenAI(ctx context.Context, system, user string) (string, error) {
	url := s.cfg.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}
	model := s.cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.7,
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("empty response")
	}
	return out.Choices[0].Message.Content, nil
}

// === Anthropic ===

func (s *Service) callAnthropic(ctx context.Context, system, user string) (string, error) {
	url := s.cfg.BaseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}
	model := s.cfg.Model
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	body := map[string]interface{}{
		"model":      model,
		"max_tokens": 1024,
		"system":     system,
		"messages": []map[string]string{
			{"role": "user", "content": user},
		},
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	for _, c := range out.Content {
		if c.Type == "text" {
			return c.Text, nil
		}
	}
	return "", errors.New("empty response")
}

// === Gemini ===

func (s *Service) callGemini(ctx context.Context, system, user string) (string, error) {
	model := s.cfg.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, s.cfg.APIKey)
	body := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": system}},
		},
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": user}}},
		},
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gemini %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("empty response")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

// === Ollama (local) ===

func (s *Service) callOllama(ctx context.Context, system, user string) (string, error) {
	url := s.cfg.BaseURL
	if url == "" {
		url = "http://localhost:11434/api/chat"
	}
	model := s.cfg.Model
	if model == "" {
		model = "llama3.2"
	}
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"stream": false,
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("ollama %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}
