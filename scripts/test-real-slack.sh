#!/bin/bash

# Test script for real Slack integration
set -e

echo "🚀 Testing Real Slack Integration..."

# Check if we have required environment variables
if [ -z "$SLACK_TOKEN" ]; then
    echo "❌ SLACK_TOKEN is required for real Slack testing"
    echo "💡 Set it with: export SLACK_TOKEN=xoxb-your-token"
    exit 1
fi

# Set environment variables for real Slack
export SLACK_ENABLED=true
export SLACK_DEMO_MODE=false
export LOG_LEVEL=debug
export SLACK_SIGNING_SECRET=${SLACK_SIGNING_SECRET:-""}
export SLACK_DEFAULT_CHANNEL=${SLACK_DEFAULT_CHANNEL:-"#ops-alerts"}

# Database settings
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=k8s_ops_portal

# API endpoint
API_URL=${API_URL:-"http://localhost:8080"}

echo "📡 API URL: $API_URL"
echo "🔑 Slack Token: ${SLACK_TOKEN:0:15}..."
echo "📢 Default Channel: $SLACK_DEFAULT_CHANNEL"
echo "🐛 Demo Mode: $SLACK_DEMO_MODE"

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
        echo "✅ SUCCESS - Check your Slack channel!"
        sleep 2  # Give time to see the message
    else
        echo "❌ FAILED"
        return 1
    fi
}

# Test health endpoint
test_endpoint "/health" "GET" "" "Health check"

# Test simple Slack message
test_endpoint "/api/v1/chatops/test/slack" "POST" '{
    "channel": "'$SLACK_DEFAULT_CHANNEL'",
    "message": "🚀 Hello from real Slack integration! This is a test message."
}' "Real Slack simple message"

# Test Slack reminder message with interactive buttons
test_endpoint "/api/v1/chatops/test/slack" "POST" '{
    "channel": "'$SLACK_DEFAULT_CHANNEL'",
    "message": "⏰ **Task Reminder** - Please check the production logs for any issues.",
    "task_id": 123
}' "Real Slack reminder message with interactive buttons"

# Test creating a task that will trigger ChatOps
echo ""
echo "🧪 Testing: Create task with ChatOps integration"
task_response=$(curl -s -X POST "$API_URL/api/v1/tasks" \
    -H "Content-Type: application/json" \
    -d '{
        "template_id": 1,
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

task_http_code=$(echo "$task_response" | grep -o "HTTP_[0-9]*" | sed 's/HTTP_//')
task_response_body=$(echo "$task_response" | sed 's/HTTP_[0-9]*$//')

echo "📊 Task Creation Status: $task_http_code"
echo "📄 Task Response: $task_response_body"

if [ "$task_http_code" -eq 200 ] || [ "$task_http_code" -eq 201 ]; then
    echo "✅ Task created successfully"
    
    # Extract task ID and execute it
    task_id=$(echo "$task_response_body" | grep -o '"id":[0-9]*' | sed 's/"id"://')
    if [ -n "$task_id" ]; then
        echo "🔄 Executing task $task_id..."
        
        test_endpoint "/api/v1/tasks/$task_id/execute" "POST" "" "Execute task $task_id"
    fi
else
    echo "❌ Failed to create task"
fi

echo ""
echo "🎉 Real Slack integration testing completed!"
echo ""
echo "📋 **Next Steps:**"
echo "1. Check your Slack channel: $SLACK_DEFAULT_CHANNEL"
echo "2. Try clicking the interactive buttons in the reminder message"
echo "3. Configure webhook URL in your Slack app:"
echo "   https://your-domain.com/api/v1/chatops/slack/webhook"
echo "4. Test slash commands: /ops-status, /ops-run, /ops-list"
echo ""
echo "🚀 **Deploy to Kubernetes:**"
echo "   kubectl apply -f deploy/local/chatops.yaml"
echo "   kubectl apply -f deploy/local/api.yaml"
echo ""
echo "🔍 **Debug logs:**"
echo "   kubectl logs -f deployment/ops-butler-api -n ops-butler" 