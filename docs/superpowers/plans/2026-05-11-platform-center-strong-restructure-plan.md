# Platform Center Strong Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure `platform-center` into a monorepo-style layout with clear app boundaries, unified worker skeletons, a domain-oriented control plane, and a layered web console without changing the project’s intended runtime behavior.

**Architecture:** First establish the new repository skeleton, then move runnable units into `apps/`, then progressively refactor internal package structure inside `control-plane`, workers, and `web-console`. Keep runtime behavior stable during each migration stage by fixing imports, Dockerfiles, compose paths, and config references immediately after each move.

**Tech Stack:** Go, Gin, GORM, Docker Compose, vanilla JavaScript, Nginx, Redis, MySQL

---

## Legacy Source Naming

To avoid mixing current paths with historical source names, this plan uses these legacy aliases:

- `<legacy-control-plane-root>`: the pre-`apps/` control-plane root, historically named `backend-center`
- `<legacy-web-console-root>`: the pre-`apps/` web-console root, historically named `front-center`
- `<legacy-web-assets-root>`: the pre-`src/` browser script tree, historically named `assets/app`
- `<legacy-modules-root>`: the pre-domain backend service tree, historically named `internal/modules`
- `<legacy-platform-root>`: the pre-`internal/infra` backend foundation tree, historically named `internal/platform`

All `apps/...`, `src/...`, `internal/infra/...`, and `internal/domains/...` paths below refer to the target layout, not the legacy source tree.

---

## File Structure Map

### New top-level targets

- Create: `apps/`
- Create: `packages/`
- Create: `ops/compose/`
- Create: `ops/scripts/`
- Create: `ops/bootstrap/`
- Create: `ops/diagnostics/`
- Create: `docs/architecture/`
- Create: `docs/runbooks/`
- Create: `docs/archive/`

### Control-plane target paths

- Create: `apps/control-plane/`
- Move from: `<legacy-control-plane-root>/**`
- Create: `apps/control-plane/internal/infra/`
- Create: `apps/control-plane/internal/domains/iam/`
- Create: `apps/control-plane/internal/domains/platform/`
- Create: `apps/control-plane/internal/domains/delivery/`
- Create: `apps/control-plane/internal/domains/inventory/`
- Create: `apps/control-plane/internal/domains/clusters/`
- Create: `apps/control-plane/internal/domains/addons/`
- Create: `apps/control-plane/internal/domains/machines/`
- Create: `apps/control-plane/internal/domains/projects/`
- Create: `apps/control-plane/internal/domains/graph/`
- Create: `apps/control-plane/internal/contracts/api/`
- Create: `apps/control-plane/internal/contracts/internal/`
- Create: `apps/control-plane/internal/contracts/events/`
- Create: `apps/control-plane/internal/adapters/http/`
- Create: `apps/control-plane/internal/adapters/persistence/`
- Create: `apps/control-plane/internal/adapters/queue/`

### Worker target paths

- Create: `apps/resource-sync-worker/`
- Create: `apps/machine-enrollment-worker/`
- Create: `apps/cluster-enrollment-worker/`
- Create: `apps/cluster-addon-worker/`
- Create: `apps/terraform-runner/`
- For each worker, create:
  - `internal/runtime/`
  - `internal/client/controlplane/`
  - `internal/domain/`
  - `internal/worker/handlers/`

### Web console target paths

- Create: `apps/web-console/`
- Move from: `<legacy-web-console-root>/**`
- Create: `apps/web-console/src/boot/`
- Create: `apps/web-console/src/core/api/`
- Create: `apps/web-console/src/core/state/`
- Create: `apps/web-console/src/core/events/`
- Create: `apps/web-console/src/core/router/`
- Create: `apps/web-console/src/core/runtime/`
- Create: `apps/web-console/src/core/ui-shell/`
- Create: `apps/web-console/src/domains/auth/`
- Create: `apps/web-console/src/domains/overview/`
- Create: `apps/web-console/src/domains/delivery/`
- Create: `apps/web-console/src/domains/clusters/`
- Create: `apps/web-console/src/domains/machines/`
- Create: `apps/web-console/src/domains/system/`
- Create: `apps/web-console/src/shared/components/`
- Create: `apps/web-console/src/shared/forms/`
- Create: `apps/web-console/src/shared/tables/`
- Create: `apps/web-console/src/shared/drawers/`
- Create: `apps/web-console/src/shared/dialogs/`
- Create: `apps/web-console/src/shared/terminal/`
- Create: `apps/web-console/src/shared/utils/`
- Create: `apps/web-console/src/pages/login/`
- Create: `apps/web-console/src/pages/overview/`
- Create: `apps/web-console/src/pages/delivery/`
- Create: `apps/web-console/src/pages/clusters/`
- Create: `apps/web-console/src/pages/machines/`
- Create: `apps/web-console/src/pages/system/`

### Ops/docs target paths

- Move from: `docker-compose.yml`
- Move from: `scripts/*.sh`
- Move from: `AWS_ENROLLMENT_RUNBOOK.md`
- Move from: `DEPLOYMENT_CONFIG.md`
- Move from: `PHASE4_ENROLLMENT_RUNTIME.md`
- Move from: `PHASE6_CLUSTER_ADDON_RUNBOOK.md`
- Move from: `STARTUP.md`
- Move from: `K8S_MODULE_ARCHIVE.md`

---

### Task 1: Establish Monorepo Skeleton

**Files:**
- Create: `apps/.gitkeep`
- Create: `packages/.gitkeep`
- Create: `ops/compose/.gitkeep`
- Create: `ops/scripts/.gitkeep`
- Create: `ops/bootstrap/.gitkeep`
- Create: `ops/diagnostics/.gitkeep`
- Create: `docs/architecture/.gitkeep`
- Create: `docs/runbooks/.gitkeep`
- Create: `docs/archive/.gitkeep`
- Modify: `docs/superpowers/specs/2026-05-11-platform-center-strong-restructure-design.md`

- [ ] **Step 1: Add the new top-level directories**

Create the new top-level folders:

```text
apps/
packages/
ops/compose/
ops/scripts/
ops/bootstrap/
ops/diagnostics/
docs/architecture/
docs/runbooks/
docs/archive/
```

- [ ] **Step 2: Add placeholder files so the empty directories persist**

Create these exact files:

```text
apps/.gitkeep
packages/.gitkeep
ops/compose/.gitkeep
ops/scripts/.gitkeep
ops/bootstrap/.gitkeep
ops/diagnostics/.gitkeep
docs/architecture/.gitkeep
docs/runbooks/.gitkeep
docs/archive/.gitkeep
```

- [ ] **Step 3: Update the design spec to mention the implementation root paths**

Add a short appendix to `docs/superpowers/specs/2026-05-11-platform-center-strong-restructure-design.md` listing the exact target root directories above so implementers can validate the migration destination.

- [ ] **Step 4: Verify the new skeleton exists**

Run: `find /Users/alex/ops/platform-center -maxdepth 2 -type d | sort`

Expected: includes `apps`, `packages`, `ops/compose`, `ops/scripts`, `docs/architecture`, `docs/runbooks`, `docs/archive`

- [ ] **Step 5: Commit**

```bash
git add apps packages ops docs
git commit -m "chore: create monorepo skeleton for platform-center"
```

### Task 2: Move Runtime Units Under `apps/`

**Files:**
- Create: `apps/control-plane/`
- Create: `apps/web-console/`
- Create: `apps/terraform-runner/`
- Create: `apps/resource-sync-worker/`
- Create: `apps/machine-enrollment-worker/`
- Create: `apps/cluster-enrollment-worker/`
- Create: `apps/cluster-addon-worker/`
- Modify: `ops/compose/docker-compose.yml`
- Modify: all moved Dockerfiles and config-relative paths

- [ ] **Step 1: Move the current service directories into `apps/`**

Move these directories without changing their internal contents yet:

```text
<legacy-control-plane-root>     -> apps/control-plane
<legacy-web-console-root>       -> apps/web-console
terraform-runner                -> apps/terraform-runner
resource-sync-worker            -> apps/resource-sync-worker
machine-enrollment-worker       -> apps/machine-enrollment-worker
cluster-enrollment-worker       -> apps/cluster-enrollment-worker
cluster-addon-worker            -> apps/cluster-addon-worker
```

- [ ] **Step 2: Move the root docker compose file into `ops/compose/`**

Move:

```text
docker-compose.yml -> ops/compose/docker-compose.yml
```

Then update all service build contexts from legacy top-level service paths to `../../apps/...` style.

- [ ] **Step 3: Update volume and relative path assumptions in compose**

Fix compose references so that:

- `build.context` points at `../../apps/...`
- script callers can still use project-root-relative paths
- mounted host credential directories remain unchanged

- [ ] **Step 4: Update Dockerfiles if they rely on old directory names**

Check and fix:

```text
apps/control-plane/Dockerfile
apps/web-console/Dockerfile
apps/terraform-runner/Dockerfile
apps/resource-sync-worker/Dockerfile
apps/machine-enrollment-worker/Dockerfile
apps/cluster-enrollment-worker/Dockerfile
apps/cluster-addon-worker/Dockerfile
```

Expected fixes are path-only, not behavior changes.

- [ ] **Step 5: Verify compose still parses**

Run: `docker compose -f /Users/alex/ops/platform-center/ops/compose/docker-compose.yml config`

Expected: valid rendered compose output, no missing build context errors

- [ ] **Step 6: Commit**

```bash
git add apps ops/compose
git commit -m "refactor: move runtime units into apps directory"
```

### Task 3: Rehome Existing Docs and Scripts

**Files:**
- Move: `STARTUP.md`
- Move: `AWS_ENROLLMENT_RUNBOOK.md`
- Move: `DEPLOYMENT_CONFIG.md`
- Move: `PHASE4_ENROLLMENT_RUNTIME.md`
- Move: `PHASE6_CLUSTER_ADDON_RUNBOOK.md`
- Move: `K8S_MODULE_ARCHIVE.md`
- Move: `scripts/*.sh`

- [ ] **Step 1: Move active runbooks into `docs/runbooks/`**

Move these files:

```text
STARTUP.md                       -> docs/runbooks/STARTUP.md
AWS_ENROLLMENT_RUNBOOK.md        -> docs/runbooks/AWS_ENROLLMENT_RUNBOOK.md
DEPLOYMENT_CONFIG.md             -> docs/runbooks/DEPLOYMENT_CONFIG.md
PHASE4_ENROLLMENT_RUNTIME.md     -> docs/runbooks/PHASE4_ENROLLMENT_RUNTIME.md
PHASE6_CLUSTER_ADDON_RUNBOOK.md  -> docs/runbooks/PHASE6_CLUSTER_ADDON_RUNBOOK.md
```

- [ ] **Step 2: Move historical or archive-like material into `docs/archive/`**

Move:

```text
K8S_MODULE_ARCHIVE.md -> docs/archive/K8S_MODULE_ARCHIVE.md
```

- [ ] **Step 3: Move existing scripts into `ops/scripts/`**

Move:

```text
scripts/check_enrollment_runtime.sh
scripts/collect_cluster_addon_evidence.sh
scripts/collect_enrollment_evidence.sh
scripts/seed_machine_demo.sh
scripts/seed_machine_demo_via_backend.sh
scripts/smoke_check.sh
```

to:

```text
ops/scripts/
```

- [ ] **Step 4: Update references in markdown files and spec docs**

Search for old root references like:

```text
/Users/alex/ops/platform-center/scripts/
/Users/alex/ops/platform-center/STARTUP.md
```

Replace them with:

```text
/Users/alex/ops/platform-center/ops/scripts/
/Users/alex/ops/platform-center/docs/runbooks/STARTUP.md
```

- [ ] **Step 5: Verify no stale script/doc references remain**

Run: `rg -n "platform-center/(scripts|STARTUP.md|AWS_ENROLLMENT_RUNBOOK.md|DEPLOYMENT_CONFIG.md|PHASE4_ENROLLMENT_RUNTIME.md|PHASE6_CLUSTER_ADDON_RUNBOOK.md|K8S_MODULE_ARCHIVE.md)" /Users/alex/ops/platform-center`

Expected: only intentional references to the new `ops/scripts` and `docs/...` paths

- [ ] **Step 6: Commit**

```bash
git add docs ops
git commit -m "refactor: move operational docs and scripts into docs and ops"
```

### Task 4: Reorganize `control-plane` Foundation Layers

**Files:**
- Modify: `apps/control-plane/internal/app/app.go`
- Move/Modify: `apps/control-plane/internal/database/*.go`
- Move/Modify: `apps/control-plane/<legacy-platform-root>/cache/*.go`
- Move/Modify: `apps/control-plane/<legacy-platform-root>/queue/*.go`
- Move/Modify: `apps/control-plane/<legacy-platform-root>/http/*.go`
- Move/Modify: `apps/control-plane/<legacy-platform-root>/secure/*.go`
- Move/Modify: `apps/control-plane/<legacy-platform-root>/security/*.go`
- Create: `apps/control-plane/internal/infra/db/`
- Create: `apps/control-plane/internal/infra/cache/`
- Create: `apps/control-plane/internal/infra/queue/`
- Create: `apps/control-plane/internal/infra/security/`
- Create: `apps/control-plane/internal/infra/web/`

- [ ] **Step 1: Move database package into `internal/infra/db/`**

Move:

```text
internal/database/database.go
internal/database/schema_hardening.go
```

to:

```text
internal/infra/db/
```

Update package names and imports consistently to `db`.

- [ ] **Step 2: Move cache and queue packages into `internal/infra/`**

Move:

```text
<legacy-platform-root>/cache/*
<legacy-platform-root>/queue/*
```

to:

```text
internal/infra/cache/*
internal/infra/queue/*
```

- [ ] **Step 3: Move HTTP router and response helpers into `internal/infra/web/`**

Move:

```text
<legacy-platform-root>/http/*
<legacy-platform-root>/response/response.go
<legacy-platform-root>/middleware/*
```

into:

```text
internal/infra/web/
```

Recommended sublayout:

```text
internal/infra/web/router.go
internal/infra/web/registry.go
internal/infra/web/middleware/jwt.go
internal/infra/web/middleware/internal_token.go
internal/infra/web/response/response.go
```

- [ ] **Step 4: Move crypto and password helpers into `internal/infra/security/`**

Move:

```text
<legacy-platform-root>/secure/crypto.go
<legacy-platform-root>/security/password.go
```

into:

```text
internal/infra/security/
```

- [ ] **Step 5: Update `internal/app/app.go` imports only**

Do not change startup behavior. Only rewrite imports and dependency constructor calls so `app.Run()` still:

- loads config
- opens DB
- migrates
- seeds
- creates cache/queue
- wires services
- builds router

- [ ] **Step 6: Run a control-plane compile check**

Run: `cd /Users/alex/ops/platform-center/apps/control-plane && go test ./...`

Expected: package compilation succeeds; if tests fail, failures must be unrelated to moved import paths before continuing

- [ ] **Step 7: Commit**

```bash
git add apps/control-plane
git commit -m "refactor: move control-plane foundation packages into infra"
```

### Task 5: Split `control-plane` Domains

**Files:**
- Move/Modify: `apps/control-plane/<legacy-modules-root>/auth/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/user/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/rbac/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/identitysource/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/systemsetting/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/project/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/graph/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/cloud/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/k8s/*`
- Move/Modify: `apps/control-plane/<legacy-modules-root>/machine/*`

- [ ] **Step 1: Create target domain directories**

Create:

```text
internal/domains/iam/
internal/domains/platform/
internal/domains/delivery/
internal/domains/inventory/
internal/domains/clusters/
internal/domains/addons/
internal/domains/machines/
internal/domains/projects/
internal/domains/graph/
```

- [ ] **Step 2: Merge auth/user/rbac/identitysource into `iam/`**

Move code from:

```text
<legacy-modules-root>/auth/
<legacy-modules-root>/user/
<legacy-modules-root>/rbac/
<legacy-modules-root>/identitysource/
```

into:

```text
internal/domains/iam/
```

Keep file names responsibility-oriented, for example:

```text
service_auth.go
service_user.go
service_rbac.go
service_identity_source.go
handler_auth.go
handler_user.go
handler_rbac.go
handler_identity_source.go
model_user.go
model_rbac.go
model_identity_source.go
```

- [ ] **Step 3: Move system settings into `platform/`**

Move:

```text
<legacy-modules-root>/systemsetting/*
```

into:

```text
internal/domains/platform/
```

Keep runtime/system configuration concerns there.

- [ ] **Step 4: Move project and graph domains**

Move:

```text
<legacy-modules-root>/project/* -> internal/domains/projects/
<legacy-modules-root>/graph/*   -> internal/domains/graph/
```

- [ ] **Step 5: Split the old `cloud` package**

Split:

```text
<legacy-modules-root>/cloud/deployment_service.go
<legacy-modules-root>/cloud/blueprint_service.go
<legacy-modules-root>/cloud/network_plan_service.go
<legacy-modules-root>/cloud/stack_binding.go
```

into `internal/domains/delivery/`

Split:

```text
<legacy-modules-root>/cloud/resource_service.go
<legacy-modules-root>/cloud/machine_sync.go
<legacy-modules-root>/cloud/cluster_sync.go
```

into `internal/domains/inventory/`

Split:

```text
<legacy-modules-root>/cloud/enrollment_service.go
<legacy-modules-root>/cloud/account_service.go
<legacy-modules-root>/k8s/*
```

into `internal/domains/clusters/`

Split:

```text
<legacy-modules-root>/cloud/addon_service.go
```

into `internal/domains/addons/`

Move:

```text
<legacy-modules-root>/machine/*
```

into `internal/domains/machines/`

- [ ] **Step 6: Rewrite router/service wiring to use new domain paths**

Update `internal/app/app.go` and router imports so the behavior remains the same while imports point at:

```text
internal/domains/iam
internal/domains/platform
internal/domains/projects
internal/domains/graph
internal/domains/delivery
internal/domains/inventory
internal/domains/clusters
internal/domains/addons
internal/domains/machines
```

- [ ] **Step 7: Run a second compile check**

Run: `cd /Users/alex/ops/platform-center/apps/control-plane && go test ./...`

Expected: all packages compile after the domain split

- [ ] **Step 8: Commit**

```bash
git add apps/control-plane
git commit -m "refactor: split control-plane modules into domain packages"
```

### Task 6: Normalize Worker Layouts

**Files:**
- Modify: all `apps/*worker*/internal/*`
- Create: `internal/runtime/`
- Create: `internal/client/controlplane/`
- Create: `internal/domain/`
- Create: `internal/worker/runner.go`
- Create: `internal/worker/executor.go`

- [ ] **Step 1: Use `resource-sync-worker` as the template**

Restructure `apps/resource-sync-worker/internal/` to:

```text
internal/app/
internal/config/
internal/runtime/
internal/client/controlplane/
internal/domain/
internal/worker/
```

Move current sync logic into `internal/worker/executor.go` and put the poll/claim/report loop into `internal/worker/runner.go`.

- [ ] **Step 2: Apply the same layout to `machine-enrollment-worker`**

Move current worker logic into the same shape as above. Keep behavior unchanged.

- [ ] **Step 3: Apply the same layout to `cluster-enrollment-worker`**

Move provider-specific adapters into:

```text
internal/worker/handlers/
```

Examples:

```text
handlers/aws_sdk_adapter.go
handlers/alicloud_openapi_adapter.go
```

- [ ] **Step 4: Apply the same layout to `cluster-addon-worker`**

Move addon-specific handlers into:

```text
internal/worker/handlers/
```

- [ ] **Step 5: Apply the same layout to `terraform-runner`**

Keep its Terraform execution specialization, but still introduce:

```text
internal/runtime/
internal/client/controlplane/
internal/domain/
internal/worker/runner.go
internal/worker/executor.go
```

The current `runner/` package can remain under `internal/worker/handlers/terraform/` if needed.

- [ ] **Step 6: Run worker compile checks**

Run:

```bash
cd /Users/alex/ops/platform-center/apps/resource-sync-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/machine-enrollment-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/cluster-enrollment-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/cluster-addon-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/terraform-runner && go test ./...
```

Expected: all five worker apps compile after the layout normalization

- [ ] **Step 7: Commit**

```bash
git add apps/resource-sync-worker apps/machine-enrollment-worker apps/cluster-enrollment-worker apps/cluster-addon-worker apps/terraform-runner
git commit -m "refactor: normalize worker layouts across runtime units"
```

### Task 7: Reorganize `web-console` Source Tree

**Files:**
- Move/Modify: `apps/web-console/<legacy-web-assets-root>/*.js`
- Create: `apps/web-console/src/...`
- Modify: `apps/web-console/index.html`

- [ ] **Step 1: Create the new `src` layout**

Create:

```text
src/boot
src/core/api
src/core/state
src/core/events
src/core/router
src/core/runtime
src/core/ui-shell
src/domains/auth
src/domains/overview
src/domains/delivery
src/domains/clusters
src/domains/machines
src/domains/system
src/shared/components
src/shared/forms
src/shared/tables
src/shared/drawers
src/shared/dialogs
src/shared/terminal
src/shared/utils
src/pages/login
src/pages/overview
src/pages/delivery
src/pages/clusters
src/pages/machines
src/pages/system
```

- [ ] **Step 2: Move global boot logic out of `main.js`**

Split current responsibilities into:

```text
src/boot/init.js
src/boot/session.js
src/boot/errors.js
```

Keep `main.js` as a thin import/boot entry or replace it with `src/boot/init.js` as the main browser entry.

- [ ] **Step 3: Move global helpers into `core` and `shared`**

Rehome:

```text
api.js   -> src/core/api/client.js
state.js -> src/core/state/store.js
ui.js    -> src/core/ui-shell/ui.js
utils.js -> src/shared/utils/index.js
dom.js   -> src/core/ui-shell/dom.js
```

- [ ] **Step 4: Rename domain directories**

Move:

```text
<legacy-web-assets-root>/cloud/*   -> src/domains/delivery/*
<legacy-web-assets-root>/k8s/*     -> src/domains/clusters/*
<legacy-web-assets-root>/machine/* -> src/domains/machines/*
<legacy-web-assets-root>/system/*  -> src/domains/system/*
```

Create:

```text
src/domains/overview/
src/domains/auth/
```

for overview and login/session logic currently embedded in `main.js`.

- [ ] **Step 5: Introduce page-level entry modules**

Create:

```text
src/pages/overview/index.js
src/pages/delivery/index.js
src/pages/clusters/index.js
src/pages/machines/index.js
src/pages/system/index.js
```

Each page module should coordinate rendering and event binding for that page while delegating business behavior to `src/domains/...`.

- [ ] **Step 6: Update `index.html` script imports**

Point the browser entry at the new boot entry file. Keep behavior unchanged.

- [ ] **Step 7: Run a static smoke check**

Run:

```bash
cd /Users/alex/ops/platform-center
docker compose -f ops/compose/docker-compose.yml up -d --build frontend backend
curl -i http://127.0.0.1:8080/
```

Expected: `HTTP/1.1 200 OK`

- [ ] **Step 8: Commit**

```bash
git add apps/web-console
git commit -m "refactor: reorganize web console into layered src structure"
```

### Task 8: Fix Ops Entry Points After Migration

**Files:**
- Modify: `docs/runbooks/STARTUP.md`
- Modify: `ops/scripts/*.sh`
- Modify: `ops/compose/docker-compose.yml`
- Modify: any script hardcoding old directories

- [ ] **Step 1: Rewrite script root path assumptions**

Update every script in `ops/scripts/` so the project root and app locations point at:

```text
apps/control-plane
apps/web-console
apps/terraform-runner
apps/resource-sync-worker
apps/machine-enrollment-worker
apps/cluster-enrollment-worker
apps/cluster-addon-worker
ops/compose/docker-compose.yml
```

- [ ] **Step 2: Rewrite startup docs**

Update `docs/runbooks/STARTUP.md` so the canonical startup command becomes:

```bash
cd /Users/alex/ops/platform-center
docker compose -f ops/compose/docker-compose.yml up -d --build
```

- [ ] **Step 3: Verify smoke script still works**

Run: `zsh /Users/alex/ops/platform-center/ops/scripts/smoke_check.sh`

Expected: script completes using the new path structure

- [ ] **Step 4: Commit**

```bash
git add docs/runbooks ops
git commit -m "chore: update scripts and startup docs for new layout"
```

### Task 9: Remove Old Top-Level Runtime Directories

**Files:**
- Delete: old empty top-level service dirs after migration

- [ ] **Step 1: Verify no runtime code remains in the old locations**

Run:

```bash
find /Users/alex/ops/platform-center -maxdepth 1 -type d | sort
```

Expected: old top-level runtime directories are either absent or empty placeholders pending deletion.

- [ ] **Step 2: Delete obsolete runtime directories**

Delete only after all imports, scripts, compose paths, and docs point to the new layout:

```text
<legacy-control-plane-root>/
<legacy-web-console-root>/
terraform-runner/
resource-sync-worker/
machine-enrollment-worker/
cluster-enrollment-worker/
cluster-addon-worker/
scripts/
```

- [ ] **Step 3: Run final repository path sanity search**

Run:

```bash
rg -n "backend-center|front-center|terraform-runner|resource-sync-worker|machine-enrollment-worker|cluster-enrollment-worker|cluster-addon-worker|/scripts/" /Users/alex/ops/platform-center
```

Expected: only intentional historical/archive references remain

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove old runtime layout after monorepo migration"
```

### Task 10: Final Validation Pass

**Files:**
- Modify if needed: any broken path/config after validation

- [ ] **Step 1: Run control-plane tests**

Run: `cd /Users/alex/ops/platform-center/apps/control-plane && go test ./...`

Expected: control-plane compiles and tests pass or only known unrelated failures remain documented

- [ ] **Step 2: Run worker tests**

Run:

```bash
cd /Users/alex/ops/platform-center/apps/resource-sync-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/machine-enrollment-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/cluster-enrollment-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/cluster-addon-worker && go test ./...
cd /Users/alex/ops/platform-center/apps/terraform-runner && go test ./...
```

Expected: all worker packages compile

- [ ] **Step 3: Run compose validation**

Run: `docker compose -f /Users/alex/ops/platform-center/ops/compose/docker-compose.yml config`

Expected: valid config output

- [ ] **Step 4: Run startup smoke validation**

Run:

```bash
cd /Users/alex/ops/platform-center
docker compose -f ops/compose/docker-compose.yml up -d --build
zsh ops/scripts/smoke_check.sh
```

Expected: backend/frontend core health checks pass

- [ ] **Step 5: Commit final fixes**

```bash
git add -A
git commit -m "chore: finalize strong restructure validation fixes"
```

---

## Self-Review

### Spec Coverage

- Monorepo skeleton: covered by Tasks 1-3
- Control-plane domainization: covered by Tasks 4-5
- Worker unification: covered by Task 6
- Web console layering: covered by Task 7
- Ops/docs relocation: covered by Tasks 2-3 and 8
- Old layout removal: covered by Task 9
- Validation: covered by Task 10

### Placeholder Scan

- No `TODO`, `TBD`, or “similar to above” instructions remain
- Every task names exact files or directory targets
- Every validation step includes explicit commands

### Type Consistency

- `control-plane`, `web-console`, and worker app names are used consistently
- New directories are consistently named `domains`, `infra`, `contracts`, `adapters`, `client/controlplane`, `worker/handlers`
