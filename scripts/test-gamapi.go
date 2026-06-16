//go:build ignore

// Test script for GamAPI Image Generation API.
//
// Run:   go run scripts/test-gamapi.go
// This explores the API and generates a test image.
//
// API reference:
//   Base:   https://gamapi.proaccess.app/api
//   Auth:   Bearer gpi_mccY3Gib6cab2Ts7CESGVp8Evs0EtbOd
//   Endpoints:
//     GET  /models        — list available models
//     GET  /styles        — list style presets
//     GET  /aspect-ratios — list aspect ratios
//     POST /generate      — generate an image
//
// Generate request body:
//   {
//     "model":        string   [required] model ID from /models
//     "prompt":       string   [required] text description
//     "style":        string   [optional] style preset from /styles
//     "aspect_ratio": string   [optional] e.g. "square", "portrait9x16", "landscape16x9"
//   }
//
// Response (success):
//   {
//     "success":        bool
//     "generation_id":  int
//     "image_url":      string   — public CDN URL
//     "account_used":   string
//     "accounts_tried": int
//     "result": {
//       "created": int
//       "data": [{
//         "url":            string
//         "revised_prompt": string
//         "metadata": {
//           "id":           string
//           "user_id":      string
//           "workspace_id": string
//           "width":        int
//           "height":       int
//           "model":        string
//         }
//       }]
//     }
//     "quota": {
//       "used":      int
//       "limit":     int
//       "remaining": int|null
//       "unlimited": bool
//     }
//   }

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	apiBase = "https://gamapi.proaccess.app/api"
	apiKey  = "gpi_mccY3Gib6cab2Ts7CESGVp8Evs0EtbOd"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "models":
			getJSON("/models")
		case "styles":
			getJSON("/styles")
		case "ratios":
			getJSON("/aspect-ratios")
		case "generate":
			testGenerate()
		default:
			fmt.Printf("Usage: go run %s [models|styles|ratios|generate]\n", os.Args[0])
		}
		return
	}

	// Default: run all
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  GAMAPI — Full API Exploration")
	fmt.Println("═══════════════════════════════════════")

	getJSON("/models")
	getJSON("/styles")
	getJSON("/aspect-ratios")
	testGenerate()
}

func getJSON(path string) {
	fmt.Printf("\n📡 GET %s\n", path)
	fmt.Println(strings.Repeat("─", 55))

	resp := doRequest(http.MethodGet, path, nil)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	json.Indent(&pretty, body, "", "  ")
	fmt.Println(pretty.String())
}

func testGenerate() {
	fmt.Printf("\n🖼️  POST /generate\n")
	fmt.Println(strings.Repeat("─", 55))

	body := map[string]interface{}{
		"model":        "imagen-3-flash",
		"prompt":       "a cute cat wearing sunglasses, digital art",
		"style":        "illustration",
		"aspect_ratio": "square",
	}

	resp := doRequest(http.MethodPost, "/generate", body)
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	json.Indent(&pretty, raw, "", "  ")
	fmt.Println(pretty.String())

	// Extract image URL
	var result struct {
		Success   bool   `json:"success"`
		ImageURL  string `json:"image_url"`
	}
	json.Unmarshal(raw, &result)
	if result.Success && result.ImageURL != "" {
		fmt.Printf("\n✅ ✅ ✅ Image URL: %s\n", result.ImageURL)
		fmt.Println("   Open this URL in your browser to see the image!")
	}
}

func doRequest(method, path string, body interface{}) *http.Response {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, apiBase+path, reqBody)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("❌ HTTP %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}
	return resp
}
