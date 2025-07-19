### How to Fix "Metrics API not available" Error on Kind Cluster

The error "Metrics API not available" occurs because the Kubernetes Metrics API is not installed by default in Kind (Kubernetes in Docker) clusters. Based on the project files, I can see that Ops-Butler uses the Metrics API to report resource usage, as indicated in the RBAC permissions in `core.yaml` and the error mentioned in `CHANGES.md`.

### Solution

To fix this issue, you need to install the Kubernetes Metrics Server in your Kind cluster. Here's how:

#### 1. Install Metrics Server with Kind-specific configuration

```bash
# Download the metrics server manifest
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

#### 2. Patch the Metrics Server deployment to work with Kind

Kind uses self-signed certificates, so you need to configure the metrics-server to skip TLS verification:

```bash
kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'
```

#### 3. Verify the Metrics Server is running

```bash
kubectl get pods -n kube-system | grep metrics-server
```

#### 4. Test that the Metrics API is working

```bash
kubectl top nodes
kubectl top pods
```

### Alternative: Using a Kind configuration file

If you're creating a new Kind cluster, you can also enable the metrics server during cluster creation:

1. Create a Kind config file (e.g., `kind-config.yaml`):

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    metadata:
      name: config
    apiServer:
      extraArgs:
        enable-aggregator-routing: "true"
```

2. Create the cluster with this config:

```bash
kind create cluster --config kind-config.yaml
```

3. Then install the metrics server as described above.

### Why this happens

The Metrics API (`metrics.k8s.io`) is provided by the Kubernetes Metrics Server, which is not part of the core Kubernetes components and is not installed by default in Kind clusters. The Metrics Server collects resource metrics from Kubelets and exposes them through the Kubernetes API server for use by tools like `kubectl top` and the Horizontal Pod Autoscaler.

In the Ops-Butler project, the jobs created use this API to report on cluster resource usage, as shown in the RBAC permissions in the `core.yaml` file.