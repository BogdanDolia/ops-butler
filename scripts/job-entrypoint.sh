#!/bin/bash
set -e

# Function to log messages
log() {
  echo "[$(date -Iseconds)] $1"
}

# Function to handle errors
handle_error() {
  log "ERROR: $1"
  exit 1
}

# Check required environment variables
if [ -z "$TASK_ID" ]; then
  handle_error "TASK_ID environment variable is required"
fi

if [ -z "$TASK_TYPE" ]; then
  handle_error "TASK_TYPE environment variable is required"
fi

log "Starting job for task $TASK_ID of type $TASK_TYPE"

# Parse task parameters if provided
if [ -n "$TASK_PARAMS" ]; then
  log "Task parameters: $TASK_PARAMS"
  # Use jq to extract parameters
  PARAMS=$(echo "$TASK_PARAMS" | jq -r 'to_entries | .[] | "\(.key)=\(.value)"')
  for PARAM in $PARAMS; do
    export "$PARAM"
    log "Set parameter: $PARAM"
  done
fi

# Execute task based on type
case "$TASK_TYPE" in
  "collect_logs")
    if [ -z "$pod_name" ]; then
      handle_error "pod_name parameter is required for collect_logs task"
    fi
    
    log "Collecting logs from pod $pod_name"
    
    # Check if namespace is provided
    if [ -n "$namespace" ]; then
      # Check if pod exists in the specified namespace
      log "Checking for pod $pod_name in namespace $namespace"
      if ! kubectl get pod "$pod_name" -n "$namespace" &>/dev/null; then
        handle_error "Pod $pod_name not found in namespace $namespace"
      fi
      
      # Collect logs from the specified namespace
      log "Retrieving logs from pod $pod_name in namespace $namespace"
      kubectl logs "$pod_name" -n "$namespace" --tail=1000 || handle_error "Failed to retrieve logs"
    else
      # Search for pod in all namespaces
      log "Searching for pod $pod_name in all namespaces"
      pod_info=$(kubectl get pod "$pod_name" --all-namespaces -o custom-columns=NAMESPACE:.metadata.namespace --no-headers 2>/dev/null)
      
      # Check if pod was found
      if [ -z "$pod_info" ]; then
        handle_error "Pod $pod_name not found in any namespace"
      fi
      
      # If multiple pods with the same name exist in different namespaces, use the first one
      found_namespace=$(echo "$pod_info" | head -n 1)
      log "Found pod $pod_name in namespace $found_namespace"
      
      # Collect logs
      log "Retrieving logs from pod $pod_name in namespace $found_namespace"
      kubectl logs "$pod_name" -n "$found_namespace" --tail=1000 || handle_error "Failed to retrieve logs"
    fi
    
    log "Logs collected successfully"
    ;;
    
  "status")
    log "Checking system status"
    
    # Get node information
    log "Node information:"
    kubectl get nodes -o wide || log "Failed to get node information"
    
    # Get pod information
    log "Pod information:"
    kubectl get pods --all-namespaces || log "Failed to get pod information"
    
    # Get resource usage
    log "Resource usage:"
    kubectl top nodes || log "Failed to get resource usage"
    
    log "Status check completed successfully"
    ;;
    
  "custom")
    if [ -z "$TASK_SCRIPT" ]; then
      handle_error "TASK_SCRIPT environment variable is required for custom task"
    fi
    
    log "Executing custom script"
    
    # Create temporary script file
    SCRIPT_FILE=$(mktemp)
    echo "$TASK_SCRIPT" > "$SCRIPT_FILE"
    chmod +x "$SCRIPT_FILE"
    
    # Execute script
    log "Running script"
    "$SCRIPT_FILE" || handle_error "Script execution failed"
    
    # Clean up
    rm -f "$SCRIPT_FILE"
    
    log "Custom script executed successfully"
    ;;
    
  *)
    handle_error "Unknown task type: $TASK_TYPE"
    ;;
esac

log "Task $TASK_ID completed successfully"
exit 0