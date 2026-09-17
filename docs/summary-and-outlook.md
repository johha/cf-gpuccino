# GPU Support for Cloud Foundry — Summary & Outlook

> Presentation companion. This document backs the **summary** and **outlook**
> slides: what we set out to do, what actually works today, what we learned,
> and what's left.

---

## The question

Cloud Foundry has stayed CPU-only. Kubernetes solved GPU scheduling years ago.
We wanted to know what it would actually take to schedule, allocate, and expose
GPU devices to CF application containers — and how much of the platform it
touches.

The short answer: almost every layer. GPU support reaches into the BOSH
stemcell, the NVIDIA driver and Container Device Interface (CDI), garden-runc
and the OCI spec, Diego's auctioneer and cell rep, the BBS, the CAPI process
model, and finally the CLI and app manifest.

---

## Summary — what we built

### 1. Host-level GPU enablement (validated end-to-end)

- **Custom Ubuntu Noble stemcell** with a pre-baked NVIDIA driver (595.58.03).
  Zero runtime driver installation.
- **GPU compute proven on real hardware** (Tesla T4) via a BOSH lifecycle errand
  that creates a VM, runs benchmarks, and tears the VM down per test run
  (~$0.10–0.15 per run vs. $14/day for a persistent VM):
  - PyTorch: **4.45 TFLOPS FP32**, **40.79 TFLOPS FP16** (Tensor Cores)
  - TensorFlow: **1.25 TFLOPS FP32**, **3.08 TFLOPS FP16**
- **Container GPU access** via `nvidia-container-toolkit`: an errand installs the
  toolkit, generates a CDI spec, and runs `nvidia-smi` + a CUDA workload inside a
  container on a BOSH-managed VM.

### 2. GPU scheduling through the Diego stack (POC code in forks)

A GPU request now flows through the scheduler end-to-end. The code lives in
forks under `github.com/ZPascal` (see [POC forks reference](poc-forks.md)):

| Layer | What changed | Fork |
|---|---|---|
| **BBS** | GPU request fields on `DesiredLRP`; GPU-capacity veto gate in `CellState.ResourceMatch`; DB persistence | [bbs](https://github.com/ZPascal/bbs) |
| **Rep** | Populates `CellState.GPUCapacity` from executor state; carries the GPU request through allocation | [rep](https://github.com/ZPascal/rep) |
| **Executor** | `GPUManager` for discovery/allocation, wired into the container create/destroy lifecycle; GPU fields on `Resource`/`ExecutorResources` | [executor](https://github.com/ZPascal/executor) |
| **Garden** | `CDIDevices` field added to `ContainerSpec` | [garden](https://github.com/ZPascal/garden) |
| **Guardian** | Injects CDI devices client-side via `tags.cncf.io/container-device-interface` | [guardian](https://github.com/ZPascal/guardian) |
| **Auctioneer** | Preserves GPU fields in resource requests — this is the process that actually runs the veto-gate scoring | [auctioneer](https://github.com/ZPascal/auctioneer) |
| **CAPI** | GPU exposed as a v3 **app feature flag** (`gpu`): `gpu_enabled` column, app-features API + manifest, carried into the Diego recipe | [cloud_controller_ng @ `gpu-flag`](https://github.com/cloudfoundry/cloud_controller_ng/tree/gpu-flag) (upstream branch) |

### 3. Reference & docs (this repo)

Design spec, implementation plan, a Go BBS demo client, and a deploy/validation
runbook — the shared understanding the fork work was built against.

---

## What surprised us

- **The auctioneer wasn't in the original plan.** We only discovered mid-project,
  during the final whole-branch review, that the auctioneer — not the rep — is the
  process that actually runs the GPU veto-gate scoring. It had to be forked too.
- **The blast radius is real.** A single "schedule a GPU" request required
  coordinated changes across six separate repos and their `go.mod` graphs.
- **Known gap found in review:** there is no GPU-allocation rollback if
  `Garden Create` fails — an allocated GPU index can leak on that error path.

---

## Status at a glance

- ✅ Host driver + GPU compute validated on BOSH VMs (real Tesla T4 hardware)
- ✅ Container GPU access via CDI validated on a BOSH VM
- ✅ GPU request path implemented through BBS → auctioneer → rep → executor →
  garden → guardian (POC branches, 2 open PRs, **0 merged to any `main`**)
- ✅ CAPI: GPU exposed as a v3 app feature flag on an upstream
  `cloud_controller_ng` branch (`gpu-flag`)
- 🚧 Not yet wired: CF CLI, app manifest end-to-end, quotas, monitoring
- 🚧 Not yet: promotion of the toolkit errand to a long-running cell job

---

## Outlook — what's next

### From POC to community process: RFCs

In Cloud Foundry, larger changes are agreed through **RFCs** in the
[community repo](https://github.com/cloudfoundry/community/tree/main/toc/rfc)
*before* code is merged. GPU support is big enough that we think it's **too much
for a single RFC** — it spans several working groups, and the community precedent
for changes this broad is to split along working-group seams and link the pieces
via the RFC `Related RFCs` field (as the Cloud Native Buildpacks work did across
RFC-0017 → 0028 → 0031).

> A single-RFC route does have precedent — the IPv6 dual-stack RFC (rfc-0038)
> touches almost the same component set in one internally-phased document. We're
> proposing to split instead, so each working group can own and merge its part
> independently.

Our proposed RFC family — an umbrella that sets direction, plus two cross-cutting
follow-up RFCs grouped **by concern** (not one per working group), so each follow-up
deliberately spans the groups that must agree on the interfaces between them:

| RFC | Working Group(s) | Scope |
|---|---|---|
| **Umbrella: GPU workloads in CF** | cross-WG | Shared goal, CDI-based approach, scope, non-goals, and open questions; names the two follow-ups |
| **A — GPU-capable cells: host, scheduling & deployment** | Foundational Infrastructure + App Runtime Platform + App Runtime Deployments | GPU stemcell / driver + `nvidia-container-toolkit` + CDI-spec generation; BBS GPU fields, auctioneer/rep scoring, executor `GPUManager`, garden/guardian CDI injection, GPU metrics; GPU cell instance group in cf-deployment |
| **B — Requesting & consuming GPUs: API, CLI & staging** | App Runtime Interfaces + Buildpacks | GPU request model (see open question below), app manifest + `cf` CLI, quotas/usage/metering, GPU buildpack |

RFC A is the "make the platform able to run a GPU workload" story end to end; RFC B is
the developer contract on top, depending on A. **The follow-up RFCs are opened once the
umbrella is merged**, so the community agrees on direction before design work fans out.
Each can be phased internally (proof-of-concept → implementation checkpoint → rollout).
The POC branches described above are the evidence base that makes these RFCs concrete
rather than speculative.

**Key open question — the GPU resource model.** How does a developer *request* a
GPU? The POC uses a simple **feature flag** (`gpu_enabled`, on/off) rather than a
count like `gpu: 2`, on purpose: a raw count reads awkwardly next to instance
count (is it per-instance or total?). By analogy, CF doesn't ask developers for
CPU cores directly — they're derived from requested memory, so GPU may want a
similar derived or bundled model. The umbrella RFC should *name* this decision
and lay out the options (feature flag vs. count vs. derived); settling it belongs
to the API-layer RFC with community input.

### Engineering work behind the RFCs

Grouped by dependency, not calendar. Full detail in the [roadmap](roadmap.md).

**Near term — close the scheduling loop**
- Merge the open PRs (`bbs` DB-persistence, `auctioneer` build fix) into the
  `gpu-poc` branches, consolidating the reference implementation.
- Promote the `nvidia-container-toolkit` errand to a long-running `gpu-cell`
  BOSH job with `placement_tags: [gpu]`.
- Fix the GPU-allocation rollback gap on the `Garden Create` failure path.

**Mid term — make it usable by developers**
- CAPI: extend the `gpu-flag` branch (feature flag → quotas, resource summary).
- CF CLI + app manifest: expose the chosen GPU request model, plus display in `cf app`.
- Monitoring: per-container NVML metrics → Loggregator → Log Cache.

**Longer term — production hardening**
- MIG (Multi-Instance GPU) and time-slicing as schedulable units.
- AMD/ROCm variant.
- Isolation audit + security review of device-access boundaries.
- End-to-end integration test suite.

---

## The honest close

This is a POC. It proves the path works: a GPU request can travel from a desired
LRP all the way to `/dev/nvidia0` inside a running container, on BOSH-managed
hardware. Nothing is merged upstream yet, and the developer-facing half (CLI,
manifest, quotas, metrics) is still mostly design-only — though CAPI already has
a first cut as an app feature flag. The next real step is community: taking this
to the RFC family above, where the open questions — the right resource model,
multi-tenancy isolation, fractional-GPU sharing, and whether the community wants
GPU in CF at all — get decided. That conversation is what the talk is here to
start.

---

## Related docs

- [POC forks & branches reference](poc-forks.md)
- [Roadmap](roadmap.md)
- [Architecture](architecture.md)
- [Deployment guide](deployment.md)
- [Shortcuts & assumptions](shortcuts.md)
- [CF Summit abstract](cf-summit-abstract.md)
