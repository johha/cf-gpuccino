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

### Option 1: Custom Stemcell ✅ Implemented

Bake the NVIDIA driver directly into the stemcell image.

**Pros:**
- Zero runtime driver installation
- Consistent driver version across all VMs
- No external dependencies at deploy time
- Works for any BOSH deployment (not just errands)
- Fastest possible deploy/recreate

**Cons:**
- Must maintain custom stemcell pipeline
- Stemcell updates require driver re-integration
- Larger stemcell image

**Implementation:**
Custom Ubuntu Noble stemcell with NVIDIA driver pre-baked.

Source: https://github.com/johha/bosh-linux-stemcell-builder/tree/nvidia-595-cuda12.9-v1.460

```yaml
stemcells:
  - alias: default
    os: ubuntu-noble
    version: "1.562-nvidia"
```

---

### Option 2: BOSH Package with Pre-compiled Driver

Package the NVIDIA driver as a BOSH blob and copy at deploy time.

**Pros:**
- Fast deployment (~1 second driver install)
- Works with standard stemcells
- No external dependencies at deploy time

**Cons:**
- Must match kernel version exactly
- Requires recompiling when stemcell kernel changes
- Only works within BOSH releases (not for Diego cells directly)

**Implementation:**
See git history (commit 43fd8ab) for the full implementation using `nvidia-compile-release`.

Key steps:
1. Deploy `nvidia-compile-release` errand against target stemcell
2. Package compiled artifacts as BOSH blobs (module.tar.gz, tools.tar.gz, libs.tar.gz)
3. Job copies files at deploy time and runs `modprobe`

Compilation time: ~5 minutes (one-time per stemcell kernel version)
Installation time: ~1 second

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

---

### Option 4: Local Apt Mirror / Proxy

Cache apt packages locally to eliminate download time.

**Pros:**
- Simple to implement

**Cons:**
- Still requires DKMS compile (~5-7 min)
- Requires maintaining apt mirror infrastructure

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
| **Custom Stemcell** | **~0 sec** | **Medium** | **Yes** | **✅ Implemented** |
| BOSH Package | ~1 sec | Medium | Yes | In git history |
| AWS GPU AMI | ~0 sec | Low | Maybe | Not pursued |
| Local Apt Mirror | ~7-8 min | Low | Partial | Not pursued |
| Warm Pool | ~0 sec | Medium | Yes | Not needed |

---

## Recommendation

**Use the custom stemcell approach** (Option 1). It is the cleanest solution:
- Driver is always present on boot
- Works for Diego cells, errands, any BOSH deployment
- No per-release package management

The pre-compiled BOSH package approach (Option 2) remains a valid alternative when a custom stemcell pipeline is not available. It achieves ~1 second install time and is fully documented in git history.
