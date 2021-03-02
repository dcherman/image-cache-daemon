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
  -h, --help                               help for image-cache-daemon
      --image stringArray                  Images that should be pre-fetched
      --node-name string                   The node name to pull to
      --pod-name string                    The pod name
      --pod-namespace string               The namespace this pod is running in
      --pod-uid string                     The owning pod UID
      --resync-period duration             How often the daemon should re-pull images from all of the sources.  Set to 0 to disable. (default 15m0s)
      --warden-image string                The image that copies a binary to pulled containers to replace the entrypoint (default "exiges/image-cache-warden:latest")
      --watch-cluster-workflow-templates   Whether or not to watch cluster workflow templates (default true)
      --watch-cron-workflows               Whether or not to watch cron workflows (default true)
      --watch-workflow-templates           Whether or not to watch workflow templates (default true)
```
