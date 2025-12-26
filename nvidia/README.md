# NVIDIA GPU Operator Configuration

## Current Setup

This cluster uses **pre-installed NVIDIA drivers on the host** rather than containerized drivers managed by the GPU Operator.

### Why Use Pre-installed Drivers?

- **Desktop Environment**: lsnode-3 runs a desktop environment (KDE/X11) that requires the NVIDIA driver
- **Driver In Use**: The driver modules cannot be unloaded while the desktop is running
- **Stability**: Pre-installed drivers avoid upgrade conflicts with running GPU processes

### What the Operator Manages

With `driver.enabled=false`, the operator only deploys:
- ✅ NVIDIA Container Toolkit (for container GPU access)
- ✅ NVIDIA Device Plugin (for GPU resource scheduling)
- ✅ GPU Feature Discovery (for GPU labeling)
- ✅ DCGM Exporter (for GPU metrics)
- ❌ Driver installation/management (disabled)

### Installation

```bash
cd nvidia
./install.sh
```

### Verifying GPU Access

```bash
# Check that pods can see GPUs
kubectl run gpu-test --rm -it --restart=Never \
  --image=nvidia/cuda:12.0.0-base-ubuntu22.04 \
  --limits=nvidia.com/gpu=1 \
  -- nvidia-smi
```

### Host Driver Requirements

Each GPU node must have:
1. NVIDIA driver installed on the host OS
2. Driver version compatible with your GPUs
3. Driver modules loaded at boot

Check driver version on a node:
```bash
ssh lsnode-3 nvidia-smi
```

### Troubleshooting

#### Issue: Pods can't access GPU
**Solution**: Verify driver is loaded on host and container toolkit is running:
```bash
kubectl get pods -n gpu-operator | grep container-toolkit
ssh <node> lsmod | grep nvidia
```

#### Issue: Wrong driver version
**Solution**: Update driver on the host OS, then restart the node:
```bash
ssh <node>
sudo ubuntu-drivers install nvidia:570  # or your preferred version
sudo reboot
```

#### Issue: GPU not detected
**Solution**: Check that GPU is visible to the host:
```bash
ssh <node> lspci | grep -i nvidia
```

## Switching to Containerized Drivers

If you want the operator to manage drivers (not recommended for desktop nodes):

1. Stop all GPU processes on the node (X server, desktop, etc.)
2. Edit `install.sh` and set `--set driver.enabled=true`
3. Remove `nvidia.com/gpu-driver-upgrade.skip` label if present
4. Run `./install.sh`

**Warning**: This will require periodic node reboots for driver upgrades and will conflict with desktop environments.
