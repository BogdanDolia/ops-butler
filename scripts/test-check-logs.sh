#!/bin/bash

# Test script for the "check logs" task type

# Create a "check logs" task
echo "Creating a 'check logs' task..."
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": 1,
    "task_type": "check_logs",
    "params": {
      "podName": "test-pod",
      "namespace": "default",
      "chatType": "slack",
      "chatId": "test-channel"
    },
    "state": "pending",
    "origin": "web",
    "created_by": 1
  }'

echo -e "\n\nListing tasks..."
curl -X GET http://localhost:8080/api/v1/tasks

# Get the task ID from the response (in a real script, we would parse the JSON response)
TASK_ID=1

# Execute the task
echo -e "\n\nExecuting task $TASK_ID..."
curl -X POST http://localhost:8080/api/v1/tasks/$TASK_ID/execute

# Get the task logs
echo -e "\n\nGetting logs for task $TASK_ID..."
curl -X GET http://localhost:8080/api/v1/tasks/$TASK_ID/logs

echo -e "\n\nTest completed."