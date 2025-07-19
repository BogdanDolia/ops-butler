#!/bin/bash

# Test script for ChatOps functionality
set -e

echo "🚀 Testing ChatOps functionality..."

# Set environment variables for demo mode
export SLACK_ENABLED=true
export SLACK_DEMO_MODE=true
export LOG_LEVEL=debug
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=k8s_ops_portal

# API endpoint (adjust as needed)
API_URL=${API_URL:-"http://localhost:8080"}

echo "📡 API URL: $API_URL"
echo "🔧 Slack Demo Mode: $SLACK_DEMO_MODE"

# Function to test API endpoint
test_endpoint() {
    local endpoint=$1
    local method=$2
    local data=$3
    local description=$4
    
    echo ""
    echo "🧪 Testing: $description"
    echo "📍 Endpoint: $method $endpoint"
    
    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -w "HTTP_%{http_code}")
    else
        response=$(curl -s -X $method "$API_URL$endpoint" \
            -w "HTTP_%{http_code}")
    fi
    
    # Extract HTTP status code
    http_code=$(echo "$response" | grep -o "HTTP_[0-9]*" | sed 's/HTTP_//')
    response_body=$(echo "$response" | sed 's/HTTP_[0-9]*$//')
    
    echo "📊 Status: $http_code"
    echo "📄 Response: $response_body"
    
    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        echo "✅ SUCCESS"
    else
        echo "❌ FAILED"
    fi
}

# Test health endpoint
test_endpoint "/health" "GET" "" "Health check"

# Test simple Slack message
test_endpoint "/api/v1/chatops/test/slack" "POST" '{
    "channel": "#general",
    "message": "Hello from ChatOps test script! 🚀"
}' "Simple Slack message"

# Test Slack reminder message with buttons
test_endpoint "/api/v1/chatops/test/slack" "POST" '{
    "channel": "#general",
    "message": "⏰ Task reminder: Please check the production logs",
    "task_id": 42
}' "Slack reminder message with buttons"

# Test Google Chat message
test_endpoint "/api/v1/chatops/test/googlechat" "POST" '{
    "space": "spaces/test-space",
    "message": "Hello from Google Chat test! 🤖"
}' "Simple Google Chat message"

# Test creating a task that will trigger ChatOps
echo ""
echo "🧪 Testing: Create task with ChatOps integration"
task_response=$(curl -s -X POST "$API_URL/api/v1/tasks" \
    -H "Content-Type: application/json" \
    -d '{
        "params": {
            "podName": "nginx-production",
            "namespace": "default",
            "chatType": "slack",
            "chatId": "'$SLACK_DEFAULT_CHANNEL'"
        },
        "state": "pending",
        "origin": "chatops_test"
    }' \
    -w "HTTP_%{http_code}")

echo ""
echo "🎉 ChatOps testing completed!"
echo ""
echo "💡 To run with real Slack integration:"
echo "   export SLACK_TOKEN=xoxb-your-slack-bot-token"
echo "   export SLACK_DEMO_MODE=false"
echo ""
echo "📚 For more info, check the logs in debug mode." 