# Unimplemented Features in Ops Butler

Based on a thorough examination of the codebase, the following features mentioned in the README are not yet fully implemented:

## 1. Web Portal
- **Authentication and Authorization**
  - Secure login (GitHub OAuth or OIDC) is not implemented
  - No authentication logic in web/src/pages/_app.js or any other page components

- **RBAC (Role-Based Access Control)**
  - No implementation of Viewer, Operator, Admin roles
  - No permission checks in the web UI or API endpoints

- **Real-time Log Streaming**
  - WebSocket handler is not implemented (internal/api/server.go:587-590)
  - No WebSocket client implementation in the web UI

## 2. API Server
- **Template Management Endpoints**
  - handleListTemplates (internal/api/server.go:324-327)
  - handleGetTemplate (internal/api/server.go:329-332)
  - handleCreateTemplate (internal/api/server.go:334-337)
  - handleUpdateTemplate (internal/api/server.go:339-342)
  - handleDeleteTemplate (internal/api/server.go:344-347)

- **Task Management Endpoints**
  - handleUpdateTask (internal/api/server.go:389-392)
  - handleDeleteTask (internal/api/server.go:394-397)

- **Log Retrieval**
  - Logic to get logs from pods (internal/api/server.go:444, 530)

## 3. ChatOps Integration
- **Google Chat Features**
  - Missing ScheduleMessage functionality (available in Slack but not in Google Chat)
  - Missing SendEphemeralMessage functionality (available in Slack but not in Google Chat)

- **User Identity and Permissions**
  - No clear implementation of user identity validation in either gateway

- **Notification System**
  - ChatOps notification with a button to check logs (internal/api/server.go:464)

## 4. Cluster Agent
- **Log Streaming**
  - Streaming output back to the server (internal/agent/agent.go:412)

## 5. Scheduler
- **ChatOps Integration**
  - Sending via ChatOps service (internal/scheduler/scheduler.go:218)

## 6. Database
- **Reminder Repository**
  - Dedicated reminder repository (internal/api/server.go:460)

## Summary
The platform has a solid foundation with many components partially implemented, but several key features mentioned in the README are still missing or incomplete. The most significant gaps are in authentication/authorization, real-time log streaming via WebSockets, and some aspects of the ChatOps integration, particularly for Google Chat.