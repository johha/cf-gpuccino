# Shortcuts and Assumptions

This document tracks shortcuts taken during the PoC that would need to be addressed
for a production implementation.

---

## Accepted Shortcuts

### 1. GPU Validation Runs as Errand

**Current state**: The `gpu-validation-errand-noble` runs on-demand, creates VM, tests GPU,
then destroys VM automatically.

**Production approach**: For Diego GPU cells, tests would run during post-start and
report to BOSH health monitoring.

**Why this is acceptable**: Errand-based validation is perfect for development and
cost-effective for testing. Validates that the custom stemcell GPU driver works correctly.

---

### 2. Single GPU Instance Type Tested

**Current state**: Only tested on g4dn.xlarge (Tesla T4).

**Production approach**: Test on multiple instance types (T4, A10G, A100, H100)
and document any differences.

**Why this is acceptable**: T4 is representative of NVIDIA datacenter GPUs.
Driver installation process is the same across GPU types.

---

### 3. No Container GPU Access Yet

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
| NVIDIA driver installation | ✅ Validated | Pre-baked in custom stemcell (ubuntu-noble 1.365-nvidia) |
| Kernel module loading | ✅ Validated | `nvidia-smi` works on boot |
| CUDA compute | ✅ Validated | PyTorch GPU tests pass (4.45 TFLOPS FP32, 40.79 TFLOPS FP16) |
| Stemcell compatibility | ✅ Validated | Ubuntu Noble with driver 595.58.03 |
| BOSH job lifecycle | ✅ Validated | pre-start, post-start, monit all work |

---

## Future Work

When moving toward production, address these in order:

1. ~~**Custom stemcell**~~ - ✅ Complete (ubuntu-noble 1.365-nvidia with pre-baked NVIDIA driver)
2. **Multi-GPU testing** - Validate on A10G, A100 instances
3. **Container integration** - nvidia-container-toolkit + Garden changes
