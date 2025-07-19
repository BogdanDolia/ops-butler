# Implementation Summary: Fix for Pod Access Permission Issue

## Issue Description
The ops-butler job was failing with the following error when trying to collect logs from a pod in the kube-system namespace:

```
[2025-07-19T18:48:32+00:00] Starting job for task 9 of type collect_logs
[2025-07-19T18:48:32+00:00] Task parameters: {"namespace":"kube-system","pod_name":"kube-proxy-sznzp"}
[2025-07-19T18:48:32+00:00] Set parameter: namespace=kube-system
[2025-07-19T18:48:32+00:00] Set parameter: pod_name=kube-proxy-sznzp
[2025-07-19T18:48:32+00:00] Collecting logs from pod kube-proxy-sznzp
[2025-07-19T18:48:32+00:00] ERROR: Pod kube-proxy-sznzp not found in namespace kube-system
```

## Root Cause
The issue was caused by insufficient RBAC permissions for the "ops-butler-job" ServiceAccount. The ClusterRole associated with this ServiceAccount only had permissions to:
- List pods (but not get specific pods)
- Get and list nodes
- Get and list metrics for nodes

It did not have permissions to:
- Get specific pods in other namespaces (like kube-system)
- Access pod logs (pods/log resource)

## Solution
The solution was to update the ClusterRole for the "ops-butler-job" ServiceAccount to include the necessary permissions:

1. Added "get" permission for pods (previously it only had "list")
2. Added permission to access pod logs (pods/log resource) with "get" verb

These changes allow the job to:
- Get specific pods in any namespace (including kube-system)
- List pods in any namespace
- Access pod logs in any namespace

## Changes Made
Updated the ClusterRole in both core.yaml and core.yaml.example:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ops-butler-job
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]  # Added "get" permission
- apiGroups: [""]
  resources: ["pods/log"]  # Added permission for pod logs
  verbs: ["get"]
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["nodes"]
  verbs: ["get", "list"]
```

## Deployment
To apply these changes to the Kubernetes cluster, run:

```bash
kubectl apply -f deploy/k8s/core.yaml
```

After applying the changes, the ops-butler job should be able to access pods and pod logs in the kube-system namespace, resolving the reported issue.