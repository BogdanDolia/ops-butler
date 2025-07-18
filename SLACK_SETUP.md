# 🚀 Your Slack Integration Setup

Perfect! You have real Slack tokens configured. Here's how to test and deploy your integration.

## 📋 Your Configuration

I've updated your configuration files with your real Slack tokens:

```yaml
# deploy/local/chatops.yaml
SLACK_ENABLED: "true"
SLACK_TOKEN: "xoxb-8948822092324-9217588031638-SA83KBY2H0Zgie0ZKQu43cEv"
SLACK_SIGNING_SECRET: "aea37904257a9f07f1df03ee860f9594"
SLACK_DEFAULT_CHANNEL: "#ops-alerts"
SLACK_DEMO_MODE: "false"  # Real Slack integration
```

## 🧪 Quick Test (Local)

Test your integration locally:

```bash
# 1. Make sure you have a database running (or use demo mode)
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=k8s_ops_portal

# 2. Start the API server
go run cmd/api/main.go

# 3. In another terminal, run the quick test
./scripts/quick-slack-test.sh
```

## 🎯 What to Expect

After running the test, check your **#ops-alerts** channel in Slack:

1. **Simple message**: "🚀 Quick test from Ops Butler!"
2. **Interactive message**: With buttons for "Run Now", "Snooze", "Cancel"

## 🚀 Deploy to Kubernetes

```bash
# 1. Create namespace
kubectl create namespace ops-butler

# 2. Deploy configurations
kubectl apply -f deploy/local/chatops.yaml
kubectl apply -f deploy/local/api.yaml
kubectl apply -f deploy/local/db.yaml

# 3. Check deployment
kubectl get pods -n ops-butler
kubectl logs -f deployment/ops-butler-api -n ops-butler
```

## 🔧 Configure Slack App

To enable interactive buttons and slash commands:

1. **Go to your Slack App settings**: https://api.slack.com/apps
2. **Find your app** (should be the one with your token)
3. **Configure Interactive Components**:
   - Go to "Interactivity & Shortcuts"
   - Enable Interactivity
   - Request URL: `https://your-domain.com/api/v1/chatops/slack/webhook`

4. **Configure Slash Commands**:
   - Create `/ops-status` → `https://your-domain.com/api/v1/chatops/slack/slash`
   - Create `/ops-run` → `https://your-domain.com/api/v1/chatops/slack/slash`
   - Create `/ops-list` → `https://your-domain.com/api/v1/chatops/slack/slash`

## 🌐 External Access (for Webhooks)

For Slack webhooks to work, you need external access:

### Option 1: ngrok (for testing)
```bash
# Install ngrok
brew install ngrok  # or download from ngrok.com

# Start tunnel
ngrok http 8080

# Use the https URL in your Slack app webhook settings
```

### Option 2: Kubernetes LoadBalancer
```bash
# Get external IP
kubectl get svc ops-butler-webhooks -n ops-butler
```

### Option 3: Ingress (production)
```bash
# Update your domain in deploy/local/chatops.yaml
# Then apply the ingress configuration
kubectl apply -f deploy/local/chatops.yaml
```

## 🧪 Full Test Suite

Once deployed, test everything:

```bash
# Test with your actual tokens
export SLACK_TOKEN="xoxb-8948822092324-9217588031638-SA83KBY2H0Zgie0ZKQu43cEv"
export SLACK_SIGNING_SECRET="aea37904257a9f07f1df03ee860f9594"
export API_URL="https://your-domain.com"  # or ngrok URL

./scripts/test-real-slack.sh
```

## 🎛️ Interactive Features

Your Slack integration now supports:

- ✅ **Send messages** to any channel
- ✅ **Interactive buttons** (Run Now, Snooze, Cancel)
- ✅ **Slash commands** (`/ops-status`, `/ops-run`, `/ops-list`)
- ✅ **File uploads** for logs
- ✅ **Task reminders** with due dates
- ✅ **User verification** for security

## 📊 Monitor Your Integration

```bash
# Check logs
kubectl logs -f deployment/ops-butler-api -n ops-butler

# Check health
curl https://your-domain.com/health

# Check ChatOps status
curl https://your-domain.com/api/v1/chatops/test/slack \
  -H "Content-Type: application/json" \
  -d '{"channel": "#ops-alerts", "message": "Health check"}'
```

## 🛠️ Troubleshooting

### Common Issues:

1. **"Token invalid"**: Verify your bot token is correct and app is installed
2. **"Channel not found"**: Make sure the bot is added to #ops-alerts channel
3. **"Interactive buttons not working"**: Check webhook URL in Slack app settings
4. **"Permission denied"**: Ensure bot has required OAuth scopes

### Debug Commands:

```bash
# Check environment variables
kubectl exec -it deployment/ops-butler-api -n ops-butler -- env | grep SLACK

# Check ConfigMap
kubectl get configmap chatops-config -n ops-butler -o yaml

# Test webhook endpoint
curl -X POST https://your-domain.com/api/v1/chatops/slack/webhook \
  -H "Content-Type: application/json" \
  -d '{"challenge": "test"}'
```

## 🎉 Next Steps

1. **Test the quick setup**: `./scripts/quick-slack-test.sh`
2. **Create tasks** that send notifications to Slack
3. **Set up webhooks** for interactive buttons
4. **Add more channels** for different alerts
5. **Create scheduled reminders** for routine tasks

## 📚 Additional Resources

- [Complete Slack Integration Guide](docs/slack-integration.md)
- [ChatOps Testing Guide](README.chatops.md)
- [Kubernetes Deployment Guide](docs/kubernetes-deployment.md)

---

**🚀 Ready to test?** Run `./scripts/quick-slack-test.sh` and check your Slack channel! 