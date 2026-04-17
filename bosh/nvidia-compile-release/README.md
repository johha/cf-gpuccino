# NVIDIA Driver Compilation Release

Compiles NVIDIA driver artifacts for a specific stemcell kernel version. Output artifacts are used by gpu-test-release as BOSH blobs for fast driver installation.

## What It Does

BOSH errand that:
- Downloads NVIDIA .run installer (570.211.01)
- Compiles kernel modules against stemcell's kernel
- Packages modules, tools, and libraries as tarballs
- Downloads nvidia-container-toolkit

## When to Use

Recompile when:
- Stemcell version changes (different kernel)
- NVIDIA driver version changes
- Initial setup


## Usage

```bash
# Create and upload release
cd bosh/nvidia-compile-release
bosh create-release --force --tarball=/tmp/nvidia-compile.tgz
bosh upload-release /tmp/nvidia-compile.tgz

# Deploy and run errand
bosh -d nvidia-compile deploy manifest.yml
bosh -d nvidia-compile run-errand compile-nvidia --keep-alive

# Download artifacts
cd ../
bosh -d nvidia-compile scp compile/0:/var/vcap/data/nvidia-compile/module.tar.gz ./driver-artifacts/
bosh -d nvidia-compile scp compile/0:/var/vcap/data/nvidia-compile/tools.tar.gz ./driver-artifacts/
bosh -d nvidia-compile scp compile/0:/var/vcap/data/nvidia-compile/libs.tar.gz ./driver-artifacts/
bosh -d nvidia-compile scp compile/0:/var/vcap/data/nvidia-compile/toolkit.deb ./driver-artifacts/

# Cleanup
bosh -d nvidia-compile delete-deployment
```

## Output Artifacts

Located at `/var/vcap/data/nvidia-compile/` on the compile VM:

- `module.tar.gz` - Kernel modules (nvidia.ko, nvidia-uvm.ko, etc.)
- `tools.tar.gz` - nvidia-smi and utilities
- `libs.tar.gz` - CUDA libraries (libnvidia-*.so, libcuda.so)
- `toolkit.deb` - nvidia-container-toolkit

## Configuration

### Change Stemcell Version

Edit `manifest.yml`:
```yaml
stemcells:
  - alias: default
    os: ubuntu-jammy
    version: "1.1123"  # Update to target stemcell
```

### Change Driver Version

Edit `jobs/compile-nvidia/templates/run`:
```bash
DRIVER_VERSION=570.211.01  # Update version
INSTALLER_URL="https://us.download.nvidia.com/XFree86/Linux-x86_64/${DRIVER_VERSION}/NVIDIA-Linux-x86_64-${DRIVER_VERSION}.run"
```

## How It Works

The errand:
1. Installs build-essential and kernel headers
2. Downloads NVIDIA .run installer from nvidia.com
3. Compiles driver with `./installer.run --silent --kernel-source-path=...`
4. Packages compiled artifacts as tarballs

**Why .run installer instead of apt?**
- Single download vs 100+ apt packages
- No dpkg filling root filesystem
- Direct compilation control

**Why BOSH VM instead of Docker?**
- Exact stemcell kernel match
- No emulation issues on Apple Silicon
- Same environment as deployment targets

## Next Steps

After downloading artifacts, add them to gpu-test-release as BOSH blobs.

See `../gpu-test-release/README.md` for details.

