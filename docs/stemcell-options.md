# GPU Stemcell Options

This document describes options for deploying NVIDIA drivers on BOSH-managed GPU VMs.

## The Problem

Installing NVIDIA drivers at deploy time (via apt) takes **10-15 minutes** per VM because:
1. Download ~1.5GB of packages from Ubuntu repos
2. DKMS compiles kernel module against running kernel
3. Module loading and verification

For production GPU cells, this is unacceptable:
- Slow scaling when adding new cells
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

### Option 2: BOSH Package with Pre-compiled Driver

Package the NVIDIA driver as a BOSH blob and copy at deploy time.

**Pros:**
- Faster than apt (no download, no DKMS compile)
- Works with standard stemcells
- Version controlled in BOSH release

**Cons:**
- Must match kernel version exactly
- Requires rebuilding package when stemcell kernel changes
- More complex packaging

**Implementation:**
1. Compile nvidia.ko on a matching stemcell
2. Package as BOSH blob (driver binaries + kernel module)
3. Job copies files and runs `insmod` / `modprobe`

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

| Option | Deploy Time | Maintenance | Production Ready |
|--------|------------|-------------|------------------|
| Custom Stemcell | ~1 min | High | Yes |
| BOSH Package | ~2-3 min | Medium | Yes |
| AWS GPU AMI | ~1 min | Low | Maybe |
| Local Apt Mirror | ~7-8 min | Low | Partial |
| Warm Pool | ~1 min | Medium | Yes |

---

## Recommendation

**For PoC/Development:** Runtime apt install (current approach in `gpu-test-release`) is fine.
Allows easy testing of different driver versions.

**For Pre-Production:** Compiled BOSH release with pre-built driver blobs.
Reduces deploy time from 10-15 min to 2-3 min while still using standard stemcells.

**For Production CF GPU Cells:** Custom stemcell with pre-installed driver.
The stemcell approach ensures:
- Predictable, fast deploys (~1 min)
- No external dependencies during cell creation
- Consistent driver versions across the fleet
- Simpler troubleshooting (driver issues are stemcell issues)

### Recommended Path

1. Start with runtime apt install (current PoC)
2. Move to compiled release for staging/pre-prod
3. Build custom stemcell for production

This hybrid approach allows flexibility during development while ensuring
production readiness.

---

## Future Work

- [ ] Document custom stemcell build process for NVIDIA drivers
- [ ] Create stemcell CI pipeline that tracks driver updates
- [ ] Test BOSH package approach as alternative
- [ ] Evaluate nvidia-container-toolkit pre-installation in stemcell
