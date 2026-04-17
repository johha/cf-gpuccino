# Shortcuts and Assumptions

This document tracks shortcuts taken during the PoC that would need to be addressed
for a production implementation.

---

## Accepted Shortcuts

### 1. Pre-Compiled Driver Packages (vs Custom Stemcell)

**Current state**: NVIDIA driver installed from pre-compiled BOSH packages (~1 second).

**Production approach**: Custom stemcell with pre-baked driver (zero install time).

**Why this is acceptable**:
- Pre-compiled packages reduce driver install from 5-7 min to ~1 second
- Validated approach on Ubuntu Jammy stemcell kernel (5.15.0-173-generic)
- For GPU validation errands and testing, this is sufficient
- Custom stemcells recommended for production Diego GPU cells

**Effort to fix**: 1-2 weeks (learn stemcell-builder, add NVIDIA stage, set up CI).

**See also**: [Stemcell Options](stemcell-options.md)

**What we've built**:
- `nvidia-compile-release` - compiles driver for specific stemcell kernel
- `gpu-test-release` - uses pre-compiled packages via BOSH blobs
- Driver artifacts tracked with git-lfs

---

### 2. GPU Validation Tests Run as Errand

**Current state**: The `gpu-validation-errand` runs on-demand, creates VM, tests GPU, 
then destroys VM automatically.

**Production approach**: For Diego GPU cells, tests would run during post-start and
report to BOSH health monitoring.

**Why this is acceptable**: Errand-based validation is perfect for development and
cost-effective for testing. Validates the pre-compiled driver approach.

---

### 3. Single GPU Instance Type Tested

**Current state**: Only tested on g4dn.xlarge (Tesla T4).

**Production approach**: Test on multiple instance types (T4, A10G, A100, H100)
and document any differences.

**Why this is acceptable**: T4 is representative of NVIDIA datacenter GPUs.
Driver installation process is the same across GPU types.

---

### 4. No Container GPU Access Yet

**Current state**: GPU works at VM level, but not exposed to containers.

**Production approach**: Requires nvidia-container-toolkit and Garden/runc
integration (Phase 3 of roadmap).

**Why this is acceptable**: VM-level GPU validation is prerequisite for
container integration. We're following the roadmap phases.

---

## Technical Validations Completed

These items are **not shortcuts** - they've been fully validated:

| Item | Status | Evidence |
|------|--------|----------|
| GPU hardware detection | ✅ Validated | `lspci` shows Tesla T4 |
| NVIDIA driver installation | ✅ Validated | Driver 570 installs cleanly |
| Kernel module loading | ✅ Validated | `nvidia-smi` works |
| CUDA compute | ✅ Validated | PyTorch GPU tests pass (4.4 TFLOPS) |
| Stemcell compatibility | ✅ Validated | Ubuntu Jammy kernel 5.15.x works |
| BOSH job lifecycle | ✅ Validated | pre-start, post-start, monit all work |

---

## Future Work

When moving toward production, address these in order:

1. ~~**Compiled BOSH release**~~ - ✅ Complete (nvidia-compile-release + pre-compiled packages)
2. **Custom stemcell** - Bake driver into stemcell image (higher effort, optional)
3. **Multi-GPU testing** - Validate on A10G, A100 instances
4. **Container integration** - nvidia-container-toolkit + Garden changes
