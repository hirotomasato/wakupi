#!/usr/bin/env bash
# Test script for GamAPI Image Generation API
# API: https://gamapi.proaccess.app
# Key: gpi_mccY3Gib6cab2Ts7CESGVp8Evs0EtbOd
#
# Usage: bash scripts/test-gamapi.sh [generate|models|styles|ratios|all]

set -euo pipefail

API_BASE="https://gamapi.proaccess.app/api"
API_KEY="gpi_mccY3Gib6cab2Ts7CESGVp8Evs0EtbOd"
CMD="${1:-all}"

if [[ "$CMD" == "all" || "$CMD" == "models" ]]; then
  echo "========================================"
  echo "📋 MODELS"
  echo "========================================"
  curl -sS "$API_BASE/models" \
    -H "Authorization: Bearer $API_KEY" | jq
  echo -e "\n\n"
fi

if [[ "$CMD" == "all" || "$CMD" == "styles" ]]; then
  echo "========================================"
  echo "🎨 STYLES"
  echo "========================================"
  curl -sS "$API_BASE/styles" \
    -H "Authorization: Bearer $API_KEY" | jq
  echo -e "\n\n"
fi

if [[ "$CMD" == "all" || "$CMD" == "ratios" ]]; then
  echo "========================================"
  echo "📐 ASPECT RATIOS"
  echo "========================================"
  curl -sS "$API_BASE/aspect-ratios" \
    -H "Authorization: Bearer $API_KEY" | jq
  echo -e "\n\n"
fi

if [[ "$CMD" == "all" || "$CMD" == "generate" ]]; then
  echo "========================================"
  echo "🖼️  GENERATE IMAGE"
  echo "========================================"
  echo "Model: imagen-3-flash"
  echo "Prompt: a cute cat wearing sunglasses, digital art"
  echo "Style: illustration"
  echo "Ratio: square"
  echo "---"

  RESPONSE=$(curl -sS "$API_BASE/generate" \
    -X POST \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
      "model": "imagen-3-flash",
      "prompt": "a cute cat wearing sunglasses, digital art",
      "style": "illustration",
      "aspect_ratio": "square"
    }')

  echo "$RESPONSE" | jq

  # Extract and show the image URL if success
  URL=$(echo "$RESPONSE" | jq -r '.image_url // empty')
  if [[ -n "$URL" ]]; then
    echo -e "\n✅ Image URL: $URL"
    echo "   Open in browser to see the result!"
  fi
  echo -e "\n\n"
fi

echo "========================================"
echo "✅ DONE"
echo "========================================"
