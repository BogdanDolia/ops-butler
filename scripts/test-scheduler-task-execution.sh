#!/bin/bash

# Test script to verify that the scheduler now executes tasks properly

echo "🧪 Testing: Scheduler Task Execution"
echo "===================================="

# Create a test task with a past due date so it gets picked up immediately
echo ""
echo "1. Creating a test task for the scheduler to execute"

current_time=$(date -u -d '1 minute ago' '+%Y-%m-%dT%H:%M:%SZ')
test_task_response=$(curl -s -X POST "http://localhost:8080/api/v1/tasks" \
    -H "Content-Type: application/json" \
    -d '{
        "task_type": "check_logs",
        "params": {
            "podName": "ops-butler-api-77dc7d54f7-grzvn",
            "namespace": "ops-butler",
            "taskType": "check_logs"
        },
        "state": "pending",
        "due_at": "'$current_time'",
        "origin": "scheduler_test",
        "created_by": 1
    }')

echo "📄 Task creation response: $test_task_response"

# Extract task ID
task_id=$(echo "$test_task_response" | grep -o '"ID":[0-9]*' | sed 's/"ID"://')
if [ -n "$task_id" ]; then
    echo "✅ Created test task ID: $task_id"
    echo "⏰ Due date set to: $current_time (1 minute ago)"
    
    echo ""
    echo "2. Waiting for scheduler to pick up and execute the task..."
    echo "   (The scheduler polls every 30 seconds, so this may take up to 1 minute)"
    
    # Wait and check task status periodically
    for i in {1..12}; do
        sleep 10
        echo "   Checking task status (attempt $i/12)..."
        
        task_status_response=$(curl -s "http://localhost:8080/api/v1/tasks/$task_id")
        task_state=$(echo "$task_status_response" | grep -o '"state":"[^"]*' | sed 's/"state":"//')
        
        echo "   Current task state: $task_state"
        
        if [ "$task_state" = "completed" ]; then
            echo "✅ Task completed by scheduler!"
            echo "📄 Final task status: $task_status_response"
            break
        elif [ "$task_state" = "failed" ]; then
            echo "❌ Task failed during execution"
            echo "📄 Final task status: $task_status_response"
            break
        elif [ "$task_state" = "running" ]; then
            echo "🔄 Task is currently running..."
        fi
        
        if [ $i -eq 12 ]; then
            echo "⚠️  Task was not executed within 2 minutes"
            echo "📄 Final task status: $task_status_response"
        fi
    done
else
    echo "❌ Failed to create test task"
fi

# Check scheduler logs
echo ""
echo "3. Checking scheduler logs for task execution evidence"
scheduler_logs=$(kubectl logs -n ops-butler -l app=ops-butler-scheduler --tail=20 --since=2m)
echo "📋 Recent scheduler logs:"
echo "$scheduler_logs"

# Check if the logs show task execution
if echo "$scheduler_logs" | grep -q "Executing.*task"; then
    echo "✅ Scheduler logs show task execution activity"
else
    echo "⚠️  No task execution activity found in scheduler logs"
fi

echo ""
echo "4. Checking current task states"
tasks_response=$(curl -s "http://localhost:8080/api/v1/tasks")
echo "📊 Current tasks:"
echo "$tasks_response" | grep -o '"state":"[^"]*' | sort | uniq -c

echo ""
echo "🎉 Scheduler task execution testing completed!"
echo ""
echo "📋 **Expected behavior:**"
echo "- Test task should change from 'pending' → 'running' → 'completed'"
echo "- Scheduler logs should show 'Executing task' and 'completed successfully'"
echo "- Task should have a completion timestamp and exit code 0"
echo ""
echo "🔧 **If issues found:**"
echo "- Check scheduler pod status: kubectl get pods -n ops-butler -l app=ops-butler-scheduler"
echo "- Check scheduler logs: kubectl logs -n ops-butler -l app=ops-butler-scheduler"
echo "- Verify scheduler configuration and database connectivity" 