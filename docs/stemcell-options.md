# GPU Stemcell Options

This document describes options for deploying NVIDIA drivers on BOSH-managed GPU VMs.

## The Problem

Installing NVIDIA drivers at deploy time (via apt) takes **5-7 minutes** per VM because:
1. Download packages from Ubuntu repos
2. DKMS compiles kernel module against running kernel (~5-7 min)
3. Module loading and verification

For production GPU cells and frequent testing, this is slow:
- Slow iteration during development
- Long recovery time if a cell is recreated
- External dependency on Ubuntu apt repos

---

## Options

### Option 1: Custom Stemcell (Recommended for Production)

Bake the NVIDIA driver directly into the stemcell image.

**Pros:**
- Zero runtime driver installation
- Consistent driver version across all VMs
- No external dependencies at deploy time
- Fastest possible deploy/recreate

**Cons:**
- Must maintain custom stemcell pipeline
- Stemcell updates require driver re-integration
- Larger stemcell image size (~2GB larger)

**Implementation:**
```bash
# Build custom stemcell with NVIDIA driver
# See: https://bosh.io/docs/build-stemcell/

# Key steps in stemcell builder:
# 1. Add graphics-drivers PPA
# 2. Install nvidia-driver-570 nvidia-utils-570
# 3. Pre-compile DKMS module for target kernel
# 4. Include nvidia-persistenced systemd service
```

**Stemcell naming convention:**
```
bosh-aws-xen-hvm-ubuntu-jammy-go_agent-nvidia570
```

---

### Option 2: BOSH Package with Pre-compiled Driver ✅ Implemented

Package the NVIDIA driver as a BOSH blob and copy at deploy time.

**Pros:**
- Very fast deployment (~1 second driver install)
- Works with standard stemcells
- Version controlled in BOSH release
- No external dependencies at deploy time

**Cons:**
- Must match kernel version exactly
- Requires recompiling when stemcell kernel changes

**Implementation:**
✅ **Complete** - See `bosh/nvidia-compile-release/` and `bosh/gpu-test-release/`

1. Compile driver using `nvidia-compile-release` (one-time per stemcell version)
2. Package as BOSH blobs (module.tar.gz, tools.tar.gz, libs.tar.gz)
3. Job copies files and runs `modprobe`

**Measured Performance:**
- Driver installation: ~1 second (vs 5-7 min DKMS)
- Compilation time: ~10 min (one-time per stemcell)

**See:** `bosh/nvidia-compile-release/README.md` for details

---

### Option 3: AWS GPU AMI

Use an AMI that already has NVIDIA drivers installed.

**Pros:**
- AWS maintains GPU-optimized AMIs
- Drivers pre-installed and tested

**Cons:**
- Not a standard BOSH stemcell
- May have compatibility issues with BOSH agent
- Less control over driver version

**AWS GPU AMIs:**
- Deep Learning AMI (Ubuntu)
- Amazon Linux 2 with NVIDIA driver

---

### Option 4: Local Apt Mirror / Proxy

Cache apt packages locally to eliminate download time.

**Pros:**
- Simple to implement
- Works with existing scripts
- Reduces external dependencies

**Cons:**
- Still requires DKMS compile (5-7 min)
- Requires maintaining apt mirror infrastructure

**Implementation:**
- Deploy apt-cacher-ng or Artifactory
- Configure VMs to use local apt proxy
- Pre-seed cache with nvidia packages

---

### Option 5: nvidia-persistenced + Warm Pool

Keep spare GPU VMs in a "warm" state with drivers pre-installed.

**Pros:**
- Standard stemcells
- Fast scaling from warm pool

**Cons:**
- Costs money for idle VMs
- Complex orchestration

---

## Comparison

| Option | Deploy Time | Maintenance | Production Ready | Status |
|--------|------------|-------------|------------------|--------|
| Custom Stemcell | ~0 sec | High | Yes | Future |
| **BOSH Package** | **~1 sec** | **Medium** | **Yes** | **✅ Implemented** |
| AWS GPU AMI | ~0 sec | Low | Maybe | Not pursued |
| Local Apt Mirror | ~7-8 min | Low | Partial | Not pursued |
| Warm Pool | ~0 sec | Medium | Yes | Not needed |

---

## Recommendation

**Current Status:** ✅ Pre-compiled BOSH packages implemented (Option 2)
- Driver installation: ~1 second
- Validated with PyTorch and TensorFlow
- Production-ready approach

**For Development/Testing:** Use pre-compiled BOSH packages (current implementation).
Fast iteration, works with standard stemcells.

**For Production CF GPU Cells:** Two viable options:

1. **Pre-compiled BOSH packages** (current) - Ready to use now
   - ~1 second driver install
   - Medium maintenance (recompile on stemcell updates)
   - Works with standard stemcells

2. **Custom stemcell** (future) - Zero runtime install
   - Eliminates driver installation entirely
   - Higher maintenance (stemcell pipeline)
   - Best for large-scale deployments

### Current Implementation

✅ `nvidia-compile-release` - Compiles driver for specific stemcell  
✅ `gpu-test-release` - Uses pre-compiled packages  
✅ Driver artifacts tracked with git-lfs  
✅ Validated on Tesla T4 with stemcell 1.1123

---

## Future Work

- [ ] Document custom stemcell build process for NVIDIA drivers
- [ ] Create stemcell CI pipeline that tracks driver updates
- [x] ~~Test BOSH package approach~~ - ✅ Implemented and validated
- [ ] Evaluate nvidia-container-toolkit pre-installation in stemcell
