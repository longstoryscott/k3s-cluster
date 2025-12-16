# Gaming Desktop Solutions Comparison

## Solutions Evaluated

### 1. **Selkies-GStreamer** ⭐ **RECOMMENDED**
**Best for: Gaming performance, Kubernetes-native deployment, low latency**

**Pros:**
- ✅ Built specifically for gaming and GPU workloads in containers
- ✅ WebRTC streaming = ultra-low latency (<20ms)
- ✅ NVIDIA hardware encoding (NVENC) for optimal performance
- ✅ Native Kubernetes StatefulSet deployment
- ✅ Game controller and Bluetooth peripheral support
- ✅ Maintained by Google (active development)
- ✅ Full GPU acceleration for gaming
- ✅ 60+ FPS streaming capability
- ✅ Minimal maintenance required

**Cons:**
- ⚠️ Newer project (less battle-tested than VNC)
- ⚠️ Requires WebRTC-capable browser

**Use Case:** Perfect for your k3s gaming setup with RTX 2060

### 2. **Moonlight/Sunlight**
**Best for: Traditional PC gaming streaming, non-containerized environments**

**Pros:**
- ✅ Excellent gaming performance (designed for NVIDIA GameStream)
- ✅ Very low latency
- ✅ Mature, stable solution
- ✅ Great controller support
- ✅ Hardware encoding

**Cons:**
- ❌ Not designed for containers/Kubernetes
- ❌ Requires host machine setup (Sunlight server)
- ❌ More complex networking in containerized environments
- ❌ Less suitable for your cluster architecture

### 3. **Kasm Workspaces**
**Best for: Enterprise VDI, productivity workloads**

**Pros:**
- ✅ Enterprise-grade platform
- ✅ Good container support
- ✅ Professional interface

**Cons:**
- ❌ Not optimized for gaming
- ❌ Commercial/enterprise focus
- ❌ Kubernetes support still in Technical Preview
- ❌ Higher resource overhead
- ❌ More complex than needed for gaming

---

## Migration Plan: noVNC → Selkies

### Current Issues with noVNC
- High latency (>100ms typical)
- Poor mouse precision for gaming
- No hardware acceleration for video encoding
- Limited controller support

### Selkies Improvements
- **Latency**: ~10-20ms (WebRTC vs VNC)
- **Mouse**: Pixel-perfect, low-latency input
- **Encoding**: NVENC hardware acceleration
- **Controllers**: Full gamepad/joystick support
- **Performance**: 60+ FPS vs ~30 FPS with noVNC

### Installation

1. **Deploy Selkies Desktop:**
   ```bash
   cd steam/
   ./install-selkies.sh
   ```

2. **Update Gateway Routes:**
   ```bash
   cd router/
   kubectl apply -f routes.yaml
   ```

3. **Access Gaming Desktop:**
   - URL: `http://<cluster-ip>:8087`
   - Username: `user`
   - Password: `mypasswd`

4. **Setup Steam:**
   ```bash
   # Inside the desktop, run:
   /home/user/install-steam.sh
   ```

### Storage Migration (Optional)
If you want to migrate existing Steam games from noVNC:

```bash
# Scale down old deployment
kubectl scale statefulset desktop-statefulset -n steam --replicas=0

# Copy game data (if needed)
kubectl run migrate --rm -it --image=busybox -- sh
# Mount both old and new PVCs and copy data
```

### Resource Allocation
- **CPU**: 2-8 cores (dynamic scaling)
- **Memory**: 4-16GB (dynamic scaling)  
- **GPU**: Full RTX 2060 access
- **Storage**: 
  - 100GB user home directory
  - 200GB Steam games storage
  - 100GB application storage

### Performance Optimizations
- NVIDIA NVENC hardware encoding
- 60 FPS @ 1920x1080 resolution
- WebRTC for sub-20ms latency
- Game controller support with SDL2
- Persistent game storage across restarts

### Browser Compatibility
- **Chrome/Chromium**: ✅ Best performance
- **Firefox**: ✅ Full support  
- **Safari**: ⚠️ Limited WebRTC support
- **Edge**: ✅ Full support

---

## Technical Architecture

### Selkies Stack
```
Browser (WebRTC) 
    ↓ 
NGINX Gateway (:8087)
    ↓
Selkies Container (:8080)
    ↓
GPU-accelerated Desktop (XFCE4)
    ↓
Steam + Games (RTX 2060)
```

### Key Components
- **Streaming**: WebRTC with STUN/TURN
- **Encoding**: NVIDIA NVENC H.264
- **Desktop**: XFCE4 (lightweight, gaming-optimized)
- **Audio**: PulseAudio with Opus codec
- **Input**: Gamepad.js for controller support

### Network Flow
1. WebRTC establishes P2P connection to container
2. NVIDIA GPU renders game frames
3. NVENC encodes frames in hardware  
4. WebRTC streams to browser in real-time
5. Input (mouse/keyboard/gamepad) sent back via WebRTC

---

## Troubleshooting

### Common Issues

**No GPU acceleration:**
```bash
kubectl exec -it selkies-desktop-0 -n steam -- nvidia-smi
```

**Audio not working:**
```bash
kubectl logs selkies-desktop-0 -n steam | grep -i audio
```

**Controller not detected:**
```bash
kubectl exec -it selkies-desktop-0 -n steam -- ls -la /dev/input/
```

**Poor streaming quality:**
- Check browser WebRTC settings
- Verify NVENC is active: `nvidia-smi dmon -s u`
- Monitor network latency to cluster

### Performance Monitoring
```bash
# Check container resources
kubectl top pods -n steam

# Monitor GPU usage  
kubectl exec -it selkies-desktop-0 -n steam -- nvidia-smi dmon

# View Selkies metrics
kubectl logs selkies-desktop-0 -n steam | grep -i fps
```

---

## Conclusion

**Selkies-GStreamer is the optimal choice** for your k3s gaming cluster because it:

1. **Solves your latency issues** with WebRTC (~10-20ms vs 100ms+ VNC)
2. **Improves mouse precision** with hardware-accelerated streaming
3. **Maximizes RTX 2060 utilization** with NVENC encoding
4. **Provides native Kubernetes support** without complex host dependencies
5. **Requires minimal maintenance** compared to Moonlight/Sunlight host setup
6. **Supports gaming peripherals** including controllers and Bluetooth devices

The migration from noVNC will provide a dramatically better gaming experience while maintaining the same cluster-native deployment model you're already using.