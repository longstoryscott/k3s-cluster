# Steam Desktop with Selkies

Custom Docker image based on Selkies GLX Desktop with Steam pre-installed.

## Building the Image

```bash
# Make build script executable
chmod +x build-push.sh

# Build and optionally push to registry
./build-push.sh

# Or build with a specific tag
TAG=v1.0.0 ./build-push.sh
```

## Using the Custom Image

After building and pushing, update `selkies-desktop-official.yaml`:

```yaml
containers:
  - name: selkies-desktop
    image: registry.local:31500/steam-desktop:latest
    # ... rest of config
```

Then apply:

```bash
kubectl apply -f selkies-desktop-official.yaml
kubectl rollout restart statefulset/selkies-desktop-official -n steam
```

## What's Included

- Selkies GLX Desktop base image
- Steam client (official Valve package)
- All required 32-bit and 64-bit GL libraries
- Desktop portal integration (xdg-desktop-portal-kde)
- Pre-configured for immediate use

## Persistence

The following directories persist across container restarts:
- `/home/ubuntu` (50Gi) - User home directory, Steam data
- `/cache` (20Gi) - Additional cache storage

## GPU Selection

The StatefulSet is configured to use GPU 1 (RTX 3090) via the `GPU_SELECT` environment variable.

## Gamepad Support

To add gamepad support, add to the StatefulSet volumes section:

```yaml
volumeMounts:
  - name: dev-input
    mountPath: /dev/input
volumes:
  - name: dev-input
    hostPath:
      path: /dev/input
```

Then restart the pod.
