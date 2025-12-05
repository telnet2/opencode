#!/bin/bash
# Test all endpoints the TUI calls on startup

BASE="http://localhost:8080"

echo "=== Testing TUI initialization endpoints ==="
echo

endpoints=(
  "/config"
  "/config/providers"
  "/provider"
  "/agent"
  "/session"
  "/session/status"
  "/command"
  "/lsp"
  "/mcp"
  "/formatter"
  "/provider/auth"
)

for ep in "${endpoints[@]}"; do
  echo ">>> GET $ep"
  response=$(curl -s "$BASE$ep")
  # Try to parse as JSON
  if echo "$response" | jq . > /dev/null 2>&1; then
    echo "$response" | jq -c .
    echo "✓ Valid JSON"
  else
    echo "✗ INVALID JSON: $response"
  fi
  echo
done
