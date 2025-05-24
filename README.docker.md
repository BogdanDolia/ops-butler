# Docker Images for Ops Butler

This directory contains Dockerfiles and scripts to build and push Docker images for the Ops Butler project.

## Components

The Ops Butler project consists of the following components:

1. **Agent**: Kubernetes agent that runs in the cluster
2. **API**: API server that provides HTTP and gRPC endpoints
3. **Scheduler**: Scheduler that runs periodic tasks
4. **Web**: Web UI for the Ops Butler portal

## Building and Pushing Images

To build and push all images to the registry, run the following command:

```bash
./build-push.sh [version]
```

Where `[version]` is an optional parameter that specifies the version tag for the images. If not provided, the images will be tagged as `latest`.

For example:

```bash
# Build and push images with the 'latest' tag
./build-push.sh

# Build and push images with a specific version tag
./build-push.sh v1.0.0
```

## Image Names

The images will be pushed to the registry with the following names:

- `crcoerph/ops-butler:agent-[version]`
- `crcoerph/ops-butler:api-[version]`
- `crcoerph/ops-butler:scheduler-[version]`
- `crcoerph/ops-butler:web-[version]`

## Deployment

The deployment YAML files in the `deploy/local` directory have been updated to use these image names with the `latest` tag.

To deploy the Ops Butler components to a Kubernetes cluster, run:

```bash
kubectl apply -f deploy/local/
```

## Building Individual Images

If you want to build and push individual images, you can use the following commands:

```bash
# Build and push the agent image
docker build -t crcoerph/ops-butler:agent-latest -f Dockerfile.agent .
docker push crcoerph/ops-butler:agent-latest

# Build and push the API image
docker build -t crcoerph/ops-butler:api-latest -f Dockerfile.api .
docker push crcoerph/ops-butler:api-latest

# Build and push the scheduler image
docker build -t crcoerph/ops-butler:scheduler-latest -f Dockerfile.scheduler .
docker push crcoerph/ops-butler:scheduler-latest

# Build and push the web image
docker build -t crcoerph/ops-butler:web-latest -f Dockerfile.web .
docker push crcoerph/ops-butler:web-latest
```