# Load Balancer Readvertiser

The Load Balancer Readvertiser is a tool that is used for [Shoot clusters](https://github.com/gardener/documentation/wiki/Architecture). The `kube-apiserver` of a Shoot cluster must be reachable by the `kubernetes` service in the `default` namespace (usually created with service ip `100.64.0.1`). In order to enable that, the apiserver must expose its public ip address. In the Shoot setup, the only way to reach it is via a public load balancer. The underlying IP address / Hostname can change at any time. The detection of those changes is exactly the purpose of the Readvertiser. It will watch for the LoadBalancer ingress changes (refresh Hostnames entries periodically) and update the `kubernetes` endpoint of the shoot-cluster with the correct IP(s) properly.

## Flags

```console
-hostname-refresh-period duration
    the period at which the Loadbalancer's hostnames are resynced (default 30s)
-kubeconfig string
    Paths to a seed's kubeconfig. Only required if out-of-cluster.
-master string
    The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
-metrics-addr string
    The address the metric endpoint binds to. (default ":8080")
-resync-period duration
    the period at which the controller sync with the cache will happen (default 30m0s)
-service-name string
    Name of the service of type Loadbalancer (default "kube-apiserver")
-service-namespace string
    Namespace of the service of type Loadbalancer
-shoot-kubeconfig string
    Paths to a shoot's kubeconfig. Required.
-endpoint-name string
    TEST ONLY - name of the endpoint to reconcile (default "kubernetes")
-endpoint-namespace string
    TEST ONLY - namespace of the endpoint to reconcile (default "default")
```

## How to build it?

:warning: Please don't forget to update the content of the `VERSION` file before creating a new release:

```bash
$ make release
```

This will build a Go binary, create a new Docker image with the tag you specified in the `Makefile`, push it to our image registry, and clean up afterwards.

## Example manifests

Please find an example Kubernetes manifest within the [`/example`](example) directory.
