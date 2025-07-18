#!/bin/bash
set -e

# Registry and image prefix
REGISTRY="crcoerph/ops-butler"

# Version
VERSION=${1:-latest}

# Components
COMPONENTS=("agent" "api" "scheduler" "web")

# Build and push images
for component in "${COMPONENTS[@]}"; do
  echo "Building ${component} image..."

  # Special handling for web component to pass API_URL
  if [ "$component" == "web" ]; then
    # Use API_URL from environment or default to http://ops-butler-api:8080 for production
    API_URL=${API_URL:-http://ops-butler-api:8080}
    echo "Building web with API_URL=${API_URL}"
    docker build --build-arg API_URL=${API_URL} -t ${REGISTRY}:${component}-${VERSION} -f Dockerfile.${component} .
  else
    docker build -t ${REGISTRY}:${component}-${VERSION} -f Dockerfile.${component} .
  fi

  echo "Pushing ${component} image..."
  docker push ${REGISTRY}:${component}-${VERSION}

  # Tag as latest if version is not 'latest'
  if [ "$VERSION" != "latest" ]; then
    docker tag ${REGISTRY}:${component}-${VERSION} ${REGISTRY}:${component}-latest
    docker push ${REGISTRY}:${component}-latest
  fi

  echo "${component} image built and pushed successfully."
done

echo "All images built and pushed successfully."
