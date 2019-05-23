# Multicluster-Service-Account

Multicluster-service-account makes it easy for pods in a cluster to call the Kubernetes APIs of other clusters. It imports remote service account tokens into local secrets, as [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) files, and automounts them inside annotated pods.

Multicluster-service-account can be used to run any Kubernetes client from another cluster. It can also be used to build operators that control Kubernetes resources across multiple clusters, e.g., with [multicluster-controller](https://github.com/admiraltyio/multicluster-controller).

Why? Check out [Admiralty's blog post introducing multicluster-service-account](https://admiralty.io/blog/introducing-multicluster-service-account).

## How it Works

Multicluster-service-account consists of:

1. A binary, `kubemcsa`, to bootstrap clusters, allowing them to import service account secrets from one another;
    - After [installing](#step-1-installation) multicluster-service-account in cluster1 (associated with the `cluster1` context in the installer's kubeconfig), allowing cluster1 to import service account secrets from cluster2 is as simple as running
        ```sh
        kubemcsa bootstrap cluster1 cluster2
        ```
1. a ServiceAccountImport custom resource definition (CRD) and controller to import remote service account secrets as kubeconfig files;
    - Here is a sample service account import object:
        ```yaml
        apiVersion: multicluster.admiralty.io/v1alpha1
        kind: ServiceAccountImport
        metadata:
          name: cluster2-default-pod-lister
        spec:
          clusterName: cluster2
          namespace: default # source and target namespaces can be different
          name: pod-lister
        ```
    - which would generate a secret like this:
        ```yaml
        apiVersion: v1
        kind: Secret
        metadata:
          name: cluster2-default-pod-lister-token-6456p
          ... # owner reference, etc.
        type: Opaque
        data:
          config: ... # serialized kubeconfig
        ```
1. a dynamic admission webhook to automount service account import secrets inside annotated pods, the same way regular service accounts are automounted inside Pods;
    - To automount the sample kubeconfig from above inside a pod, you would annotate the pod with `multicluster.admiralty.io/service-account-import.name=cluster2-default-pod-lister`. The pod and service account import must be in the same namespace. To mount multiple service account imports inside a single pod, append their names to the annotation, separated by commas.
    - The sample kubeconfig would be mounted at `/var/run/secrets/admiralty.io/serviceaccountimports/cluster2-default-pod-lister/config`. Most Kubernetes clients accept a `--kubeconfig` option or a `KUBECONFIG` environment variable, which you would set to that path.
1. **(optional)** Go helper functions (in the `pkg/config` package) to list and load mounted service account imports.

Note: Before v0.4.0, service account import secrets used a custom format (like regular service account secrets, with an additional "server" field to locate the remote cluster's Kubernetes API). Clients were required to use custom code, e.g., the provided Go helper functions, to load the secrets as REST configs. v0.4.0+ leverages the standard kubeconfig format to make it even easier to use multicluster-service-account, without any code change, with clients written in any language.

## Getting Started

We assume that you are a cluster admin on two clusters, associated with, e.g., the contexts "cluster1" and "cluster2" in your kubeconfig. We're going to install multicluster-service-account and run a multi-cluster client example in cluster1, listing pods in cluster2.

```bash
CLUSTER1=cluster1 # change me
CLUSTER2=cluster2 # change me
```

### Step 1: Installation

Install multicluster-service-account in cluster1:

```bash
RELEASE_URL=https://github.com/admiraltyio/multicluster-service-account/releases/download/v0.4.1
MANIFEST_URL=$RELEASE_URL/install.yaml
kubectl apply -f $MANIFEST_URL --context $CLUSTER1
```

Cluster1 is now able to import service accounts, but it hasn't been given permission to import them from cluster2 yet. This is a chicken-and-egg problem: cluster1 needs a token from cluster2, before it can import service accounts from it. To solve this problem, download the kubemcsa binary and run the bootstrap command:

```bash
OS=linux # or darwin (i.e., OS X) or windows
ARCH=amd64 # if you're on a different platform, you must know how to build from source
BINARY_URL="$RELEASE_URL/kubemcsa-$OS-$ARCH"
curl -Lo kubemcsa $BINARY_URL
chmod +x kubemcsa
sudo mv kubemcsa /usr/local/bin

kubemcsa bootstrap $CLUSTER1 $CLUSTER2
```

### Step 2: Example

The `multicluster-client` example includes:

- in cluster2:
  - a service account named `pod-lister` in the default namespace, bound to a role that can only list pods in its namespace;
  - a dummy NGINX deployment (to have pods to list);
- in cluster1:
  - a new label on the `default` namespace, `multicluster-service-account=enabled`, to instruct multicluster-service-account to automount service account import secrets inside annotated pods;
  - a service account import named `cluster2-default-pod-lister`, importing `pod-lister` from the default namespace of cluster2;
  - a `multicluster-client` job, whose pod is annotated to automount `cluster2-default-pod-lister`'s secret—it will list the pods in the default namespace of cluster2, and stop without restarting (we'll check the logs).

```bash
kubectl config use-context $CLUSTER2
kubectl create serviceaccount pod-lister
kubectl create role pod-lister --verb=list --resource=pods
kubectl create rolebinding pod-lister --role=pod-lister \
  --serviceaccount=default:pod-lister
kubectl run nginx --image nginx

kubectl config use-context $CLUSTER1
kubectl label namespace default multicluster-service-account=enabled
cat <<EOF | kubectl create -f -
apiVersion: multicluster.admiralty.io/v1alpha1
kind: ServiceAccountImport
metadata:
  name: $CLUSTER2-default-pod-lister
spec:
  clusterName: $CLUSTER2
  namespace: default
  name: pod-lister
---
apiVersion: batch/v1
kind: Job
metadata:
  name: multicluster-client
spec:
  template:
    metadata:
      annotations:
        multicluster.admiralty.io/service-account-import.name: $CLUSTER2-default-pod-lister
    spec:
      restartPolicy: Never
      containers:
      - name: multicluster-client
        image: quay.io/admiralty/multicluster-service-account-example-multicluster-client:latest
EOF
```

In cluster1, check that:

1. The service account import controller created a secret for the `cluster2-default-pod-lister` service account import, containing a kubeconfig file populated with the token and namespace of the remote service account, and the URL and root certificate of the remote Kubernetes API:
    ```bash
    kubectl get secret -l multicluster.admiralty.io/service-account-import.name=$CLUSTER2-default-pod-lister -o jsonpath={.items[0].data.config} | base64 -D
    # the data is base64-encoded
    ```
1. The service account import secret was mounted inside the `multicluster-client` pod by the service account import admission controller:
    ```bash
    kubectl get pod -l job-name=multicluster-client -o yaml
    # look at volumes and volume mounts
    ```
1. The `multicluster-client` pod was able to list pods in the default namespace of cluster2:
    ```bash
    kubectl logs job/multicluster-client
    ```

## Service Account Imports

Service account imports tell the service account import controller to maintain a secret in the same namespace, containing the remote service account's namespace and token, as well as the URL and root certificate of the remote Kubernetes API, which are all necessary data to configure a Kubernetes client. The secret is formatted as a standard [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file, which most Kubernetes clients can understand. If a pod needs to call several clusters, it will use several service account imports, e.g.:

```yaml
apiVersion: multicluster.admiralty.io/v1alpha1
kind: ServiceAccountImport
metadata:
  name: cluster2-default-pod-lister
spec:
  clusterName: cluster2
  namespace: default
  name: pod-lister
---
apiVersion: multicluster.admiralty.io/v1alpha1
kind: ServiceAccountImport
metadata:
  name: cluster3-default-pod-lister
spec:
  clusterName: cluster3
  namespace: default
  name: pod-lister
```

## Annotations

In namespaces labeled with `multicluster-service-account=enabled`, the `multicluster.admiralty.io/service-account-import.name` annotation on a pod (or pod template) tells the service account import admission controller to automount the corresponding secrets inside it. If a pod needs several service account imports, separate their names with commas, e.g.:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multicluster-client
  annotations:
    multicluster.admiralty.io/service-account-import.name: cluster2-default-pod-lister,cluster3-default-pod-lister
spec:
  # ...
```

Note: just like with local service accounts, there is a race condition if a service account import and a pod requesting it are created at the same time: the service account import admission controller will likely reject the pod because the secret to automount won't be ready. Luckily, if the pod is controlled by another object, such as a deployment, job, etc., pod creation will be retried.

## (Optional) Client Configuration

Multicluster-service-account includes a Go library (cf. [`pkg/config`](pkg/config)) to facilitate the creation of [client-go `rest.Config`](https://godoc.org/k8s.io/client-go/rest#Config) instances from service account imports. From there, you can create [`kubernetes.Clientset`](https://godoc.org/k8s.io/client-go/kubernetes#NewForConfig) instances as usual. The namespaces of the remote service accounts are also provided:

```go
cfg, ns, err := NamedServiceAccountImportConfigAndNamespace("cluster2-default-pod-lister")
// ...
clientset, err := kubernetes.NewForConfig(cfg)
// ...
pods, err := clientset.CoreV1().Pods(ns).List(metav1.ListOptions{})
```

Usually, however, you don't want to hardcode the name of the mounted service account import. If you only expect one, you can get a Config for it and its remote namespace like this:

```go
cfg, ns, err := ServiceAccountImportConfigAndNamespace()
```

If several service account imports are mounted, you can get Configs and namespaces for all of them by name as a `map[string]ConfigAndNamespaceTuple`:

```go
all, err := AllServiceAccountImportConfigsAndNamespaces()
// ...
for name, cfgAndNs := range all {
  cfg := cfgAndNs.Config
  ns := cfgAndNs.Namespace
  // ...
}
```

### Generic Client Configuration

The true power of multicluster-service-account's `config` package is in its generic functions, that can fall back to kubeconfig contexts or regular service accounts when no service account import is mounted:

```go
cfg, ns, err := ConfigAndNamespace()
```

```go
all, err := AllNamedConfigsAndNamespaces()
```

The service account import controller uses `AllNamedConfigsAndNamespaces()` internally. The [generic client example](examples/generic-client/) uses `ConfigAndNamespace()`.

## API Reference

For more details on the `config` package, or to better understand how the service account import controller and admission control work, please refer to the API documentation:

https://godoc.org/admiralty.io/multicluster-service-account/

or

```bash
go get admiralty.io/multicluster-service-account
godoc -http=:6060
```

then http://localhost:6060/pkg/admiralty.io/multicluster-service-account/
