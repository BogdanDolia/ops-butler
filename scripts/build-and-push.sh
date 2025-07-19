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

# Registry prefix
REGISTRY="crcoerph/ops-butler"

# Default tag
TAG=${TAG:-latest}

# Components to build
COMPONENTS=("core" "job" "webui")

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
  handle_error "Docker is not installed or not in PATH"
fi

# Check if user is logged in to Docker
if ! docker info &> /dev/null; then
  log "You may need to log in to Docker registry"
  log "Run: docker login"
fi

# Function to build and push a component
build_and_push() {
  local component=$1
  local dockerfile="Dockerfile.${component}"
  local image_name="${REGISTRY}-${component}:${TAG}"
  
  log "Building ${component} component..."
  
  # Check if Dockerfile exists
  if [ ! -f "${dockerfile}" ]; then
    handle_error "Dockerfile ${dockerfile} not found"
  fi
  
  # Build the image
  log "Running: docker build -t ${image_name} -f ${dockerfile} ."
  if ! docker build -t "${image_name}" -f "${dockerfile}" .; then
    handle_error "Failed to build ${component} component"
  fi
  
  log "${component} component built successfully"
  
  # Push the image if requested
  if [ "$PUSH" = "true" ]; then
    log "Pushing ${image_name} to registry..."
    if ! docker push "${image_name}"; then
      handle_error "Failed to push ${component} component to registry"
    fi
    log "${component} component pushed successfully"
  fi
}

# Parse command line arguments
PUSH="false"
while [[ $# -gt 0 ]]; do
  case $1 in
    --push)
      PUSH="true"
      shift
      ;;
    --tag)
      TAG="$2"
      shift 2
      ;;
    --component)
      if [[ " ${COMPONENTS[*]} " =~ " $2 " ]]; then
        COMPONENTS=("$2")
      else
        handle_error "Invalid component: $2. Valid components are: core, job, webui"
      fi
      shift 2
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --push              Push images to registry after building"
      echo "  --tag TAG           Use specified tag instead of 'latest'"
      echo "  --component COMP    Build only specified component (core, job, or webui)"
      echo "  --help              Show this help message"
      exit 0
      ;;
    *)
      handle_error "Unknown option: $1"
      ;;
  esac
done

# Main execution
log "Starting build process for ${REGISTRY} with tag ${TAG}"

for component in "${COMPONENTS[@]}"; do
  build_and_push "${component}"
done

if [ "$PUSH" = "true" ]; then
  log "All components built and pushed successfully"
else
  log "All components built successfully"
  log "To push images to registry, run: $0 --push"
fi

exit 0