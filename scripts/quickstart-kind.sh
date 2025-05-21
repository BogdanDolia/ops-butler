#!/bin/bash
set -e

# K8s Ops Portal - Local Development Environment Setup Script
# This script sets up a local Kubernetes development environment using kind

echo "🚀 Setting up K8s Ops Portal development environment..."

# Check prerequisites
echo "📋 Checking prerequisites..."

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "❌ kind is not installed. Please install kind first:"
    echo "   https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first:"
    echo "   https://kubernetes.io/docs/tasks/tools/install-kubectl/"
    exit 1
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

echo "✅ All prerequisites satisfied!"

# Create kind cluster
echo "🔄 Creating kind cluster 'ops-portal'..."

# Check if cluster already exists
if kind get clusters | grep -q "ops-portal"; then
    echo "⚠️  Kind cluster 'ops-portal' already exists. Skipping creation."
else
    # Create a kind cluster with port mappings for the portal
    cat <<EOF | kind create cluster --name ops-portal --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 8080
    protocol: TCP
  - containerPort: 30443
    hostPort: 8443
    protocol: TCP
EOF
    echo "✅ Kind cluster 'ops-portal' created successfully!"
fi

# Set kubectl context to the new cluster
kubectl cluster-info --context kind-ops-portal

# Deploy necessary components
echo "🔄 Deploying K8s Ops Portal components..."

# Create namespace
kubectl create namespace ops-portal --dry-run=client -o yaml | kubectl apply -f -

# Build and load Docker images
# This section is commented out because the images need to be built manually
# See the note below for instructions

# Apply Kubernetes manifests
echo "🔄 Applying Kubernetes manifests..."
kubectl apply -k deploy/local/

echo "⚠️  Note: This script assumes you have already built the Docker images."
echo "   If you haven't built the images yet, you can do so with:"
echo "   docker build -t ops-portal-api:dev ./api"
echo "   docker build -t ops-portal-web:dev ./web"
echo "   docker build -t ops-portal-scheduler:dev ./cmd/scheduler"
echo "   docker build -t ops-portal-agent:dev ./cmd/agent"
echo "   kind load docker-image ops-portal-api:dev --name ops-portal"
echo "   kind load docker-image ops-portal-web:dev --name ops-portal"
echo "   kind load docker-image ops-portal-scheduler:dev --name ops-portal"
echo "   kind load docker-image ops-portal-agent:dev --name ops-portal"

echo "🔄 Waiting for deployments to be ready..."
kubectl -n ops-portal wait --for=condition=available --timeout=300s deployment --all

echo "✅ K8s Ops Portal development environment is ready!"
echo ""
echo "📊 Access the portal at: http://localhost:8080"
echo "🔑 Default credentials: admin / admin123"
echo ""
echo "📝 Useful commands:"
echo "   - View all resources: kubectl -n ops-portal get all"
echo "   - View logs: kubectl -n ops-portal logs -l app=ops-portal-api"
echo "   - Delete cluster: kind delete cluster --name ops-portal"
echo ""
echo "Happy developing! 🎉"
