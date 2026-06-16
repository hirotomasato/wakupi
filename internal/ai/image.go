package ai

import (
	"bytes"
	"context"
	"encoding/base64"
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

// === GamAPI (gamapi.proaccess.app) ===

type gamAPIRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Style       string `json:"style,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
}

type gamAPIResponse struct {
	Success   bool   `json:"success"`
	ImageURL  string `json:"image_url"`
	AccountUsed string `json:"account_used"`
	Result    struct {
		Created int `json:"created"`
		Data    []struct {
			URL            string `json:"url"`
			RevisedPrompt  string `json:"revised_prompt"`
			B64JSON        string `json:"b64_json"`
			Metadata       struct {
				ID          string `json:"id"`
				UserID      string `json:"user_id"`
				WorkspaceID string `json:"workspace_id"`
				Width       int    `json:"width"`
				Height      int    `json:"height"`
				Model       string `json:"model"`
			} `json:"metadata"`
		} `json:"data"`
	} `json:"result"`
	Quota struct {
		Used      int  `json:"used"`
		Limit     int  `json:"limit"`
		Remaining *int `json:"remaining"`
		Unlimited bool `json:"unlimited"`
	} `json:"quota"`
}

func (s *Service) generateGamAPIImage(ctx context.Context, prompt string, opts ImageOptions) ([]ImageResult, error) {
	model := "imagen-3-flash"
	if opts.Model != "" {
		model = opts.Model
	}

	reqBody := gamAPIRequest{
		Model:       model,
		Prompt:      prompt,
		Style:       opts.Style,
		AspectRatio: opts.AspectRatio,
	}
	data, _ := json.Marshal(reqBody)

	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gamapi.proaccess.app/api"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/generate", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gamapi %d: %s", resp.StatusCode, string(raw))
	}

	var out gamAPIResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if !out.Success {
		return nil, fmt.Errorf("gamapi generation failed")
	}

	var results []ImageResult
	for _, d := range out.Result.Data {
		r := ImageResult{
			URL:           d.URL,
			RevisedPrompt: d.RevisedPrompt,
			Width:         d.Metadata.Width,
			Height:        d.Metadata.Height,
			Model:         d.Metadata.Model,
			B64JSON:       d.B64JSON,
		}
		if r.URL == "" && out.ImageURL != "" {
			r.URL = out.ImageURL
		}
		results = append(results, r)
	}
	return results, nil
}

// listGamAPIModels fetches available model IDs from GamAPI.
func (s *Service) listGamAPIModels(ctx context.Context) ([]string, error) {
	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gamapi.proaccess.app/api"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	var out struct {
		FreeModels []string `json:"free_models"`
	}
	if err := s.getJSON(ctx, baseURL+"/models", map[string]string{"Authorization": "Bearer " + s.cfg.APIKey}, &out); err != nil {
		return nil, err
	}
	if len(out.FreeModels) > 0 {
		return out.FreeModels, nil
	}

	// Fallback: parse the models map
	var out2 struct {
		Models map[string]struct {
			Name     string `json:"name"`
			Provider string `json:"provider"`
			Tier     string `json:"tier"`
		} `json:"models"`
	}
	if err := s.getJSON(ctx, baseURL+"/models", map[string]string{"Authorization": "Bearer " + s.cfg.APIKey}, &out2); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out2.Models))
	for id := range out2.Models {
		models = append(models, id)
	}
	return models, nil
}

// listGamAPIStyles fetches available style presets from GamAPI.
func (s *Service) listGamAPIStyles(ctx context.Context) (map[string]string, error) {
	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gamapi.proaccess.app/api"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	var out struct {
		Styles map[string]string `json:"styles"`
	}
	if err := s.getJSON(ctx, baseURL+"/styles", map[string]string{"Authorization": "Bearer " + s.cfg.APIKey}, &out); err != nil {
		return nil, err
	}
	return out.Styles, nil
}

// listGamAPIAspectRatios fetches available aspect ratios from GamAPI.
func (s *Service) listGamAPIAspectRatios(ctx context.Context) (map[string]string, error) {
	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gamapi.proaccess.app/api"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	var out struct {
		AspectRatios map[string]string `json:"aspect_ratios"`
	}
	if err := s.getJSON(ctx, baseURL+"/aspect-ratios", map[string]string{"Authorization": "Bearer " + s.cfg.APIKey}, &out); err != nil {
		return nil, err
	}
	return out.AspectRatios, nil
}

// decodeBase64JSON is a helper for when the frontend needs to convert a base64
// image payload into binary (e.g. to save to a temp file for WhatsApp upload).
func DecodeBase64JSON(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}
