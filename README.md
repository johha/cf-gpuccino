# cf-gpuccino – GPU Support for Cloud Foundry

A reference/prototype implementation that extends Cloud Foundry to schedule, allocate, and expose GPU devices to application containers using CDI (Container Device Interface) and the existing Diego/Garden stack.

---

## Architecture Diagram

```
  Developer
     │
     │  cf push myapp --gpu 1
     ▼
 ┌──────────┐    REST     ┌──────────┐   BBS RPC  ┌─────────────┐
 │  CF CLI  │────────────▶│  CAPI    │────────────▶│     BBS     │
 └──────────┘             └──────────┘             └──────┬──────┘
                                                          │ LRP desired
                                                          ▼
                                                   ┌─────────────┐
                                                   │  Diego Auc- │
                                                   │  tioneer    │
                                                   └──────┬──────┘
                                                          │ BidForGPU
                                          ┌───────────────┴────────────────┐
                                          ▼                                ▼
                                   ┌─────────────┐                 ┌─────────────┐
                                   │  Diego Rep  │                 │  Diego Rep  │
                                   │ (GPU cell)  │                 │ (CPU cell)  │
                                   └──────┬──────┘                └─────────────┘
                                          │ allocate GPU
                                          ▼
                                   ┌─────────────┐
                                   │  Executor   │  GPUManager.Allocate()
                                   └──────┬──────┘
                                          │ ContainerSpec + CDIDevices
                                          ▼
                                   ┌─────────────┐
                                   │ garden-runc │  CDI injection → runc spec
                                   └──────┬──────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │    runc     │  /dev/nvidia0 in container
                                   └──────┬──────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │     App     │  CUDA_VISIBLE_DEVICES=0
                                   │  Container  │
                                   └─────────────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │  GPU Metrics│  → Loggregator → Log Cache
                                   │  Collector  │
                                   └─────────────┘
```

---

## Component Overview

| Layer | Component | Role in GPU Support |
|---|---|---|
| Kernel/CGroup | cgroups v1/v2 | Isolate GPU device access per container |
| Container Runtime | runc + CDI | Inject GPU devices into OCI bundle via CDI specs |
| Garden | garden-runc | Translate `CDIDevice` specs into runc container configs |
| Diego Rep | `rep/gpu_capacity.go` | Advertise GPU capacity; score bids in auction |
| Diego Executor | `executor/gpu_manager.go` | Discover, allocate, and release GPU devices |
| BBS | `bbs/models/gpu.proto` | Carry `GPURequest` in LRP/Task run-info |
| CAPI | `capi/` | Accept `gpu` field in process update API; persist to BBS |
| CF CLI | (standard CLI) | Pass `--gpu` / manifest `gpu: 1` through to CAPI |
| Monitoring | `monitoring/gpu_metrics.go` | Collect per-container GPU utilization; emit to Loggregator |
| BOSH | `bosh/` | Deploy GPU cells with nvidia-container-toolkit job |

---

## Quick Start – How the Pieces Fit Together

1. **Operator** deploys a GPU cell using the BOSH manifest in `bosh/manifests/gpu-cell.yml`.
   The `nvidia-toolkit` BOSH job installs nvidia-container-toolkit and writes CDI specs under `/var/vcap/data/cdi/specs/`.

2. **Operator** registers the CDI spec dir with garden-runc (via `--cdi-spec-dirs` flag).

3. **Developer** pushes an app with GPU resources:
   ```
   cf push myapp -f manifest.yml
   # manifest.yml contains:  resources: { gpu: 1, gpu_type: nvidia }
   ```

4. **CAPI** accepts the request via `PATCH /v3/processes/:guid` and stores `GPURequest` in the BBS desired LRP.

5. **Diego Auctioneer** calls each Rep's `/state` endpoint; Reps with GPUs report `gpu_capacity` and win the bid via `BidForGPU`.

6. **Diego Executor** calls `GPUManager.Allocate()`, obtains a GPU index, builds a `ContainerSpec` with `CDIDevices: [{Name: "nvidia.com/gpu=0"}]`, and passes it to garden-runc.

7. **garden-runc** expands the CDI device name into `/dev/nvidia0`, `/dev/nvidiactl`, `/dev/nvidia-uvm` entries and the required library mounts in the OCI bundle, then invokes `runc`.

8. **GPU Buildpack** (if used at staging) sets `CUDA_VISIBLE_DEVICES` and library paths in `.profile.d/gpu-env.sh`.

9. **Monitoring** collects `gpu_utilization` and `gpu_memory_used` metrics per container and forwards them to Loggregator as v2 envelopes.

---

## Repository Layout

```
cf-gpuccino/
├── README.md                          ← this file
├── go.mod
├── bbs/
│   └── models/
│       └── gpu.proto                  ← Protobuf: GPURequest in RunInfo
├── garden/
│   └── gpu.go                         ← CDI device specs for garden-runc
├── executor/
│   └── gpu_manager.go                 ← GPU discovery & allocation
├── rep/
│   └── gpu_capacity.go                ← Diego Rep auction GPU capacity
├── capi/
│   ├── models/
│   │   └── gpu_process.go             ← CF v3 API models
│   └── api/
│       └── gpu_handler.go             ← HTTP handlers
├── cdi/
│   └── specs/
│       ├── nvidia-gpu.json            ← CDI spec for NVIDIA GPUs
│       └── amd-gpu.json               ← CDI spec for AMD/ROCm GPUs
├── bosh/
│   ├── manifests/
│   │   └── gpu-cell.yml               ← BOSH manifest excerpt
│   └── jobs/
│       └── nvidia-toolkit/
│           ├── spec                   ← BOSH job spec
│           └── templates/
│               └── config.json.erb    ← nvidia-container-toolkit config
├── buildpack/
│   └── gpu-buildpack/
│       └── README.md
├── monitoring/
│   └── gpu_metrics.go                 ← GPU metrics → Loggregator
└── docs/
    ├── architecture.md
    └── deployment.md
```

---

## Further Reading

- [Architecture Details](docs/architecture.md)
- [Deployment Guide](docs/deployment.md)
- [GPU Buildpack](buildpack/gpu-buildpack/README.md)
- [CDI Specification](https://github.com/cncf-tags/container-device-interface)
- [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/index.html)
