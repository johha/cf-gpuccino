# GPU Test Release

Validates GPU functionality on BOSH-managed VMs using a GPU-enabled stemcell.
Two lifecycle errands cover progressively deeper layers of the stack.

## Errands

### 1. `gpu-validation-errand-noble` — Host-level GPU

Verifies the pre-baked driver on the stemcell and runs PyTorch/TensorFlow
benchmarks on the host. Confirms the driver and CUDA runtime work end-to-end.

```bash
bosh -d gpu-test deploy bosh/gpu-test-release/manifests/gpu-test-noble.yml
bosh -d gpu-test run-errand gpu-validation-errand-noble
```

Expected output:
```
✅ Tesla T4, 595.58.03, 15360 MiB
✅ PyTorch tests PASSED
✅ TensorFlow tests PASSED
✅ GPU VALIDATION PASSED
```

### 2. `container-gpu-errand-noble` — Container-level GPU (CDI)

Installs Docker and `nvidia-container-toolkit`, generates a CDI spec, and
runs `nvidia-smi` plus a small CUDA workload **inside a container** via
`--device=nvidia.com/gpu=all`. This is the prerequisite for Garden/Diego
integration.

```bash
bosh -d container-gpu-test deploy bosh/gpu-test-release/manifests/container-gpu-test-noble.yml
bosh -d container-gpu-test run-errand container-gpu-errand-noble
```

Expected output:
```
✅ Host driver: Tesla T4, 595.58.03
✅ nvidia-container-toolkit installed
✅ CDI spec generated at /etc/cdi/nvidia.yaml
✅ Container nvidia-smi succeeded
✅ Container CUDA workload OK
✅ CONTAINER GPU VALIDATION PASSED
```

## Stemcell

Uses a custom Ubuntu Noble stemcell with NVIDIA driver pre-baked:

```yaml
stemcells:
  - alias: default
    os: ubuntu-noble
    version: "1.562-nvidia"
```

Stemcell source: https://github.com/johha/bosh-linux-stemcell-builder/tree/nvidia-595-cuda12.9-v1.460

**Critical:** The stemcell already contains the NVIDIA driver. No BOSH packages or runtime compilation needed.

## Architecture

```
jobs/
├── gpu-validation-errand-noble/
│   ├── run            - Verifies driver, installs ML frameworks, runs tests
│   └── test-gpu.py    - PyTorch and TensorFlow validation tests
└── container-gpu-errand-noble/
    └── run            - Installs Docker + nvidia-container-toolkit, generates
                         CDI spec, runs nvidia-smi and a CUDA workload in a
                         container.
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

### VM Type

```yaml
instance_groups:
  - name: gpu-validation
    vm_type: gpu-small  # g4dn.xlarge with Tesla T4
```

## Troubleshooting

### "No GPU detected"

Wrong VM type or GPU not available. Verify `vm_type: gpu-small` points to a GPU instance.

Check: `bosh -d gpu-test ssh gpu-validation/0 -c "lspci | grep -i nvidia"`

### Driver not found after boot

The stemcell may not have the NVIDIA driver baked in correctly. Verify the stemcell version and check:

```bash
bosh -d gpu-test ssh gpu-validation/0 -c "nvidia-smi"
```

## Pre-Compiled Package Approach (Alternative)

Before the custom stemcell, this release used pre-compiled BOSH packages for driver distribution. The approach compiled driver artifacts for a specific stemcell kernel using `nvidia-compile-release`, then distributed them as BOSH blobs (~1 second install time).

This approach is preserved in git history (commit 43fd8ab) and remains valid if a custom stemcell is not available. Key steps:
1. Compile artifacts: `bosh -d nvidia-compile run-errand compile-nvidia`
2. Add as BOSH blobs: `bosh add-blob module.tar.gz nvidia-driver-570/module.tar.gz`
3. Use `gpu-validation-errand` job (with package dependencies)

The custom stemcell is the cleaner approach as it eliminates runtime driver installation entirely and works for any BOSH deployment, not just errands.
