# Ops-Butler

Ops-Butler is a ChatOps tool for Kubernetes operations management, focused on simplicity and extensibility. It provides a Slack interface for common operations tasks and a web dashboard for monitoring and management.

## Architecture

Ops-Butler consists of two main components:

1. **Core Pod**: A StatefulSet that handles all core logic, including command processing, task orchestration, and Slack interactions. It uses SQLite as the default embedded database.

2. **Web UI Pod**: A Deployment that provides a web interface for viewing task statuses, logs, and basic configurations.

Tasks are executed as Kubernetes Jobs, which are short-lived, on-demand, and triggered from the Core pod or Web UI.

## Features

- **ChatOps via Slack**: Interact with your Kubernetes cluster using Slack slash commands
- **Task Execution**: Run operations tasks as Kubernetes Jobs
- **Web Dashboard**: View task status, logs, and configurations
- **Embedded Database**: Uses SQLite by default, with options for external databases
- **Extensibility**: Add custom tasks and scripts

## Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured to access your cluster
- Docker for building images
- Slack workspace with permissions to create a bot

## Setup

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/ops-butler.git
cd ops-butler
```

### 2. Configure Slack Integration

1. Create a Slack App at https://api.slack.com/apps
2. Enable the following features:
   - Slash Commands (create a command `/ops`)
   - Bot Token Scopes: `chat:write`, `commands`, `channels:read`
3. Install the app to your workspace
4. Note the Bot Token, App Token, and Signing Secret

### 3. Build Docker Images

#### Option 1: Using the helper script

The repository includes a helper script to build and push images to the Docker registry:

```bash
# Build all images
./scripts/build-and-push.sh

# Build and push all images to the registry
./scripts/build-and-push.sh --push

# Build a specific component
./scripts/build-and-push.sh --component core

# Build with a specific tag
./scripts/build-and-push.sh --tag v1.0.0

# Show help
./scripts/build-and-push.sh --help
```

The script will build images with the registry prefix `crcoerph/ops-butler`.

#### Option 2: Manual build

```bash
# Build Core image
docker build -t crcoerph/ops-butler-core:latest -f Dockerfile.core .

# Build Web UI image
docker build -t crcoerph/ops-butler-webui:latest -f Dockerfile.webui .

# Build Job image
docker build -t crcoerph/ops-butler-job:latest -f Dockerfile.job .

# Push images to registry
docker push crcoerph/ops-butler-core:latest
docker push crcoerph/ops-butler-webui:latest
docker push crcoerph/ops-butler-job:latest
```

### 4. Configure Kubernetes Deployment

1. Edit the Slack secrets in `deploy/k8s/core.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ops-butler-core-secrets
  namespace: ops-butler
type: Opaque
data:
  # These values should be base64 encoded
  SLACK_BOT_TOKEN: "base64-encoded-bot-token"
  SLACK_APP_TOKEN: "base64-encoded-app-token"
  SLACK_SIGNING_KEY: "base64-encoded-signing-key"
```

2. Customize other configuration values in the ConfigMaps if needed.

### 5. Deploy to Kubernetes

```bash
# Create namespace and resources
kubectl apply -f deploy/k8s/core.yaml
kubectl apply -f deploy/k8s/webui.yaml
```

### 6. Configure Slack to Send Commands to Ops-Butler

1. In your Slack App configuration, set the Request URL for your slash command to:
   `https://your-ingress-domain/slack/command`

2. Set the Event Subscriptions URL to:
   `https://your-ingress-domain/slack/event`

## Usage

### Slack Commands

- `/ops help` - Show help message
- `/ops status` - Show system status
- `/ops collect-logs [pod-name]` - Collect logs from a pod
- `/ops list` - List recent tasks

### Web UI

Access the Web UI at `https://your-ingress-domain/`

The dashboard provides:
- Task list and status
- Task logs
- Template management
- Basic configuration

## Database Configuration

By default, Ops-Butler uses SQLite as an embedded database. To use an external database:

1. Edit the ConfigMap in `deploy/k8s/core.yaml`:

```yaml
DB_TYPE: "postgres"  # or "mysql"
DB_HOST: "your-db-host"
DB_PORT: "5432"
DB_USER: "your-db-user"
DB_PASSWORD: "your-db-password"
DB_NAME: "ops_butler"
DB_SSL_MODE: "disable"  # for postgres
```

2. Apply the changes:

```bash
kubectl apply -f deploy/k8s/core.yaml
```

## Adding Custom Tasks

To add custom tasks:

1. Create a template in the Web UI with your script
2. Use the `/ops custom [template-name]` command in Slack

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n ops-butler
```

### View Pod Logs

```bash
kubectl logs -n ops-butler deployment/ops-butler-webui
kubectl logs -n ops-butler statefulset/ops-butler-core
```

### Check Persistent Volume

```bash
kubectl get pvc -n ops-butler
```

## License

[MIT License](LICENSE)