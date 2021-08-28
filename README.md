## image-cache-daemon

Image Cache Daemon is a service to pre-pull / cache images on Kubernetes before they're needed

### Synopsis

When it's desirable to run a container as fast as possible (such as when using Argo Workflows), image pull
time can be a significant contributor to slow pod start times.  The image cache daemon is a service that's intended to help
mitigate that by discovering images to pull from a variety of sources, then pulling those images on each node before they're
actually needed.

## Installation

```bash
kubectl apply -f https://raw.githubusercontent.com/dcherman/image-cache-daemon/master/manifests/install.yaml -n image-cache-daemon
```

```
image-cache-daemon [flags]
```

### Options

```
      --configmap-selector string               The selector to use when monitoring for ConfigMap sources (default "app.kubernetes.io/part-of=image-cache-daemon")
  -h, --help                                    help for image-cache-daemon
      --image stringArray                       Images that should be pre-fetched
      --node-name string                        The node name to pull to
      --pod-name string                         The pod name
      --pod-namespace string                    The namespace this pod is running in
      --pod-uid string                          The owning pod UID
      --resync-period duration                  How often the daemon should re-pull images from all of the sources.  Set to 0 to disable. (default 15m0s)
      --warden-image string                     The image that copies a binary to pulled containers to replace the entrypoint (default "exiges/image-cache-warden:latest")
      --watch-argo-cluster-workflow-templates   Whether or not to watch cluster workflow templates (default true)
      --watch-argo-cron-workflows               Whether or not to watch cron workflows (default true)
      --watch-argo-workflow-templates           Whether or not to watch workflow templates (default true)
      --watch-configmaps                        Whether or not to watch ConfigMaps for images to pull.  Must match the --config-map-selector (default true)
```

## Sources

### ConfigMap

The ConfigMap source is useful when you want to separate the list of images that you're pulling from the installation of the cache daemon.  It's also useful if you have a dynamic list
of images to pull that aren't part of one of the other sources.

By default, all ConfigMaps that match the label selector `"app.kubernetes.io/part-of=image-cache-daemon"` will be considered as a source for the cache daemon in any namespace that it has privileges to read.  If you would like to restrict the set of ConfigMaps that it reads, you can so do by changing the selector, or restricting the namespaces that the cache daemon can read via RBAC.

Example Usage:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-image-configmap
  labels:
    app.kubernetes.io/part-of: image-cache-daemon
data:
  images: |
    ["alpine", "debian"]
```