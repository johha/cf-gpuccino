# Shortcuts and Assumptions

This document tracks shortcuts taken during the PoC that would need to be addressed
for a production implementation.

---

## Accepted Shortcuts

### 1. Runtime Driver Installation (vs Custom Stemcell)

**Current state**: NVIDIA driver is installed via apt at deploy time (~10-15 min).

**Production approach**: Custom stemcell with pre-baked driver (~1 min deploy).

**Why this is acceptable**:
- We've validated that the driver installs and works correctly on the Ubuntu Jammy
  stemcell kernel (5.15.x)
- The technical risk is retired - it's now a question of effort/knowledge to build
  the custom stemcell
- For PoC/development, the runtime install is sufficient

**Effort to fix**: 1-2 weeks (learn stemcell-builder, add NVIDIA stage, set up CI).

**See also**: [Stemcell Options](stemcell-options.md)

---

### 2. GPU Validation Tests Run Manually

**Current state**: The `gpu-validation` job installs PyTorch but tests must be
triggered manually via `run-validation` script.

**Production approach**: Tests should run automatically during post-start and
report results to BOSH health monitoring.

**Why this is acceptable**: Manual testing is fine for PoC validation.

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

When moving toward production, address these shortcuts in order:

1. **Compiled BOSH release** - Pre-package driver binaries (medium effort)
2. **Custom stemcell** - Bake driver into stemcell image (higher effort)
3. **Multi-GPU testing** - Validate on A10G, A100 instances
4. **Container integration** - nvidia-container-toolkit + Garden changes
