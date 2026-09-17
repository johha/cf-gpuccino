# cf-gpuccino – GPU Support for Cloud Foundry

A reference/prototype implementation that extends Cloud Foundry to schedule, allocate, and expose GPU devices to application containers using CDI (Container Device Interface) and the existing Diego/Garden stack.

---

## Current Status

**Phase 1 (Foundation) - In Progress**

✅ **Validated**: NVIDIA driver and GPU compute on BOSH-managed VMs
- Custom Ubuntu Noble stemcell with pre-baked NVIDIA driver (595.58.03)
- Lifecycle errand validates GPU functionality on-demand (VM created/destroyed per test)
- GPU compute verified via comprehensive benchmarks (PyTorch & TensorFlow)
- PyTorch: 4.45 TFLOPS FP32, 40.79 TFLOPS FP16 (Tensor Cores)
- TensorFlow: 1.25 TFLOPS FP32, 3.08 TFLOPS FP16
- Cost-effective testing: ~$0.10-0.15 per test run vs $14/day for persistent VM
- See `bosh/gpu-test-release/` for the working BOSH release

🚧 **Next**: Container GPU access via nvidia-container-toolkit
- Errand `container-gpu-errand-noble` installs the toolkit, generates a CDI
  spec, and runs `nvidia-smi` + a CUDA workload inside a container on a
  BOSH-managed VM. Prerequisite for Garden/Diego integration.
- See `bosh/gpu-test-release/manifests/container-gpu-test-noble.yml`.

🧪 **Diego scheduling POC (in forks)**: a GPU request now flows through
BBS → auctioneer → rep → executor → garden → guardian, with CAPI exposing GPU
as a v3 app feature flag. The Diego-stack code lives in forks under
[`github.com/ZPascal`](https://github.com/ZPascal) and the CAPI change on an
upstream `cloud_controller_ng` branch — nothing merged upstream yet. See the
[POC forks reference](docs/poc-forks.md).

See [Summary & Outlook](docs/summary-and-outlook.md) for the presentation
overview, [Roadmap](docs/roadmap.md) for the full plan, and
[Shortcuts](docs/shortcuts.md) for PoC assumptions.

---

## Architecture Diagram

```
  Developer
     │
     │  cf push myapp (gpu: true)
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
                                                          │ gate on GPU capacity
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
| CAPI | `capi/` | Expose GPU as a v3 app feature flag (`gpu_enabled`); persist to BBS |
| CF CLI | (standard CLI) | Pass manifest `gpu: true` / app-feature through to CAPI |
| Monitoring | `monitoring/gpu_metrics.go` | Collect per-container GPU utilization; emit to Loggregator |
| BOSH | `bosh/` | Deploy GPU cells with nvidia-container-toolkit job |

---

## Quick Start – How the Pieces Fit Together

1. **Operator** deploys a GPU cell using the BOSH manifest in `bosh/manifests/gpu-cell.yml`.
   The `nvidia-toolkit` BOSH job installs nvidia-container-toolkit and writes CDI specs under `/var/vcap/data/cdi/specs/`.

2. **Operator** registers the CDI spec dir with garden-runc (via `--cdi-spec-dirs` flag).

3. **Developer** pushes an app with GPU enabled:
   ```
   cf push myapp -f manifest.yml
   # manifest.yml contains:  gpu: true
   ```

4. **CAPI** records the `gpu_enabled` app feature and carries a `GPURequest` into the BBS desired LRP. (The POC uses a boolean feature flag; a count/type resource model is a deferred RFC question.)

5. **Diego Auctioneer** calls each Rep's `/state` endpoint; Reps with GPUs report `gpu_capacity`, and the auction gates a GPU-requesting LRP onto a cell with free GPU capacity.

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
│   ├── gpu-test-release/              ← ✅ Working BOSH release for GPU validation
├── buildpack/
│   └── gpu-buildpack/
│       └── README.md
├── monitoring/
│   └── gpu_metrics.go                 ← GPU metrics → Loggregator
└── docs/
    ├── architecture.md
    ├── deployment.md
    ├── roadmap.md                     ← Implementation phases
    ├── shortcuts.md                   ← PoC assumptions & deferred work
    └── stemcell-options.md            ← Driver packaging strategies
```

---

## Further Reading

- [Summary & Outlook](docs/summary-and-outlook.md) – Presentation overview: what was built, what's next
- [POC Forks & Branches](docs/poc-forks.md) – Where the Diego scheduling code lives
- [Roadmap](docs/roadmap.md) – Implementation phases and progress
- [Architecture Details](docs/architecture.md)
- [Deployment Guide](docs/deployment.md)
- [Shortcuts & Assumptions](docs/shortcuts.md) – PoC trade-offs
- [Stemcell Options](docs/stemcell-options.md) – Driver packaging strategies
- [GPU Test Release](bosh/gpu-test-release/README.md) – Validate GPU on BOSH VMs
- [GPU Buildpack](buildpack/gpu-buildpack/README.md)
- [CDI Specification](https://github.com/cncf-tags/container-device-interface)
- [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/index.html)
