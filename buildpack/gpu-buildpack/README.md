# GPU Buildpack for Cloud Foundry

A Cloud Foundry buildpack that configures GPU-enabled application containers at staging time. It sets the necessary environment variables (`CUDA_VISIBLE_DEVICES`, `LD_LIBRARY_PATH`, `HIP_VISIBLE_DEVICES`) and mounts GPU-specific dependencies.

---

## Usage

### 1. Mark your app as GPU-enabled

Create a marker file in your app directory:

```bash
touch .gpu-enabled
```

Or, if you have GPU-specific Python requirements, create `requirements-gpu.txt`:

```
torch==2.2.0+cu118
torchvision==0.17.0+cu118
--extra-index-url https://download.pytorch.org/whl/cu118
```

### 2. Push with the GPU buildpack support

```bash
cf push myapp -b python_buildpack
```

Or in your `manifest.yml`:

```yaml
applications:
  - name: myapp
    buildpacks:
      - python_buildpack
    resources:
      memory: 2G
      gpu: 1
      gpu_type: nvidia
```

---

## Environment Variables Set

The buildpack writes `.profile.d/gpu-env.sh` which is sourced at every container start:

| Variable | Default | Description |
|---|---|---|
| `CUDA_VISIBLE_DEVICES` | `NoDevFiles` (if not set by executor) | GPU indices visible to CUDA. The Diego Executor overrides this via CDI injection. |
| `LD_LIBRARY_PATH` | `/usr/local/cuda/lib64:/usr/lib/x86_64-linux-gnu:…` | Prepends CUDA runtime library paths. |
| `HIP_VISIBLE_DEVICES` | `` (empty) | AMD/ROCm GPU visibility. Set by the AMD CDI spec. |

---

## CDI Integration

At runtime the Diego Executor injects the GPU into the container using CDI (Container Device Interface):

1. The Executor calls `GPUManager.Allocate()` to reserve a GPU index.
2. It adds `CDIDevices: [{Name: "nvidia.com/gpu=0"}]` to the `ContainerSpec`.
3. garden-runc passes the CDI name to runc which resolves it via the CDI registry.
4. runc mounts `/dev/nvidia0`, `/dev/nvidiactl`, `/dev/nvidia-uvm`, and the NVIDIA library tree into the container.
5. The CDI spec sets `CUDA_VISIBLE_DEVICES=0` – this takes precedence over the buildpack default.

The buildpack's `.profile.d/gpu-env.sh` provides safe defaults so the app does not crash if it is run without GPU (e.g. during staging health checks).

---
