package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatMessage is one turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// ChatOptions tweaks a streaming request. Zero values fall back to provider/config defaults.
type ChatOptions struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	System      string  `json:"system"`
}

// StreamFunc receives incremental text deltas as they arrive.
type StreamFunc func(delta string)

// ChatStream runs a multi-turn completion and streams text deltas through onDelta.
// It blocks until the stream finishes, ctx is cancelled, or an error occurs.
func (s *Service) ChatStream(ctx context.Context, msgs []ChatMessage, opts ChatOptions, onDelta StreamFunc) error {
	if !s.Enabled() {
		return errors.New("AI not configured")
	}
	switch s.cfg.Provider {
	case ProviderOpenAI:
		return s.streamOpenAI(ctx, msgs, opts, onDelta)
	case ProviderAnthropic:
		return s.streamAnthropic(ctx, msgs, opts, onDelta)
	case ProviderGemini:
		return s.streamGemini(ctx, msgs, opts, onDelta)
	case ProviderOllama:
		return s.streamOllama(ctx, msgs, opts, onDelta)
	case ProviderGamAPI:
		return fmt.Errorf("GamAPI hanya mendukung image generation — gunakan tab Image di Playground")
	}
	return fmt.Errorf("unknown provider: %s", s.cfg.Provider)
}

func (s *Service) modelOr(fallback string, opts ChatOptions) string {
	if opts.Model != "" {
		return opts.Model
	}
	if s.cfg.Model != "" {
		return s.cfg.Model
	}
	return fallback
}

func temperatureOr(opts ChatOptions) float64 {
	if opts.Temperature > 0 {
		return opts.Temperature
	}
	return 0.7
}

// prependSystem injects a system turn at the front when one is supplied via opts
// and not already present in the message list.
func prependSystem(msgs []ChatMessage, system string) []ChatMessage {
	if system == "" {
		return msgs
	}
	if len(msgs) > 0 && msgs[0].Role == "system" {
		return msgs
	}
	out := make([]ChatMessage, 0, len(msgs)+1)
	out = append(out, ChatMessage{Role: "system", Content: system})
	return append(out, msgs...)
}

// === OpenAI (SSE) ===

func (s *Service) streamOpenAI(ctx context.Context, msgs []ChatMessage, opts ChatOptions, onDelta StreamFunc) error {
	url := s.cfg.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}
	msgs = prependSystem(msgs, opts.System)
	apiMsgs := make([]map[string]string, 0, len(msgs))
	for _, m := range msgs {
		apiMsgs = append(apiMsgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	body := map[string]interface{}{
		"model":       s.modelOr("gpt-4o-mini", opts),
		"messages":    apiMsgs,
		"temperature": temperatureOr(opts),
		"stream":      true,
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	resp, err := s.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
	}
	return scanSSE(ctx, resp.Body, func(payload string) error {
		if payload == "[DONE]" {
			return errStopStream
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return nil // skip keep-alive / non-JSON lines
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onDelta(chunk.Choices[0].Delta.Content)
		}
		return nil
	})
}

// === Anthropic (SSE) ===

func (s *Service) streamAnthropic(ctx context.Context, msgs []ChatMessage, opts ChatOptions, onDelta StreamFunc) error {
	url := s.cfg.BaseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}
	// Anthropic keeps system separate from the message list.
	system := opts.System
	apiMsgs := make([]map[string]string, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			if system == "" {
				system = m.Content
			}
			continue
		}
		apiMsgs = append(apiMsgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	body := map[string]interface{}{
		"model":       s.modelOr("claude-haiku-4-5-20251001", opts),
		"max_tokens":  4096,
		"messages":    apiMsgs,
		"temperature": temperatureOr(opts),
		"stream":      true,
	}
	if system != "" {
		body["system"] = system
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := s.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(raw))
	}
	return scanSSE(ctx, resp.Body, func(payload string) error {
		var evt struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(payload), &evt); err != nil {
			return nil
		}
		switch evt.Type {
		case "content_block_delta":
			if evt.Delta.Text != "" {
				onDelta(evt.Delta.Text)
			}
		case "message_stop":
			return errStopStream
		}
		return nil
	})
}

// === Gemini (streamGenerateContent SSE) ===

func (s *Service) streamGemini(ctx context.Context, msgs []ChatMessage, opts ChatOptions, onDelta StreamFunc) error {
	model := s.modelOr("gemini-1.5-flash", opts)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", model, s.cfg.APIKey)

	system := opts.System
	contents := make([]map[string]interface{}, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			if system == "" {
				system = m.Content
			}
			continue
		}
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]string{{"text": m.Content}},
		})
	}
	body := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature": temperatureOr(opts),
		},
	}
	if system != "" {
		body["system_instruction"] = map[string]interface{}{
			"parts": []map[string]string{{"text": system}},
		}
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini %d: %s", resp.StatusCode, string(raw))
	}
	return scanSSE(ctx, resp.Body, func(payload string) error {
		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return nil
		}
		for _, c := range chunk.Candidates {
			for _, p := range c.Content.Parts {
				if p.Text != "" {
					onDelta(p.Text)
				}
			}
		}
		return nil
	})
}

// === Ollama (newline-delimited JSON) ===

func (s *Service) streamOllama(ctx context.Context, msgs []ChatMessage, opts ChatOptions, onDelta StreamFunc) error {
	url := s.cfg.BaseURL
	if url == "" {
		url = "http://localhost:11434/api/chat"
	}
	msgs = prependSystem(msgs, opts.System)
	apiMsgs := make([]map[string]string, 0, len(msgs))
	for _, m := range msgs {
		apiMsgs = append(apiMsgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	body := map[string]interface{}{
		"model":    s.modelOr("llama3.2", opts),
		"messages": apiMsgs,
		"stream":   true,
		"options":  map[string]interface{}{"temperature": temperatureOr(opts)},
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama %d: %s", resp.StatusCode, string(raw))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Message.Content != "" {
			onDelta(chunk.Message.Content)
		}
		if chunk.Done {
			break
		}
	}
	return scanner.Err()
}

// errStopStream is a sentinel returned by SSE handlers to end the scan cleanly.
var errStopStream = errors.New("stream stop")

// scanSSE reads a Server-Sent Events body, extracting "data:" payloads and
// passing them to handle. Cancellation is honored between events.
func scanSSE(ctx context.Context, body io.Reader, handle func(payload string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if err := handle(payload); err != nil {
			if errors.Is(err, errStopStream) {
				return nil
			}
			return err
		}
	}
	return scanner.Err()
}
