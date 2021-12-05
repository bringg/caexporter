# Cluster Autoscaler Exporter

A Prometheus Exporter for reporting the last activity of the Kubernetes Cluster Autoscaler, gathered from the CAS config map.

## Usage

```shell
usage: caexporter [<flags>]

Flags:
  --help              Show context-sensitive help (also try --help-long and --help-man).
  --web.listen-address=":8080"
                      Address to listen on for web interface and telemetry.
  --web.telemetry-path="/metrics"
                      Path under which to expose metrics.
  --collector.request-timeout="10"
                      Kubernetes API request timeout in seconds.
  --log.debug="false" Sets log level to debug.
```

## Run with Docker

```shell
docker run -p 8080:8080 bringg/caexporter
```

## Running in a Kubernetes cluster

This exporter requires certain permissions in order to get the status from the Cluster Autoscaler config map.

The `kubernetes.yaml` file includes manifests for a simple deployment that runs the exporter image, and the necessary role and role binding resources.

To run the exporter in your Kubernetes cluster, please make sure to install these resources:

```shell
kubectl apply -f kubernetes.yaml
```
