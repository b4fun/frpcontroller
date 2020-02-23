# Get Start

This guide help us get start with installing & using frpcontroller in your k8s cluster.

## What's this?

[frp (fast reverse proxy)][frp] is a simple yet powerful reverse proxy for exposing local server behind NAT/firewall to the Internet. You can read more in [frp][frp]'s repostiory.

`frpcontroller` is a Kubernetes controller for managing frp endpoints & services.

[frp]: https://github.com/fatedier/frp

## Installation

We can install `frpcontroller` with pre-generated specs:

```
$ kubectl apply -f https://raw.githubusercontent.com/b4fun/frpcontroller/master/release/latest/install.yaml
namespace/frpcontroller-system created
customresourcedefinition.apiextensions.k8s.io/endpoints.frp.go.build4.fun created
customresourcedefinition.apiextensions.k8s.io/services.frp.go.build4.fun created
role.rbac.authorization.k8s.io/frpcontroller-leader-election-role created
clusterrole.rbac.authorization.k8s.io/frpcontroller-manager-role created
clusterrole.rbac.authorization.k8s.io/frpcontroller-proxy-role created
rolebinding.rbac.authorization.k8s.io/frpcontroller-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/frpcontroller-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/frpcontroller-proxy-rolebinding created
service/frpcontroller-controller-manager-metrics-service created
deployment.apps/frpcontroller-controller-manager created
```

This will install & setup `frpcontroller` under `frpcontroller-system` namespace.

```
$ kubectl -n frpcontroller-system get pods
$ kubectl -n frpcontroller-system get pods -w
NAME                                               READY   STATUS    RESTARTS   AGE
frpcontroller-controller-manager-7bf6d5f7f-tb998   2/2     Running   0          20s
```

## Expose my service

Before exposing the traffic, we need to setup a frp server endpoint which is public accessible.
Let's say we have a `frps.ini` config like this:

```ini
[common]
bind_port = 1111
log_level = info
token = helloworld
```

And server's public ip is `<server-ip>`.

### Create an `Endpoint`

The first step we need to do is create an `Endpoint` resource to describe the public server endpoint.

```yaml
apiVersion: frp.go.build4.fun/v1
kind: Endpoint
metadata:
  name: hello-endpoint
spec:
  addr: '<server-ip>'
  # bind port of the frp server
  port: 1111
  # token of the frp server
  token: helloworld
```

And create it:

```
$ kubectl apply -f doc/example/endpoint.yaml
endpoint.frp.go.build4.fun/hello-endpoint created
$ kubectl get endpoint.frp.go.build4.fun
NAME             AGE
hello-endpoint   31s
```

### Create a `Service`

Next step is to create a `Service` resource to expose the underlying workload.
In this example, we'll expose an nginx pod:

```yaml
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.17.8-alpine
    ports:
      - containerPort: 80

---
apiVersion: frp.go.build4.fun/v1
kind: Service
metadata:
  name: hello-service
spec:
  endpoint: hello-endpoint
  # Just like corev1/Service, we use selector to select pods by labels.
  selector:
    app: nginx
  ports:
    - protocol: TCP
      # remotePort specifies the server side port to use
      remotePort: 8083
      # localPort specifies the client side service port to expose
      localPort: 80
      name: nginx-80
```

The create it:

```
$ kubectl apply -f doc/example/client_service.yaml
pod/nginx created
service.frp.go.build4.fun/hello-service created
$ kubectl get service.frp.go.build4.fun
NAME            AGE
hello-service   73s
```

This step will create a `corev1/Service` resource:

```
$ kubectl get service
NAME                       TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
hello-service-frpc-srt55   ClusterIP   10.0.70.159   <none>        8083/TCP   111s
```

...and a sidecar frp pod:

```
$ kubectl get pods
NAME                        READY   STATUS             RESTARTS   AGE
hello-endpoint-frpc-ph5rw   1/1     Running            0          2m10s
nginx                       1/1     Running            0          2m14s
```

Now the client service had been exposed to frp server's 8083 port, we can acces it with:

```
$ curl http://<server-ip>:8083
```

The full example can be found from [example](./example).

## Uninstallation

```
$ kubectl delete -f https://raw.githubusercontent.com/b4fun/frpcontroller/master/release/latest/install.yaml
namespace "frpcontroller-system" deleted
customresourcedefinition.apiextensions.k8s.io "endpoints.frp.go.build4.fun" deleted
customresourcedefinition.apiextensions.k8s.io "services.frp.go.build4.fun" deleted
role.rbac.authorization.k8s.io "frpcontroller-leader-election-role" deleted
clusterrole.rbac.authorization.k8s.io "frpcontroller-manager-role" deleted
clusterrole.rbac.authorization.k8s.io "frpcontroller-proxy-role" deleted
rolebinding.rbac.authorization.k8s.io "frpcontroller-leader-election-rolebinding" deleted
clusterrolebinding.rbac.authorization.k8s.io "frpcontroller-manager-rolebinding" deleted
clusterrolebinding.rbac.authorization.k8s.io "frpcontroller-proxy-rolebinding" deleted
service "frpcontroller-controller-manager-metrics-service" deleted
deployment.apps "frpcontroller-controller-manager" deleted
```