# Deployment Guide – cf-gpuccino

## Prerequisites

### Host requirements (GPU cells)

| Requirement | Minimum version | Notes |
|---|---|---|
| NVIDIA GPU driver | 525.x | Must be pre-installed in the BOSH stemcell image, or installed via a post-start script before garden starts |
| nvidia-container-toolkit | 1.14.x | Installed by the `nvidia-toolkit` BOSH job |
| runc | 1.1.0 | Included in garden-runc; CDI support requires ≥ 1.1.0 |
| Linux kernel | 5.4 (CGroup v1) / 5.10 (CGroup v2) | Ubuntu Jammy stemcell recommended |

### BOSH requirements

- BOSH Director with access to a GPU-capable IaaS instance type (e.g. `p3.xlarge` on AWS, `n1-standard-4` + `nvidia-tesla-v100` on GCP).
- A BOSH stemcell with NVIDIA drivers pre-installed, **or** a post-start script that installs them. Ubuntu Jammy (`ubuntu-jammy`) works well; the NVIDIA driver must be installed before the BOSH agent starts the garden job.
- The `cf-gpuccino` BOSH release (containing the `nvidia-toolkit` job) uploaded to the Director.

---

## BOSH Deployment Steps

### 1. Upload the cf-gpuccino release

```bash
bosh create-release --final --tarball cf-gpuccino-1.0.0.tgz
bosh -e my-env upload-release cf-gpuccino-1.0.0.tgz
```

### 2. Prepare the GPU stemcell

Use a stemcell that includes NVIDIA drivers, or add a `pre-start` script to the `nvidia-toolkit` job that runs the NVIDIA driver installer:

```bash
bosh -e my-env upload-stemcell \
  https://bosh.io/d/stemcells/bosh-aws-xen-hvm-ubuntu-jammy-go_agent?v=1.260
```

### 3. Deploy the GPU cell instance group

```bash
bosh -e my-env -d cf deploy cf-deployment.yml \
  -o bosh/manifests/gpu-cell.yml \
  -v gpu_cell_count=2
```

The `gpu-cell.yml` ops-file adds the `gpu-cell` instance group to your existing CF deployment.

### 4. Verify the deployment

```bash
bosh -e my-env -d cf instances --ps | grep gpu-cell
```

---

## Verifying GPU Availability on Cells

SSH to a GPU cell and confirm the NVIDIA driver and CDI setup:

```bash
bosh -e my-env -d cf ssh gpu-cell/0

# On the cell VM:
nvidia-smi                          # verify GPU is visible
ls /var/vcap/data/cdi/specs/        # CDI spec files should be present
cat /var/vcap/data/cdi/specs/nvidia-gpu.json

# Check garden-runc started with CDI support:
grep cdi /var/vcap/sys/log/garden/garden.stdout.log
```

Expected nvidia-smi output:

```
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 525.89.02    Driver Version: 525.89.02    CUDA Version: 12.0     |
|-------------------------------+----------------------+----------------------+
| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
|   0  Tesla V100-SXM2...  Off  | 00000000:00:1E.0 Off |                    0 |
```

---

## cf push Example with GPU Resources

### App manifest (`manifest.yml`)

```yaml
applications:
  - name: gpu-app
    buildpacks:
      - https://github.com/johha/cf-gpuccino/buildpack/gpu-buildpack
      - python_buildpack
    memory: 4G
    disk_quota: 2G
    instances: 1
    resources:
      gpu: 1
      gpu_type: nvidia
    env:
      PYTORCH_CUDA_ALLOC_CONF: max_split_size_mb:512
```

### Push the app

```bash
touch .gpu-enabled          # or create requirements-gpu.txt
cf push gpu-app -f manifest.yml
```

### Verify GPU access inside the container

```bash
cf ssh gpu-app -c "nvidia-smi"
cf ssh gpu-app -c "python -c 'import torch; print(torch.cuda.is_available())'"
```

### View GPU metrics

```bash
cf log-cache gpu-app --envelope-type gauge | grep gpu_utilization
```

---

## Troubleshooting

### `nvidia-smi` not found inside the container

The CDI spec mounts `/usr/bin/nvidia-smi` into the container.  Check that:

1. The CDI spec file is present: `ls /var/vcap/data/cdi/specs/`.
2. garden-runc was started with `--cdi-spec-dirs=/var/vcap/data/cdi/specs`.
3. runc version supports CDI: `runc --version` (need ≥ 1.1.0).

### Container fails to start with "insufficient free GPUs"

The `GPUManager.Allocate` call failed.  Check:

```bash
# On the cell:
cat /var/vcap/sys/log/rep/rep.stdout.log | grep -i gpu
```

Common causes:
- All GPUs are already allocated to other containers.
- `nvidia-smi` failed during GPUManager initialisation (check `dmesg` for driver errors).

### CDI spec not picked up

```bash
# Validate CDI spec JSON:
cat /var/vcap/data/cdi/specs/nvidia-gpu.json | python3 -m json.tool

# Check runc CDI resolution (run from container's bundle dir):
runc spec --cdi-device nvidia.com/gpu=0
```

### App sees `CUDA_VISIBLE_DEVICES=NoDevFiles`

This means the executor did not inject the CDI device.  Confirm the `GPURequest` reached the executor by checking BBS:

```bash
cfdot desired-lrp-scheduling-infos | jq '.[] | select(.process_guid == "YOUR-APP-GUID") | .run_info.gpu_request'
```
