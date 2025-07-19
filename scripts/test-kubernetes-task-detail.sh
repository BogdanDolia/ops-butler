#!/bin/bash

# Test script for task detail functionality in Kubernetes environment

echo "🧪 Testing: Task Detail Functionality in Kubernetes"
echo "=================================================="

# Test API endpoints directly (these should work)
echo ""
echo "1. Testing: Direct API access"
echo "API Base URL: http://localhost:8080"

# Test tasks list
echo "Testing tasks list..."
tasks_response=$(curl -s http://localhost:8080/api/v1/tasks)
if echo "$tasks_response" | grep -q '"tasks"'; then
    echo "✅ Tasks list API working"
    
    # Extract a task ID for testing
    task_id=$(echo "$tasks_response" | grep -o '"ID":[0-9]*' | head -1 | sed 's/"ID"://')
    if [ -n "$task_id" ]; then
        echo "📍 Found task ID: $task_id"
        
        # Test individual task endpoint
        echo "Testing individual task API..."
        task_response=$(curl -s http://localhost:8080/api/v1/tasks/$task_id)
        if echo "$task_response" | grep -q '"ID":'; then
            echo "✅ Individual task API working"
            echo "📄 Task: $task_response"
        else
            echo "❌ Individual task API failed"
        fi
        
        # Test task execution endpoint
        echo "Testing task execution API..."
        execute_response=$(curl -s -X POST http://localhost:8080/api/v1/tasks/$task_id/execute)
        if echo "$execute_response" | grep -q -E '(success|completed|running)'; then
            echo "✅ Task execution API working"
        else
            echo "⚠️  Task execution API response: $execute_response"
        fi
        
        # Test task logs endpoint  
        echo "Testing task logs API..."
        logs_response=$(curl -s http://localhost:8080/api/v1/tasks/$task_id/logs)
        if echo "$logs_response" | grep -q '"logs"'; then
            echo "✅ Task logs API working"
        else
            echo "⚠️  Task logs API response: $logs_response"
        fi
    else
        echo "❌ No task ID found in response"
    fi
else
    echo "❌ Tasks list API failed"
    echo "📄 Response: $tasks_response"
fi

# Test web UI accessibility (through proxy)
echo ""
echo "2. Testing: Web UI Proxy Functionality"

# Test if web UI can proxy API calls
echo "Testing web UI API proxy..."
web_api_response=$(curl -s http://localhost:8080/api/v1/tasks)
if echo "$web_api_response" | grep -q '"tasks"'; then
    echo "✅ Web UI API proxy working"
else
    echo "❌ Web UI API proxy failed"
    echo "📄 Response: $web_api_response"
fi

# Check web service status
echo ""
echo "3. Testing: Kubernetes Web Service Status"
web_pod_status=$(kubectl get pods -n ops-butler -l app=ops-butler-web -o jsonpath='{.items[0].status.phase}')
echo "Web pod status: $web_pod_status"

if [ "$web_pod_status" = "Running" ]; then
    echo "✅ Web pod is running"
    
    # Check if the web pod is ready
    web_pod_ready=$(kubectl get pods -n ops-butler -l app=ops-butler-web -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}')
    echo "Web pod ready status: $web_pod_ready"
    
    if [ "$web_pod_ready" = "True" ]; then
        echo "✅ Web pod is ready"
    else
        echo "❌ Web pod is not ready"
        echo "Pod logs:"
        kubectl logs -n ops-butler -l app=ops-butler-web --tail=10
    fi
else
    echo "❌ Web pod is not running"
    kubectl get pods -n ops-butler -l app=ops-butler-web
fi

echo ""
echo "🎉 Kubernetes Task Detail Testing completed!"
echo ""
echo "📋 **Next Steps to test in browser:**"
echo "1. Open http://localhost:8080/tasks"
echo "2. Click 'View' on any task to go to /tasks/[id]"
echo "3. The page should load task details without 'Failed to load task' error"
echo "4. Try clicking 'Execute Task' if the task is pending"
echo "5. Try clicking 'Get Logs' to retrieve task logs"
echo ""
echo "🔧 **If issues persist:**"
echo "- Check browser console for any CORS or API errors"
echo "- Verify the web UI environment variables: kubectl exec -it -n ops-butler deployment/ops-butler-web -- env | grep API" 