package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// === OpenAI DALL-E ===

type openAIImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

type openAIImageResponse struct {
	Data []struct {
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
}

func (s *Service) generateOpenAIImage(ctx context.Context, prompt string, opts ImageOptions) ([]ImageResult, error) {
	model := "dall-e-3"
	if opts.Model != "" {
		model = opts.Model
	}
	size := "1024x1024"
	if opts.Size != "" {
		size = opts.Size
	}
	n := 1
	if opts.Count > 1 {
		n = opts.Count
	}
	// DALL-E 3 only supports n=1
	if model == "dall-e-3" {
		n = 1
	}

	body := openAIImageRequest{
		Model:  model,
		Prompt: prompt,
		N:      n,
		Size:   size,
	}
	data, _ := json.Marshal(body)

	url := "https://api.openai.com/v1/images/generations"
	if s.cfg.BaseURL != "" {
		base := strings.TrimSuffix(s.cfg.BaseURL, "/chat/completions")
		base = strings.TrimSuffix(base, "/")
		url = base + "/images/generations"
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai image %d: %s", resp.StatusCode, string(raw))
	}

	var out openAIImageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	results := make([]ImageResult, 0, len(out.Data))
	for _, d := range out.Data {
		results = append(results, ImageResult{
			URL:           d.URL,
			RevisedPrompt: d.RevisedPrompt,
			Model:         model,
		})
	}
	return results, nil
}

// === Gemini Imagen ===

type geminiImageResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (s *Service) generateGeminiImage(ctx context.Context, prompt string, opts ImageOptions) ([]ImageResult, error) {
	model := "imagen-3.0-generate-001"
	if opts.Model != "" {
		model = opts.Model
	}
	count := 1
	if opts.Count > 1 && opts.Count <= 4 {
		count = opts.Count
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:predict?key=%s", model, s.cfg.APIKey)

	instances := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		instances[i] = map[string]interface{}{"prompt": prompt}
	}
	body := map[string]interface{}{
		"instances": instances,
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gemini image %d: %s", resp.StatusCode, string(raw))
	}

	var out geminiImageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	var results []ImageResult
	for _, c := range out.Candidates {
		for _, p := range c.Content.Parts {
			if p.InlineData == nil || p.InlineData.Data == "" {
				continue
			}
			results = append(results, ImageResult{
				URL:     "data:" + p.InlineData.MimeType + ";base64," + p.InlineData.Data,
				B64JSON: p.InlineData.Data,
				Model:   model,
			})
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("gemini returned no images")
	}
	return results, nil
}
