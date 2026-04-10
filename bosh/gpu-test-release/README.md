# GPU Test BOSH Release

A self-contained BOSH release for validating GPU support on BOSH-managed VMs.

## What It Does

**Lifecycle errand** that creates a GPU VM on-demand, validates GPU functionality, then destroys the VM.

**Tests Both Frameworks:**
- PyTorch 2.5.1+cu121
- TensorFlow 2.21.0

**3 Simple Tests:**
1. **GPU detection** - Verifies CUDA is available
2. **FP32 matrix multiplication** - Validates baseline compute (TFLOPS)
3. **FP16 matrix multiplication** - Validates Tensor Cores (speedup vs FP32)

**PoC Design:**
- No configuration needed (hardcoded values)
- Tests everything automatically
- Clean, repeatable validation
- **Cost-effective**: VM only exists during test execution (~5-10 minutes)

No manual SSH required.

## Architecture

This release uses **pure bash and Python scripts** without ERB templating:
- All scripts are self-contained bash/Python (no `.erb` files)
- No properties needed - hardcoded values for PoC
- Single lifecycle errand installs driver + runs tests + cleans up

**Lifecycle Errand:**
- `bosh deploy` creates deployment metadata (no VM)
- `bosh run-errand` creates VM → installs driver → runs tests → destroys VM
- You only pay for GPU instance during test execution

---

## Jobs

### gpu-validation-errand

Lifecycle errand that:
1. Installs NVIDIA driver 570 from Ubuntu graphics-drivers PPA
2. Installs Python 3.11 + PyTorch + TensorFlow in a temporary virtualenv
3. Runs 3 validation tests (GPU detection, FP32 matmul, FP16 matmul)
4. Reports results for both frameworks
5. Cleans up (temp virtualenv removed)

**No properties needed** - always tests both PyTorch and TensorFlow with standard settings (4096x4096 matrices, 10 iterations).

## Usage

### Prerequisites

1. BOSH director with GPU-enabled cloud config (vm_type with GPU instance and 30GB+ root disk)
2. Ubuntu Jammy stemcell (HVM variant for AWS)

### Deploy

```bash
# From the bbl-state directory
cd ~/SAPDevelop/ghtools/bbl-cantina-state/environments/han/bbl-state
eval "$(bbl print-env)"

# Create and upload the release
RELEASE_DIR=~/SAPDevelop/ghcom/cf-gpuccino/bosh/gpu-test-release
bosh create-release --dir=${RELEASE_DIR} --version=3.1.0 --force
bosh upload-release --dir=${RELEASE_DIR}

# Deploy (creates deployment metadata, no VM created)
bosh -d gpu-test deploy ${RELEASE_DIR}/manifests/gpu-test.yml

# Run validation errand (creates VM, installs driver, runs tests, destroys VM)
# Takes ~10-15 min: 5-7 min for driver install + 3-5 min for framework install/test
bosh -d gpu-test run-errand gpu-validation

# Optional: Keep VM alive for debugging
bosh -d gpu-test run-errand gpu-validation --keep-alive

# Run again anytime (fresh VM each time)
bosh -d gpu-test run-errand gpu-validation
```

### Check Results

```bash
# Errand output shows results directly in the console
bosh -d gpu-test run-errand gpu-validation

# If you used --keep-alive, you can SSH to check logs
bosh -d gpu-test ssh gpu-validation/0 -c "cat /var/vcap/sys/log/gpu-validation-errand/results-pytorch.json"
bosh -d gpu-test ssh gpu-validation/0 -c "cat /var/vcap/sys/log/gpu-validation-errand/results-tensorflow.json"

# View nvidia-smi output
bosh -d gpu-test ssh gpu-validation/0 -c "nvidia-smi"
```

### Cleanup

```bash
# Delete deployment (no cost impact since VM is already destroyed after errand)
bosh -d gpu-test delete-deployment

# If you used --keep-alive and the VM is still running:
bosh -d gpu-test stop gpu-validation  # Stops VM
bosh -d gpu-test delete-deployment    # Removes deployment
```

## Expected Results

On a Tesla T4 (g4dn.xlarge), both PyTorch and TensorFlow tests should pass:

**PyTorch:**
- FP32: ~4.2 TFLOPS
- FP16: ~41 TFLOPS (9-10x speedup via Tensor Cores)

**TensorFlow:**
- FP32: ~1.8 TFLOPS
- FP16: ~5.6 TFLOPS (3x speedup)

PyTorch generally shows better GPU performance, especially with Tensor Cores.

All 3 tests should report `"status": "PASSED"` for both frameworks.

## Cost

| Instance Type | GPU | Cost (eu-west-1) | Cost per Test Run |
|---------------|-----|------------------|-------------------|
| g4dn.xlarge | 1x T4 | ~$0.58/hour | ~$0.10-0.15 |
| g4dn.2xlarge | 1x T4 | ~$0.90/hour | ~$0.15-0.23 |
| g5.xlarge | 1x A10G | ~$1.00/hour | ~$0.17-0.25 |

**Lifecycle errand advantage**: VM only exists for ~10-15 minutes per test run, not 24/7. Each test costs ~$0.10-0.25 instead of $14/day for a persistent VM.

## Troubleshooting

**"No space left on device" during driver install:**
- Root disk too small. GPU vm_type needs 30GB+ root disk.
- Check cloud-config: `root_disk: { size: 30720, type: gp3 }`

**nvidia-smi not found:**
- Driver installation failed during errand execution
- Check errand output for error details
- Try running again: `bosh -d gpu-test run-errand gpu-validation`

**CUDA not available in PyTorch:**
- Rare. Driver module load may have failed
- Run errand again - fresh VM will get clean driver install

**First run takes 10-15 minutes:**
- Normal. Driver DKMS compile (5-7 min) + ML framework download (3-5 min)
- Subsequent runs take the same time since VM is destroyed and recreated each time

**Errand times out:**
- Default timeout may be too short for slow networks
- Increase with: `bosh -d gpu-test run-errand gpu-validation --download-logs`

## Cloud Config Requirements

The gpu-small vm_type needs a larger root disk for NVIDIA drivers:

```yaml
vm_types:
- name: gpu-small
  cloud_properties:
    instance_type: g4dn.xlarge
    root_disk:
      size: 30720  # 30GB - required for NVIDIA driver packages
      type: gp3
    ephemeral_disk:
      size: 51200
      type: gp3
```

See `gpu-ops.yml` in the bbl-state cloud-config directory.

