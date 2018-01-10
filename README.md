# AWS Load Balancer Readvertiser

The AWS Load Balancer Readvertiser is a tool that is used for AWS [Shoot clusters](https://github.com/gardener/documentation/wiki/Architecture). The `kube-apiserver` of a Shoot cluster must be reachable by the `kubernetes` service in the `default` namespace (usually created with service ip `100.64.0.1`). In order to enable that, the apiserver must expose its public ip address. In the Shoot setup, the only way to reach it is via a public load balancer. However, in AWS you don't get an IP address for your load balancers, but only a hostname. The underlying IP address can change at any time. The detection of those changes is exactly the purpose of the Readvertiser. It will watch for them and update the `--advertise-address` flag of the `kube-apiserver` deployment with the correct IP properly.

## Constraints

The `kube-apiserver` deployment must reside in the same namespace as the Readvertiser has been deployed to.

## How to build it?

:warning: Please don't forget to update the content of the `VERSION` file before creating a new release:

```bash
$ make release
```

This will build a Go binary, create a new Docker image with the tag you specified in the `Makefile`, push it to our image registry, and clean up afterwards.

## Example manifests

Please find an example Kubernetes manifest within the [`/example`](example) directory.
