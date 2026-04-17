# cf-gpuccino Roadmap

This roadmap describes the planned milestones for bringing full GPU support to
Cloud Foundry.  Items are grouped into phases by logical dependency, not by
calendar time.

---

## Phase 1 – Foundation: Infrastructure & Host-Level Enablement

The groundwork that every other phase depends on.

- [x] **GPU test BOSH release** – validate NVIDIA driver installation on BOSH-managed
  VMs using Ubuntu Jammy stemcell. Proves driver + kernel compatibility.
  See `bosh/gpu-test-release/`.
- [x] **NVIDIA driver compilation release** – pre-compiled driver packages via
  `nvidia-compile-release`. Compiles kernel modules for specific stemcell kernel,
  packages as BOSH blobs. See `bosh/nvidia-compile-release/`.
- [x] **Pre-compiled driver packages** – gpu-test-release uses pre-compiled
  BOSH packages for fast driver installation (~1 second vs 5-7 min DKMS).
  Artifacts tracked with git-lfs in `bosh/driver-artifacts/`.
- [x] **nvidia-container-toolkit package** – packaged as BOSH blob, ready for
  GPU cell deployment.
- [ ] **BOSH stemcell with NVIDIA driver (production)** – for production workloads,
  produce a Ubuntu Jammy stemcell variant with pre-baked NVIDIA kernel module
  and user-space driver. Eliminates runtime driver install entirely.
  (Alternative approach validated with pre-compiled packages)
- [ ] **`nvidia-toolkit` BOSH job** – install and configure
  `nvidia-container-toolkit` on GPU cells; enable CDI mode by default.
- [ ] **`nvidia-persistenced` BOSH job** – keep driver state alive between
  container starts (required for MIG / multi-GPU setups).
- [ ] **`nvidia-fabricmanager` BOSH job** – multi-GPU NVLink fabric management
  (needed for A100/H100 nodes).
- [ ] **`gpu-cell` instance group** – dedicated BOSH instance group using
  GPU-capable IaaS flavours (`p3.xlarge`, `a2-highgpu-*`, …) with
  `placement_tags: [gpu]`.
- [ ] **AMD/ROCm variant** – parallel job for ROCm toolkit and `/dev/kfd` setup.
- [ ] **CDI spec generation** – auto-generate `/var/vcap/data/cdi/specs/*.json`
  for both NVIDIA and AMD at BOSH deploy time.

---

## Phase 2 – Core Scheduling: Diego BBS & Cell Rep

Teach the Diego scheduler about GPU resources.

- [ ] **BBS Protobuf extensions** – add `gpu_limit`, `gpu_type`, and
  `gpu_indices` fields to `RunInfo`, `DesiredLRP`, and `ActualLRP`.
- [ ] **Cell Rep – GPU inventory** – discover GPUs at cell startup via
  `nvidia-smi` / NVML and advertise `TotalResources.GPUs` to the BBS.
- [ ] **Auctioneer – GPU dimension** – include GPU count in the scoring
  and placement algorithm alongside CPU/RAM/disk.
- [ ] **Rep bid validation** – `BidForLRP` / `BidForTask` checks free GPU
  capacity before accepting a work unit.
- [ ] **BBS migration** – database schema change + data migration for the
  new LRP fields.

---

## Phase 3 – Container Runtime: Executor, Garden & OCI

Wire GPU requests through to the actual container.

- [ ] **Diego Executor – GPU manager** – `Allocate` / `Release` logic that
  pins specific GPU indices to containers; persists state across executor
  restarts.
- [ ] **Garden API extension** – add `CDIDevices []string` and
  `Devices []DeviceSpec` to `garden.ContainerSpec`.
- [ ] **garden-runc propagation** – translate `ContainerSpec.CDIDevices` into
  OCI `config.json` entries (`linux.devices`, `linux.resources.devices`,
  CDI annotations) before handing off to runc.
- [ ] **runc CDI integration** – verify runc ≥ 1.1 `--cdi-spec-dirs` flag is
  passed; fall back to legacy OCI-hook approach for older runc versions.
- [ ] **cgroup v1 device allow-list** – write `devices.allow` entries
  (`c 195:* rwm`, `c 507:* rwm`) when CDI is unavailable.
- [ ] **cgroup v2 / eBPF device filter** – attach
  `BPF_PROG_TYPE_CGROUP_DEVICE` programme per sandbox when running on a
  v2-only kernel.

---

## Phase 4 – CF API: CAPI & Quotas

Expose GPU resources through the Cloud Controller API.

- [ ] **CAPI DB migration** – add `gpu` (integer) and `gpu_type` (string)
  columns to the `processes` table.
- [ ] **`ProcessModel` extension** – surface new fields in the CAPI Ruby
  process model and update `ProcessUpdateMessage`.
- [ ] **v3 API endpoints** – accept and return `gpu` / `gpu_type` fields in
  `POST /v3/apps/:guid/processes`, `PATCH /v3/processes/:guid`, and
  `GET /v3/apps/:guid`.
- [ ] **CAPI → Diego conversion** – translate GPU fields in
  `DesiredLRPFromApp` / `app_runner` when building a `DesiredLRP`.
- [ ] **GPU quota dimension** – add `gpu` to Org and Space quota definitions;
  enforce at app push and scale time.
- [ ] **Resource summary endpoint** – include `gpu_usage` in
  `GET /v3/organizations/:guid/usage_summary`.

---

## Phase 5 – CF CLI & App Manifests

Developer-facing tooling.

- [ ] **App manifest schema** – support `resources.gpu` and
  `resources.gpu_type` keys in `manifest.yml`.
- [ ] **`cf push` parsing** – CF CLI reads GPU manifest fields and sends them
  to CAPI.
- [ ] **`cf scale` flag** – `--gpu <n>` flag for `cf scale`.
- [ ] **`cf app` display** – show GPU allocation in the app details table.
- [ ] **`cf create-quota` / `cf update-quota`** – `--gpu <n>` flag for quota
  management commands.

---

## Phase 6 – Buildpack & Runtime Libraries

Ensure GPU libraries are available inside the container.

- [ ] **CDI bind-mount (preferred)** – validate that host driver libraries are
  correctly bind-mounted via CDI before finalising Phase 3.
- [ ] **`gpu-buildpack` – detect** – detect `.gpu-enabled` marker or
  `requirements-gpu.txt` in the app source.
- [ ] **`gpu-buildpack` – compile** – write `.profile.d/gpu-env.sh` exporting
  `CUDA_VISIBLE_DEVICES`, `LD_LIBRARY_PATH`, and `HIP_VISIBLE_DEVICES`.
- [ ] **`gpu-buildpack` – health check** – optional GPU smoke-test script run
  at container start.
- [ ] **Buildpack registry** – publish the buildpack to the CF buildpack
  registry / `cf create-buildpack`.

---

## Phase 7 – Monitoring, Metrics & Observability

Make GPU usage visible to operators and app developers.

- [ ] **NVML metrics collection** – collect per-GPU utilisation and memory
  figures from each allocated container (using `go-nvml`).
- [ ] **Loggregator v2 gauge envelopes** – emit `gpu_utilization`,
  `gpu_memory_used`, and `gpu_memory_total` as tagged gauge envelopes with
  `source_id` = app GUID.
- [ ] **Log Cache integration** – verify envelopes are queryable via
  `cf log-cache <app> --envelope-type gauge`.
- [ ] **BOSH health checks** – report GPU device health (`nvidia-smi` exit
  code) as a BOSH VM health indicator.
- [ ] **Stratos / CF dashboard** – optional: surface GPU metrics in the
  Stratos web UI.

---

## Phase 8 – Hardening, Security & Multi-tenancy

Production-readiness improvements.

- [ ] **GPU isolation audit** – verify no cross-container device access is
  possible via cgroup / eBPF rules.
- [ ] **MIG (Multi-Instance GPU) support** – slice A100/H100 GPUs into
  isolated MIG instances; model each MIG slice as a schedulable resource unit.
- [ ] **Time-slicing support** – allow multiple containers to share one GPU
  via NVIDIA time-slicing configuration.
- [ ] **Operator documentation** – GPU cell setup guide, stemcell requirements,
  CDI troubleshooting runbook.
- [ ] **Integration test suite** – end-to-end tests: `cf push` GPU app → Diego
  schedules on GPU cell → NVML metrics appear in Log Cache.
- [ ] **Security review** – penetration test of device-access boundaries;
  address any findings.

---

## Backlog / Future Considerations

Items not yet scoped into a specific phase.

- Kubernetes-bridge (KubeCF) GPU support via device-plugin API.
- GPU sharing / fractional GPU (e.g. NVIDIA MPS).
- InfiniBand / RDMA for GPU cluster networking.
- Windows GPU containers (DirectX / WDDM path).
- GPU-aware placement constraints (GPU model, VRAM size) in `cf push`.
