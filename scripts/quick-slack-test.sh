#!/bin/bash

# Quick test script for your real Slack tokens
set -e

echo "🚀 Quick Slack Integration Test"
echo "================================="

# Use your actual tokens from the ConfigMap
export SLACK_TOKEN="xoxb-8948822092324-9217588031638-SA83KBY2H0Zgie0ZKQu43cEv"
export SLACK_SIGNING_SECRET="aea37904257a9f07f1df03ee860f9594"
export SLACK_DEFAULT_CHANNEL="#ops-alerts"
export SLACK_ENABLED=true
export SLACK_DEMO_MODE=false
export LOG_LEVEL=debug

echo "🔑 Token: ${SLACK_TOKEN:0:15}..."
echo "📢 Channel: $SLACK_DEFAULT_CHANNEL"
echo "🚫 Demo Mode: $SLACK_DEMO_MODE"
echo ""

# API endpoint
API_URL=${API_URL:-"http://localhost:8080"}

# Quick test function
quick_test() {
    local description=$1
    local endpoint=$2
    local data=$3
    
    echo "🧪 $description"
    
    response=$(curl -s -X POST "$API_URL$endpoint" \
        -H "Content-Type: application/json" \
        -d "$data" \
        -w "HTTP_%{http_code}")
    
    http_code=$(echo "$response" | grep -o "HTTP_[0-9]*" | sed 's/HTTP_//')
    response_body=$(echo "$response" | sed 's/HTTP_[0-9]*$//')
    
    if [ "$http_code" -eq 200 ]; then
        echo "✅ SUCCESS - Check your Slack!"
        echo "📄 Response: $response_body"
    else
        echo "❌ FAILED (Status: $http_code)"
        echo "📄 Response: $response_body"
    fi
    echo ""
}

# Test 1: Simple message
quick_test "Simple Slack message" "/api/v1/chatops/test/slack" '{
    "channel": "#ops-alerts",
    "message": "🚀 Quick test from Ops Butler!"
}'

# Test 2: Interactive buttons
quick_test "Interactive reminder message" "/api/v1/chatops/test/slack" '{
    "channel": "#ops-alerts",
    "message": "⏰ **Task Reminder**: Check production logs",
    "task_id": 999
}'

echo "🎉 Quick test completed!"
echo ""
echo "📋 Next steps:"
echo "1. Check your Slack channel: $SLACK_DEFAULT_CHANNEL"
echo "2. Click the interactive buttons to test"
echo "3. Deploy to Kubernetes with: kubectl apply -f deploy/local/"
echo ""
echo "🚀 For full testing, run: ./scripts/test-real-slack.sh" 