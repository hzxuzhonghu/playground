## Install Istio ambient mesh and Kmesh

## Add namespace to kmesh

```bash
kubectl create ns test
kubectl label ns test istio.io/dataplane-mode=kmesh
```

## Deploy fortio

```bash
kubectl apply -f fortio-client.yaml
kubectl apply -f fortio-server.yaml
```

## Configure fortio-server to use a specific waypoint

we can deploy a waypoint called fortio-server-waypoint for the fortio-server service:


Istio 1.22

```bash
istioctl x waypoint apply -n test --name fortio-server-waypoint

```
Istio 1.23

```bash
istioctl waypoint apply -n test --name fortio-server-waypoint

```

**Note** We should customize the waypoint with kmesh-waypoint image.

```bash
kubectl annotate gateway fortio-server-waypoint sidecar.istio.io/proxyImage=ghcr.io/kmesh-net/waypoint:latest
```

Label the fortio-server service to use the fortio-server-waypoint waypoint:

```bash
kubectl label service fortio-server -n test istio.io/use-waypoint=fortio-server-waypoint
```


## Run fortio test

```
sh ../test.sh
```
