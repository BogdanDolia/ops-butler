# Slack Integration Guide

This guide explains how to set up real Slack integration with your Ops Butler deployment.

## Prerequisites

- ✅ Slack workspace with admin permissions
- ✅ Kubernetes cluster
- ✅ Domain name with SSL/TLS certificate (for webhooks)
- ✅ kubectl configured

## Step 1: Create Slack App

1. Go to https://api.slack.com/apps
2. Click "Create New App" → "From scratch"
3. Choose app name: `Ops Butler`
4. Select your workspace

### Configure OAuth & Permissions

In your Slack app, go to **OAuth & Permissions** and add these scopes:

**Bot Token Scopes:**
- `chat:write` - Send messages
- `chat:write.public` - Send messages to public channels
- `files:write` - Upload files
- `channels:read` - Read channel information
- `chat:write.customize` - Customize messages
- `commands` - Use slash commands
- `im:read` - Read direct messages
- `mpim:read` - Read group direct messages

### Configure Interactive Components

1. Go to **Interactivity & Shortcuts**
2. Enable Interactivity
3. Set Request URL: `https://your-domain.com/api/v1/chatops/slack/webhook`
4. Save Changes

### Configure Slash Commands

Create these slash commands:

1. **Command:** `/ops-status`
   - **Request URL:** `https://your-domain.com/api/v1/chatops/slack/slash`
   - **Description:** Show Ops Butler system status
   - **Usage Hint:** No parameters needed

2. **Command:** `/ops-run`
   - **Request URL:** `https://your-domain.com/api/v1/chatops/slack/slash`
   - **Description:** Execute a task by ID
   - **Usage Hint:** `<task-id>`

3. **Command:** `/ops-list`
   - **Request URL:** `https://your-domain.com/api/v1/chatops/slack/slash`
   - **Description:** List recent tasks
   - **Usage Hint:** No parameters needed

### Install App to Workspace

1. Go to **Install App**
2. Click "Install to Workspace"
3. Authorize the app
4. Copy the **Bot User OAuth Token** (starts with `xoxb-`)

## Step 2: Configure Kubernetes Secrets

Update your `deploy/local/chatops.yaml` with your real tokens:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chatops-config
  namespace: ops-butler
data:
  SLACK_ENABLED: "true"
  SLACK_TOKEN: "xoxb-your-actual-slack-token-here"
  SLACK_SIGNING_SECRET: "your-signing-secret-here"
  SLACK_DEFAULT_CHANNEL: "#ops-alerts"
  SLACK_DEMO_MODE: "false"
  SLACK_APP_ID: "A08PHCZKLQH"  # Your App ID
  SLACK_BOT_USER_ID: "U08PHCZKLQH"  # Your Bot User ID
```

**To get your App ID and Bot User ID:**
1. Go to your Slack app **Basic Information**
2. Copy **App ID** 
3. Go to **OAuth & Permissions** 
4. Copy **Bot User ID**

## Step 3: Deploy to Kubernetes

```bash
# Create namespace
kubectl create namespace ops-butler

# Deploy ChatOps configuration
kubectl apply -f deploy/local/chatops.yaml

# Deploy API server with ChatOps integration
kubectl apply -f deploy/local/api.yaml

# Deploy other components
kubectl apply -f deploy/local/db.yaml
kubectl apply -f deploy/local/redis.yaml

# Check deployment
kubectl get pods -n ops-butler
```

## Step 4: Configure Ingress/Load Balancer

For webhooks to work, you need external access:

### Option A: LoadBalancer

```yaml
# Already configured in chatops.yaml
type: LoadBalancer
```

### Option B: Ingress with SSL

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ops-butler-webhook-ingress
  namespace: ops-butler
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - your-domain.com
      secretName: ops-butler-webhook-tls
  rules:
    - host: your-domain.com
      http:
        paths:
          - path: /api/v1/chatops
            pathType: Prefix
            backend:
              service:
                name: ops-butler-webhooks
                port:
                  number: 8080
```

## Step 5: Test Integration

### Local Testing

```bash
# Export your tokens
export SLACK_TOKEN="xoxb-your-token"
export SLACK_SIGNING_SECRET="your-signing-secret"
export SLACK_DEFAULT_CHANNEL="#ops-alerts"

# Test locally
./scripts/test-real-slack.sh
```

### Production Testing

```bash
# Test health endpoint
curl https://your-domain.com/health

# Test Slack message
curl -X POST https://your-domain.com/api/v1/chatops/test/slack \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "#ops-alerts",
    "message": "🚀 Production Slack integration test!"
  }'
```

## Step 6: Verify Setup

1. **Check Kubernetes logs:**
   ```bash
   kubectl logs -f deployment/ops-butler-api -n ops-butler
   ```

2. **Look for these log messages:**
   - `"Initializing ChatOps service"`
   - `"ChatOps service initialized successfully"`
   - `"Slack client initialized"`

3. **Test in Slack:**
   - Send a test message via API
   - Try slash commands: `/ops-status`
   - Click interactive buttons in reminder messages

## Troubleshooting

### Common Issues

1. **"ChatOps service not available"**
   - Check if `SLACK_ENABLED=true` in ConfigMap
   - Verify Slack token is set correctly
   - Check pod logs for initialization errors

2. **"Token invalid"**
   - Verify your bot token starts with `xoxb-`
   - Check if app is installed in workspace
   - Ensure bot has required permissions

3. **"Webhook not receiving events"**
   - Check if your domain is accessible from internet
   - Verify SSL certificate is valid
   - Test webhook URL manually

4. **"Interactive buttons not working"**
   - Verify Request URL in Slack app settings
   - Check signing secret is correct
   - Look for webhook verification errors in logs

### Debug Commands

```bash
# Check ConfigMap
kubectl get configmap chatops-config -n ops-butler -o yaml

# Check environment variables in pod
kubectl exec -it deployment/ops-butler-api -n ops-butler -- env | grep SLACK

# Check service endpoints
kubectl get svc -n ops-butler

# Check ingress
kubectl get ingress -n ops-butler
```

## Security Best Practices

1. **Use Kubernetes Secrets for tokens:**
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: slack-secrets
     namespace: ops-butler
   type: Opaque
   data:
     SLACK_TOKEN: <base64-encoded-token>
     SLACK_SIGNING_SECRET: <base64-encoded-secret>
   ```

2. **Restrict webhook access:**
   ```yaml
   # Add IP whitelist for Slack
   nginx.ingress.kubernetes.io/whitelist-source-range: "0.0.0.0/0"
   ```

3. **Enable webhook verification:**
   - Always set `SLACK_SIGNING_SECRET`
   - Check logs for verification errors

## Interactive Features

Once configured, your Slack integration supports:

- 📤 **Message sending** to any channel
- 🎛️ **Interactive buttons** for task management
- 📋 **File uploads** for logs
- ⚡ **Slash commands** for quick actions
- 🔔 **Scheduled reminders** for tasks
- 👥 **User identity** verification

## Next Steps

1. Configure Google Chat integration (optional)
2. Set up task templates
3. Create automated reminders
4. Add more slash commands
5. Customize message templates 