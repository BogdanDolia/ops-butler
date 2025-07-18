#!/bin/bash

# Test webhook functionality
set -e

echo "🔗 Testing Slack Webhook Integration"
echo "===================================="

# API endpoint
API_URL=${API_URL:-"https://9f2c1fa5290d.ngrok-free.app"}

echo "📡 API URL: $API_URL"
echo ""

# Test 1: Health check
echo "🏥 Health Check:"
curl -s "$API_URL/health" | jq '.'
echo ""

# Test 2: URL verification
echo "🔐 URL Verification Test:"
curl -s -X POST "$API_URL/api/v1/chatops/slack/webhook" \
  -H "Content-Type: application/json" \
  -d '{"type": "url_verification", "challenge": "test123"}' | jq '.'
echo ""

# Test 3: Simulated interactive component
echo "🎛️ Interactive Component Test:"
curl -s -X POST "$API_URL/api/v1/chatops/slack/webhook" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'payload={"type":"block_actions","user":{"id":"U123","name":"testuser"},"actions":[{"action_id":"run_now","value":"task_123"}]}' | jq '.'
echo ""

# Test 4: Send test message
echo "💬 Test Message:"
curl -s -X POST "$API_URL/api/v1/chatops/test/slack" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "#ops-alerts",
    "message": "🧪 Webhook test message",
    "task_id": 123
  }' | jq '.'
echo ""

echo "🎉 Webhook testing completed!"
echo ""
echo "📋 Next steps:"
echo "1. Check Slack app settings at https://api.slack.com/apps"
echo "2. Set Request URL to: $API_URL/api/v1/chatops/slack/webhook"
echo "3. Test interactive buttons in your Slack channel"
echo "4. Check server logs for webhook requests" 