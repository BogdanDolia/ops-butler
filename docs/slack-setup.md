# Slack Integration Setup Guide

This guide provides detailed instructions for setting up the Slack integration for Ops-Butler.

## Prerequisites

- A Slack workspace where you have permissions to create apps
- Admin access to your Kubernetes cluster

## Step 1: Create a Slack App

1. Go to [https://api.slack.com/apps](https://api.slack.com/apps)
2. Click "Create New App"
3. Choose "From scratch"
4. Enter "Ops-Butler" as the app name
5. Select your workspace
6. Click "Create App"

## Step 2: Configure Bot Token Scopes

1. In the left sidebar, click on "OAuth & Permissions"
2. Scroll down to "Scopes" section
3. Under "Bot Token Scopes", add the following scopes:
   - `chat:write` - Send messages as the app
   - `commands` - Add slash commands to the app
   - `channels:read` - View basic information about public channels
   - `groups:read` - View basic information about private channels
   - `im:read` - View basic information about direct messages
   - `mpim:read` - View basic information about group direct messages

## Step 3: Install App to Workspace

1. Scroll back to the top of the "OAuth & Permissions" page
2. Click "Install to Workspace"
3. Review the permissions and click "Allow"
4. Note the "Bot User OAuth Token" that appears - you'll need this later

## Step 4: Create a Slash Command

1. In the left sidebar, click on "Slash Commands"
2. Click "Create New Command"
3. Fill in the following details:
   - Command: `/ops`
   - Request URL: `https://your-ingress-domain/slack/command` (you'll update this after deployment)
   - Short Description: "Ops-Butler ChatOps commands"
   - Usage Hint: "[help|status|collect-logs|list]"
4. Click "Save"

## Step 5: Enable Event Subscriptions

1. In the left sidebar, click on "Event Subscriptions"
2. Toggle "Enable Events" to On
3. Set the Request URL to `https://your-ingress-domain/slack/event` (you'll update this after deployment)
4. Under "Subscribe to bot events", add the following events:
   - `message.channels` - A message was posted to a channel
   - `message.groups` - A message was posted to a private channel
5. Click "Save Changes"

## Step 6: Get App Credentials

You'll need the following credentials to configure Ops-Butler:

1. **Bot Token**: From the "OAuth & Permissions" page, copy the "Bot User OAuth Token" (starts with `xoxb-`)
2. **Signing Secret**: From the "Basic Information" page, under "App Credentials", click "Show" next to "Signing Secret"
3. **App Token** (optional): From the "Basic Information" page, under "App-Level Tokens", click "Generate Token and Scopes", name it "ops-butler-app-token", add the `connections:write` scope, and click "Generate". Copy the token (starts with `xapp-`)

## Step 7: Configure Ops-Butler with Slack Credentials

1. Base64 encode each of the credentials:

```bash
echo -n "xoxb-your-bot-token" | base64
echo -n "your-signing-secret" | base64
echo -n "xapp-your-app-token" | base64
```

2. Edit the `deploy/k8s/core.yaml` file and update the Secret section:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ops-butler-core-secrets
  namespace: ops-butler
type: Opaque
data:
  SLACK_BOT_TOKEN: "base64-encoded-bot-token"
  SLACK_SIGNING_KEY: "base64-encoded-signing-secret"
  SLACK_APP_TOKEN: "base64-encoded-app-token"
```

3. Apply the changes:

```bash
kubectl apply -f deploy/k8s/core.yaml
```

## Step 8: Update Slack App with Actual URLs

After deploying Ops-Butler and setting up your ingress, update your Slack App with the actual URLs:

1. Go back to [https://api.slack.com/apps](https://api.slack.com/apps) and select your Ops-Butler app
2. Update the Slash Command Request URL:
   - Go to "Slash Commands"
   - Click on the `/ops` command
   - Update the Request URL to your actual URL
   - Click "Save"
3. Update the Event Subscriptions Request URL:
   - Go to "Event Subscriptions"
   - Update the Request URL
   - Click "Save Changes"

## Step 9: Test the Integration

1. In your Slack workspace, type `/ops help`
2. You should receive a response from Ops-Butler with available commands
3. Try other commands like `/ops status` to verify the integration is working

## Troubleshooting

### Command Not Working

If the slash command doesn't work:

1. Check the logs of the Core pod:
```bash
kubectl logs -n ops-butler statefulset/ops-butler-core
```

2. Verify the Slack credentials are correct
3. Ensure your ingress is properly configured and accessible from the internet
4. Check that the Request URLs in your Slack App configuration are correct

### Verification Errors

If you see verification errors in the logs:

1. Double-check your Signing Secret
2. Ensure the Request URLs are using HTTPS if required
3. Verify that your ingress is properly forwarding requests to the Core pod

### Permission Errors

If you see permission errors in the logs:

1. Verify that you've added all the required scopes to your Slack App
2. Reinstall the app to your workspace to apply the updated scopes

## Advanced Configuration

### Custom Slash Command

You can change the slash command from `/ops` to something else:

1. Create a new Slash Command in your Slack App
2. Update the Core pod to recognize the new command by editing the ConfigMap

### Multiple Workspaces

To use Ops-Butler with multiple Slack workspaces:

1. Create a Slack App in each workspace
2. Configure multiple sets of credentials in the Core pod
3. Update the Core code to handle multiple workspaces