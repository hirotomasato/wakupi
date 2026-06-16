//go:build ignore

// Test script: AI Image Generation through the Go service layer.
//
// Run:   go run scripts/test-image-gen.go [gamapi|openai|gemini|all]
//
// This tests the provider implementations directly, without needing Wails.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"wakupi/internal/ai"
)

func main() {
	provider := "gamapi"
	if len(os.Args) > 1 {
		provider = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if provider == "all" || provider == "gamapi" {
		fmt.Println("═══════════════════════════════════════")
		fmt.Println("  GAMAPI — Image Generation Test")
		fmt.Println("═══════════════════════════════════════")
		testGamAPI(ctx)
	}

	if provider == "all" || provider == "openai" {
		fmt.Println("\n═══════════════════════════════════════")
		fmt.Println("  OPENAI — DALL-E Test")
		fmt.Println("═══════════════════════════════════════")
		fmt.Println("(skipped — needs valid OpenAI key in env OPENAI_API_KEY)")
		key := os.Getenv("OPENAI_API_KEY")
		if key != "" {
			testOpenAI(ctx, key)
		}
	}

	if provider == "all" || provider == "gemini" {
		fmt.Println("\n═══════════════════════════════════════")
		fmt.Println("  GEMINI — Imagen Test")
		fmt.Println("═══════════════════════════════════════")
		fmt.Println("(skipped — needs valid Gemini key in env GEMINI_API_KEY)")
		key := os.Getenv("GEMINI_API_KEY")
		if key != "" {
			testGemini(ctx, key)
		}
	}
}

func testGamAPI(ctx context.Context) {
	svc := ai.New(ai.Config{
		Provider: ai.ProviderGamAPI,
		APIKey:   "gpi_mccY3Gib6cab2Ts7CESGVp8Evs0EtbOd",
		Enabled:  true,
	})

	// List models
	fmt.Println("\n📋 Models:")
	models, err := svc.ListGamAPIModels(ctx)
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  Total: %d\n", len(models))
	for i, m := range models {
		if i < 5 {
			fmt.Printf("  • %s\n", m)
		}
	}
	if len(models) > 5 {
		fmt.Printf("  ... and %d more\n", len(models)-5)
	}

	// List styles
	styles, _ := svc.ListGamAPIStyles(ctx)
	fmt.Printf("\n🎨 Styles: %d\n", len(styles))
	for k, v := range styles {
		fmt.Printf("  • %s → %s\n", k, v)
	}

	// List ratios
	ratios, _ := svc.ListGamAPIAspectRatios(ctx)
	fmt.Printf("\n📐 Ratios: %d\n", len(ratios))
	for k, v := range ratios {
		fmt.Printf("  • %s → %s\n", k, v)
	}

	// Generate image
	fmt.Println("\n🖼️  Generating image...")
	results, err := svc.GenerateImage(ctx, "a cute cat wearing sunglasses, digital art", ai.ImageOptions{
		Model:       "imagen-3-flash",
		Style:       "illustration",
		AspectRatio: "square",
	})
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ Generated %d image(s)\n", len(results))
	for i, r := range results {
		fmt.Printf("  [%d] URL: %s\n", i+1, r.URL)
		if r.Width > 0 || r.Height > 0 {
			fmt.Printf("      Size: %d×%d\n", r.Width, r.Height)
		}
		if r.Model != "" {
			fmt.Printf("      Model: %s\n", r.Model)
		}
		if r.RevisedPrompt != "" && r.RevisedPrompt != "a cute cat wearing sunglasses, digital art" {
			fmt.Printf("      Revised: %s\n", r.RevisedPrompt)
		}
	}
}

func testOpenAI(ctx context.Context, apiKey string) {
	svc := ai.New(ai.Config{
		Provider: ai.ProviderOpenAI,
		APIKey:   apiKey,
		Enabled:  true,
	})

	results, err := svc.GenerateImage(ctx, "a cute cat wearing sunglasses", ai.ImageOptions{
		Model: "dall-e-3",
		Size:  "1024x1024",
	})
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ Generated %d image(s)\n", len(results))
	for i, r := range results {
		fmt.Printf("  Image %d: %s\n", i+1, r.URL)
	}
}

func testGemini(ctx context.Context, apiKey string) {
	svc := ai.New(ai.Config{
		Provider: ai.ProviderGemini,
		APIKey:   apiKey,
		Enabled:  true,
	})

	results, err := svc.GenerateImage(ctx, "a cute cat wearing sunglasses", ai.ImageOptions{
		Count: 1,
	})
	if err != nil {
		fmt.Printf("  ❌ Error: %v\n", err)
		return
	}
	fmt.Printf("  ✅ Generated %d image(s)\n", len(results))
	if len(results) > 0 {
		fmt.Printf("  Image: data URI (%s bytes)\n", formatLen(results[0].B64JSON))
	}
}

func formatLen(s string) string {
	l := len(s)
	if l < 1024 {
		return fmt.Sprintf("%d", l)
	}
	return fmt.Sprintf("%.1f KB", float64(l)/1024)
}

var _ = strings.TrimSpace
