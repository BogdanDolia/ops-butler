# ChatOps Testing Guide

This guide explains how to test the ChatOps functionality in Ops Butler.

## Quick Start (Demo Mode)

For quick testing without a real Slack token:

```bash
# Set environment variables
export SLACK_ENABLED=true
export SLACK_DEMO_MODE=true
export LOG_LEVEL=debug

# Start the API server
go run cmd/api/main.go

# In another terminal, test the endpoints
./scripts/test-chatops.sh
```

## Test Endpoints

### Simple Message Test
```bash
curl -X POST http://localhost:8080/api/v1/chatops/test/slack \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "#general",
    "message": "Hello from Ops Butler! 🚀"
  }'
```

### Reminder Message with Buttons
```bash
curl -X POST http://localhost:8080/api/v1/chatops/test/slack \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "#general",
    "message": "⏰ Task reminder: Please check the production logs",
    "task_id": 42
  }'
```

## Real Slack Integration

To test with real Slack:

1. **Create a Slack App** at https://api.slack.com/apps
2. **Set Bot Token Scopes**:
   - `chat:write`
   - `chat:write.public`
   - `files:write`
   - `channels:read`
   - `chat:write.customize`

3. **Configure Interactive Components**:
   - Request URL: `https://your-domain.com/api/v1/chatops/slack/webhook`

4. **Set Environment Variables**:
   ```bash
   export SLACK_ENABLED=true
   export SLACK_TOKEN=xoxb-your-slack-bot-token
   export SLACK_SIGNING_SECRET=your-slack-signing-secret
   export SLACK_DEMO_MODE=false
   ```

5. **Test with real Slack**:
   ```bash
   # The same test endpoints will now send real messages to Slack
   curl -X POST http://localhost:8080/api/v1/chatops/test/slack \
     -H "Content-Type: application/json" \
     -d '{
       "channel": "#general",
       "message": "Hello from real Slack integration! 🚀"
     }'
   ```

## Slash Commands

The system supports these Slack slash commands:

- `/ops-status` - Show system status
- `/ops-run <task-id>` - Execute a task
- `/ops-list` - List recent tasks

Configure them in your Slack app with:
- Request URL: `https://your-domain.com/api/v1/chatops/slack/slash`

## Interactive Buttons

When you send a reminder message with a task ID, it creates interactive buttons:

- 🚀 **Run Now** - Execute the task immediately
- ⏰ **Snooze 2h** - Delay the task for 2 hours
- ❌ **Cancel** - Cancel the task
- 📋 **View Full Logs** - Download full execution logs

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SLACK_ENABLED` | Enable Slack integration | `false` | Yes |
| `SLACK_TOKEN` | Slack bot token | `""` | Yes (unless demo mode) |
| `SLACK_DEMO_MODE` | Enable demo mode | `false` | No |
| `SLACK_SIGNING_SECRET` | Slack signing secret | `""` | No (for webhook verification) |
| `SLACK_DEFAULT_CHANNEL` | Default channel | `general` | No |
| `GOOGLE_CHAT_ENABLED` | Enable Google Chat | `false` | No |

## Demo Mode Features

In demo mode (`SLACK_DEMO_MODE=true`):
- ✅ No real Slack token required
- ✅ Messages logged to console
- ✅ All endpoints work
- ✅ Interactive buttons simulated
- ✅ Perfect for development/testing

## Troubleshooting

### "ChatOps service not available"

1. **Check if Slack is enabled**:
   ```bash
   curl http://localhost:8080/health
   ```

2. **Enable Slack**:
   ```bash
   export SLACK_ENABLED=true
   ```

3. **Check logs** for detailed error messages:
   ```bash
   export LOG_LEVEL=debug
   ```

### "Failed to send message"

1. **In demo mode**: Check server logs for simulated messages
2. **In real mode**: Verify your Slack token and bot permissions

### Webhook not working

1. **Check ngrok/tunnel**: Ensure your webhook URL is accessible
2. **Verify signing secret**: Set `SLACK_SIGNING_SECRET` correctly
3. **Check logs**: Look for webhook verification errors

## Examples

See `cmd/test-slack/main.go` for a complete Go example of testing the ChatOps endpoints.

Run the test script:
```bash
go run cmd/test-slack/main.go
```

Or use the shell script:
```bash
./scripts/test-chatops.sh
``` 