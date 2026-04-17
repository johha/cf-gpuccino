# GPU Test Release

Validates GPU functionality using pre-compiled NVIDIA driver packages. Lifecycle errand creates a GPU VM on-demand, installs driver from BOSH packages, and runs PyTorch/TensorFlow validation tests.

## What It Does

- Installs NVIDIA driver from pre-compiled packages (~1 second)
- Installs Python and ML frameworks (runtime)
- Runs GPU validation tests (PyTorch, TensorFlow)
- VM destroyed automatically after completion


## Usage

```bash
# Deploy
bosh -d gpu-test deploy bosh/gpu-test-release/manifests/gpu-test.yml

# Run validation errand
bosh -d gpu-test run-errand gpu-validation-errand
```

Expected output:
```
✅ Driver installation complete (took ~1 second)
✅ Tesla T4, 570.211.01, 15360 MiB
✅ PyTorch tests PASSED
✅ TensorFlow tests PASSED
✅ GPU VALIDATION PASSED
```

## Architecture

The release uses pre-compiled NVIDIA driver packages:

```
packages/
├── nvidia-driver-570/
│   ├── module.tar.gz  - Kernel modules
│   ├── tools.tar.gz   - nvidia-smi
│   └── libs.tar.gz    - CUDA libraries
└── nvidia-container-toolkit/
    └── toolkit.deb    - nvidia-ctk

jobs/
└── gpu-validation-errand/
    ├── run            - Installs driver from packages
    └── test-gpu.py    - Validation tests
```

## Pre-Compiled Driver Packages

Packages contain artifacts compiled by `nvidia-compile-release`:

**module.tar.gz** - Kernel modules:
- nvidia.ko (main driver)
- nvidia-uvm.ko (unified memory)
- nvidia-modeset.ko, nvidia-drm.ko, etc.

**tools.tar.gz** - Management tools:
- nvidia-smi

**libs.tar.gz** - CUDA libraries:
- libnvidia-*.so, libcuda.so

**toolkit.deb** - Container toolkit:
- nvidia-ctk for container GPU access

### Updating Packages

When stemcell or driver version changes, recompile using `nvidia-compile-release`:

```bash
# 1. Compile artifacts (see ../nvidia-compile-release/README.md)
cd bosh/nvidia-compile-release
bosh -d nvidia-compile deploy manifest.yml  # Update stemcell version first
bosh -d nvidia-compile run-errand compile-nvidia --keep-alive

# 2. Download artifacts
cd ../
bosh -d nvidia-compile scp compile/0:/var/vcap/data/nvidia-compile/*.tar.gz ./driver-artifacts/
bosh -d nvidia-compile scp compile/0:/var/vcap/data/nvidia-compile/*.deb ./driver-artifacts/

# 3. Add as blobs
cd gpu-test-release
bosh add-blob ../driver-artifacts/module.tar.gz nvidia-driver-570/module.tar.gz
bosh add-blob ../driver-artifacts/tools.tar.gz nvidia-driver-570/tools.tar.gz
bosh add-blob ../driver-artifacts/libs.tar.gz nvidia-driver-570/libs.tar.gz
bosh add-blob ../driver-artifacts/toolkit.deb nvidia-container-toolkit/toolkit.deb

# 4. Create and upload release
bosh create-release --force --tarball=/tmp/gpu-test.tgz
bosh upload-release /tmp/gpu-test.tgz

# 5. Update stemcell version in manifests/gpu-test.yml to match
```

## Validation Tests

### PyTorch
- GPU detection
- FP32 matrix multiplication (4096x4096)
- FP16 matrix multiplication (Tensor Cores)

### TensorFlow
- GPU detection  
- FP32 matrix multiplication (4096x4096)
- FP16 matrix multiplication (Tensor Cores)

Both output JSON results for automation.

## Configuration

### Stemcell Version

**Critical:** Must match kernel version used to compile driver modules.

```yaml
stemcells:
  - alias: default
    os: ubuntu-jammy
    version: "1.1123"  # Must match nvidia-compile-release
```

Kernel modules compiled for 5.15.0-173-generic will NOT work on 5.15.0-175-generic.

### VM Type

```yaml
instance_groups:
  - name: gpu-validation
    vm_type: gpu-small  # g4dn.xlarge with Tesla T4
```

## Troubleshooting

### "modprobe: ERROR: could not insert 'nvidia'"

Kernel version mismatch. Stemcell version must match the one used in nvidia-compile-release.

Check kernel: `bosh -d gpu-test ssh gpu-validation/0 -c "uname -r"`

### "No GPU detected"

Wrong VM type or GPU not available. Verify `vm_type: gpu-small` points to GPU instance.

Check: `bosh -d gpu-test ssh gpu-validation/0 -c "lspci | grep -i nvidia"`

