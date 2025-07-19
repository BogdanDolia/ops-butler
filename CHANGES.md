# Changes

## 2025-07-19: Fixed RBAC permissions for ops-butler-job service account

### Issue
The ops-butler-job service account was missing necessary permissions to list pods and nodes at the cluster scope and access the metrics API, resulting in the following errors:

```
Error from server (Forbidden): pods is forbidden: User "system:serviceaccount:ops-butler:ops-butler-job" cannot list resource "pods" in API group "" at the cluster scope
Error from server (Forbidden): nodes is forbidden: User "system:serviceaccount:ops-butler:ops-butler-job" cannot list resource "nodes" in API group "" at the cluster scope
[2025-07-19T18:23:20+00:00] Failed to get pod information
[2025-07-19T18:23:20+00:00] Resource usage:
error: Metrics API not available
```

### Changes Made
1. Added a ClusterRole named `ops-butler-job` with the following permissions:
   - Permission to list pods at the cluster scope
   - Permission to get and list nodes at the cluster scope
   - Permission to get and list node metrics

2. Added a ClusterRoleBinding named `ops-butler-job` that binds the ClusterRole to the `ops-butler-job` service account.

These changes allow the jobs created by ops-butler to:
- List pods across all namespaces (for the status check command)
- List nodes in the cluster (for the status check command)
- Access node metrics (for resource usage reporting)

### Files Modified
- `/deploy/k8s/core.yaml`
- `/deploy/k8s/core.yaml.example`

### How to Apply
Apply the updated configuration to your Kubernetes cluster:

```bash
kubectl apply -f deploy/k8s/core.yaml
```