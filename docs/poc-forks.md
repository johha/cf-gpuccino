# GPU Scheduling POC — Repos & Branches Reference

The GPU scheduling code changes for this POC live in forks of the upstream
Cloud Foundry components, not in this repository. This repo (`cf-gpuccino`)
is the **reference/docs repo**: it holds the design spec, implementation
plan, BBS demo client, and deploy/validation runbook.

The Diego-stack forks are under [`github.com/ZPascal`](https://github.com/ZPascal),
forked from the corresponding `cloudfoundry/*` upstream. The CAPI change lives on
a branch of the **upstream** `cloudfoundry/cloud_controller_ng` repo, and the GPU
stemcell on a fork of `bosh-linux-stemcell-builder`.

**Totals: 8 repos touched to make it work** — 6 ZPascal Diego-stack forks +
1 upstream CAPI branch + 1 stemcell-builder fork (plus this reference/docs repo).
10 feature branches, 2 open PRs, 0 merges into any `main`.

---

## Summary table

| Repo | Location | Branches | Open PRs |
|---|---|---|---|
| [bbs](https://github.com/ZPascal/bbs) | ZPascal fork | `gpu-poc`, `gpu-poc-db-persistence` | 1 |
| [rep](https://github.com/ZPascal/rep) | ZPascal fork | `gpu-poc` | 0 |
| [executor](https://github.com/ZPascal/executor) | ZPascal fork | `gpu-poc` | 0 |
| [garden](https://github.com/ZPascal/garden) | ZPascal fork | `gpu-poc` | 0 |
| [guardian](https://github.com/ZPascal/guardian) | ZPascal fork | `gpu-poc` | 0 |
| [auctioneer](https://github.com/ZPascal/auctioneer) | ZPascal fork | `gpu-poc`, `gpu-poc-fix-build` | 1 |
| [cloud_controller_ng](https://github.com/cloudfoundry/cloud_controller_ng/tree/gpu-flag) | upstream branch | `gpu-flag` | 0 |
| [bosh-linux-stemcell-builder](https://github.com/johha/bosh-linux-stemcell-builder/tree/nvidia-595-cuda12.9-v1.460) | fork | `nvidia-595-cuda12.9-v1.460` | 0 |
| cf-gpuccino | this repo (docs) | `integrate-gpu-support-cloudfoundry` | — |

---

## bbs — [github.com/ZPascal/bbs](https://github.com/ZPascal/bbs)

| Branch | Base | Status |
|---|---|---|
| `gpu-poc` | `main` | Pushed |
| `gpu-poc-db-persistence` | `gpu-poc` | Pushed — [PR #1 open](https://github.com/ZPascal/bbs/pull/1) → `gpu-poc` |

**`gpu-poc` commits:**
- `8d157cd` Add GPU request fields to DesiredLRPResource and DesiredLRP
- `c547c3f` Add GPUCapacity veto gate to CellState.ResourceMatch
- `7aafc46` Fix root module to use the in-tree patched bbs/models

**`gpu-poc-db-persistence` commits (on top of `gpu-poc`):**
- `cda4cb0` Persist GPU request to the desired_lrps table (Critical #2 follow-up)

## rep — [github.com/ZPascal/rep](https://github.com/ZPascal/rep)

| Branch | Base | Status |
|---|---|---|
| `gpu-poc` | `main` | Pushed |

**Commits:**
- `1fd159a` Populate CellState.GPUCapacity from executor state; carry GPU request through allocation

## executor — [github.com/ZPascal/executor](https://github.com/ZPascal/executor)

| Branch | Base | Status |
|---|---|---|
| `gpu-poc` | `main` | Pushed |

**Commits:**
- `9dadbfa` Add local go.mod and port GPUManager from cf-gpuccino
- `d0f7cf2` Add GPU fields to Resource and ExecutorResources
- `f8a99fd` Wire GPUManager into the container create/destroy lifecycle
- `4eea403` Fix depot.client.TotalResources dropping GPUTotal/GPUType (Critical #4)
- `f634107` Fix remaining final-review findings: CDI vendor fallback, CUDA UUIDs (#5, #6, #12, #13)

## garden — [github.com/ZPascal/garden](https://github.com/ZPascal/garden)

| Branch | Base | Status |
|---|---|---|
| `gpu-poc` | `main` | Pushed |

**Commits:**
- `e886364` Add CDIDevices field to ContainerSpec

## guardian — [github.com/ZPascal/guardian](https://github.com/ZPascal/guardian)

| Branch | Base | Status |
|---|---|---|
| `gpu-poc` | `main` | Pushed |

**Commits:**
- `6fd9ddb` Inject CDI devices client-side via tags.cncf.io/container-device-interface
- `459f460` Fix GPU POC final review issues #9, #15, #16

## auctioneer — [github.com/ZPascal/auctioneer](https://github.com/ZPascal/auctioneer)

> Not part of the original plan — forked mid-project after the final
> whole-branch review found it's the process that actually runs the GPU
> veto-gate scoring.

| Branch | Base | Status |
|---|---|---|
| `gpu-poc` | `main` | Pushed |
| `gpu-poc-fix-build` | `gpu-poc` | Pushed — [PR #1 open](https://github.com/ZPascal/auctioneer/pull/1) → `gpu-poc` |

**`gpu-poc` commits:**
- `c501cc2` Fix auctioneer to preserve GPU fields in resource requests

**`gpu-poc-fix-build` commits (on top of `gpu-poc`):**
- `da17ead` Fix cmd/auctioneer build break: remove UseV2API-gated V1 metric emitter

## bosh-linux-stemcell-builder — [github.com/johha/bosh-linux-stemcell-builder @ `nvidia-595-cuda12.9-v1.460`](https://github.com/johha/bosh-linux-stemcell-builder/tree/nvidia-595-cuda12.9-v1.460)

> Fork of the BOSH stemcell builder. Produces the custom Ubuntu Noble stemcell
> with the NVIDIA driver (595.58.03, CUDA 12.9) pre-baked, so GPU cells need zero
> runtime driver installation. See [stemcell options](stemcell-options.md).

| Branch | Base | Status |
|---|---|---|
| `nvidia-595-cuda12.9-v1.460` | upstream stemcell builder | Pushed |

## cloud_controller_ng (CAPI) — [cloudfoundry/cloud_controller_ng @ `gpu-flag`](https://github.com/cloudfoundry/cloud_controller_ng/tree/gpu-flag)

> Not a fork — a branch on the **upstream** `cloudfoundry/cloud_controller_ng`
> repo. Exposes GPU as a v3 **app feature flag** (`gpu`) rather than a raw
> process resource: a `gpu_enabled` column on apps, surfaced through the
> app-features API and app manifest, and passed into the Diego app recipe.

| Branch | Base | Status |
|---|---|---|
| `gpu-flag` | upstream `main` | Pushed |

**Commits:**
- `bfe97ff` add gpu feature flag

**What it touches:**
- `app/presenters/v3/app_gpu_feature_presenter.rb` (new)
- `app/controllers/v3/app_features_controller.rb`, `app/models/helpers/app_features.rb`
- `db/migrations/20260910120000_add_gpu_enabled_to_apps.rb` (new) — adds `gpu_enabled` to apps
- `lib/cloud_controller/diego/app_recipe_builder.rb` — carries the flag into the Diego recipe
- OpenAPI / v3 manifest docs + accompanying specs

## cf-gpuccino (this repo — reference/docs, not a fork)

Branch `integrate-gpu-support-cloudfoundry`. Contains the design spec,
implementation plan, Go BBS demo client, and deploy/validation runbook.

- `b8336fd` Add design spec for Diego/DiegoCell GPU integration POC
- `0190cba` Add implementation plan for Diego/DiegoCell GPU integration POC
- `bd01a9a`, `38bf46a` Guardian mechanism corrections (spec + plan)
- `f5a34e1` Add known gap: no GPU-allocation rollback on Garden Create failure
- `fa180c4` Add Go BBS client demo and README for GPU POC
- `5bec0e6` Add GPU POC deploy and validation runbook
- `bcf31c9` docs: fix GPU POC documentation issues from final review
- `89fdfcd`, `ec66396`, `64b32f7` Auctioneer discovery/fix documentation updates
