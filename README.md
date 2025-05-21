# K8s Ops Portal

A Kubernetes-native self-service operations portal with multi-cluster support and deep ChatOps integration.

## Overview

K8s Ops Portal is an open-source platform that enables DevOps teams to create, manage, and execute operational tasks across multiple Kubernetes clusters. It provides a web interface, ChatOps integration, and a robust task scheduling system.

## Features

### Web Portal
- Secure login (GitHub OAuth or OIDC)
- CRUD for Task Templates: each template wraps a script (bash, kubectl, helm, terraform, etc.) plus a JSON/YAML schema of parameters
- RBAC roles: Viewer, Operator, Admin
- Real-time task execution log streaming (WebSocket)

### Cluster Agents
- Outbound gRPC/WebSocket tunnel to Portal API (no inbound ports)
- Executes tasks in its own cluster namespace and streams logs/chunks
- Supports label-based targeting (e.g. env=prod, region=eu)

### Task Manager / Reminders
- Each TaskInstance may have due_at (ISO-8601)
- Scheduler component creates a Reminder that, at the due time, posts an interactive message into Slack or Google Chat
- Buttons: "Run now", "Snooze 2 h", "Cancel"
- On "Run now" the task is dispatched to the appropriate agent; first 40 log lines are returned inline, full log as file

### ChatOps Gateways
- Slack: Block Kit interactive messages, scheduled reminders via chat.scheduleMessage, file uploads via files.upload
- Google Chat: v2 Cards with RunFunction actions
- Both gateways validate user identity and permissions

## Architecture

The system consists of the following components:

1. **API Server**: Handles HTTP requests, WebSocket connections, and gRPC calls
2. **Database**: Stores templates, tasks, reminders, and execution logs
3. **Scheduler**: Manages task scheduling and reminders
4. **Cluster Agents**: Run on each Kubernetes cluster to execute tasks
5. **ChatOps Gateways**: Integrate with Slack and Google Chat
6. **Web UI**: Provides a user-friendly interface for managing tasks

## Getting Started

### Prerequisites

- Kubernetes cluster (1.19+)
- Helm (3.0+)
- kubectl

### Quick Start

```bash
# Clone the repository
git clone https://github.com/BogdanDolia/ops-butler.git
cd ops-butler

# Start a local development environment
./scripts/quickstart-kind.sh
```

## Deployment

### Using Helm

```bash
helm repo add ops-butler https://BogdanDolia.github.io/ops-butler/charts
helm install ops-portal ops-butler/ops-portal
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.