# Changelog

All notable changes to Prism will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **`.golangci.yml` migrated to the golangci-lint v2 schema.** The config
  declared `version: 2` but used v1-schema keys (`linters-settings`,
  `issues.exclude-rules`, `run.skip-dirs`, `output.format`), which golangci-lint
  v2 silently rejected and replaced with defaults — so `make lint` was running
  with the wrong linter set (cyclop threshold 10 instead of 15, test-file
  exclusions ignored). Rewrote to the valid v2 layout (`linters.settings`,
  `linters.exclusions.rules/paths`, `default: standard`); `golangci-lint config
  verify` now passes and the intended thresholds/exclusions apply. Note:
  `make lint` is a local developer tool (CI gates on `go vet` + `gofmt`, not
  golangci-lint); with the config now honored it surfaces pre-existing lint debt
  to be addressed separately.

### Changed
- **On-demand instance pricing now uses truffle (spawn adoption Phase 3d).** The
  on-demand compute rate recorded on every instance (`Instance.HourlyRate`) is
  now sourced from `github.com/spore-host/truffle` via a new
  `OnDemandPricingClient` (`pkg/aws/ondemand_pricing.go`), mirroring the spot
  path shipped in #659. This completes the truffle pricing consolidation:
  truffle wraps the AWS Price List API with a 24h cache and a static fallback and
  is **region-aware**, replacing Prism's hand-rolled Price List query whose
  static fallback table was us-east-1-only — so per-region estimates (e.g.
  us-west-2) are now accurate rather than us-east-1 approximations. The static
  `getHourlyRate` estimate is retained purely as the on-truffle-error fallback,
  so `HourlyRate` is never left at 0 on the list/refresh path. The old on-demand
  Price List code (`GetInstanceHourlyRate`/`queryPricingAPI`/`parsePriceFromJSON`
  + its 24h cache) is removed; `PricingClient` now holds only EBS volume rates
  (truffle has no storage pricing). Spot/on-demand rate selection
  (`EffectiveComputeRate`) is unchanged. Verified against real AWS (t3.large
  us-west-2 → the correct regional list price).

### Added
- **spawn adoption — opt-in launch capabilities (Phase 3a).** Three new opt-in
  `prism workspace launch` flags, all default-off (empty/false = today's
  behavior): `--spot-max-price <$/hr>` (a real spot bid cap; requires `--spot`),
  `--efa` (attach an Elastic Fabric Adapter NIC for MPI/HPC), and
  `--placement-group <name>` (launch into a cluster placement group). They thread
  through `types.LaunchRequest` → `launch.ToLaunchConfig` → the EC2 launcher
  (`configToRunInput` builds the EFA NIC and placement group, mirroring spawn so
  the eventual launch swap stays behavior-identical). Validation:
  `--spot-max-price` without `--spot` is rejected, and `--efa` on a non-EFA
  instance type is rejected via a new `supportsEFA` capability allowlist
  (modeled on `supportsHibernation`). CLI-only for now (the GUI launch form
  already omits optional launch fields from the wire); GUI exposure is a
  follow-up. Third step of the spawn-adoption milestone.
- **spawn adoption — instance lifecycle now runs on spawn (Phase 2).** Instance
  stop/start/hibernate/terminate now route through `github.com/spore-host/spawn`'s
  client via a `hybridLauncher` (`pkg/aws/hybrid_launcher.go`): lifecycle → spawn,
  launch → Prism's existing EC2 path. Behavior is preserved — spawn's `StopInstance`
  maps hibernate directly, its errors are `%w`-wrapped so the graceful "already
  gone" delete still fires, and Prism's transient-failure retry is kept by wrapping
  the spawn calls in `WithRetry`. Launch deliberately stays on Prism's launcher for
  now: spawn v0.65.0 hardcodes the root block-device name to `/dev/xvda` and doesn't
  derive it from the AMI, which would misplace the root-volume-size override on
  Prism's Ubuntu/Rocky AMIs (root `/dev/sda1`); launch moves to spawn once that gap
  is closed upstream. Validated end-to-end against real AWS
  (`test/integration/spawn_lifecycle_test.go`: launch → stop → start → hibernate
  → resume → terminate on a live Ubuntu 24.04 instance). Second step of the
  spawn-adoption milestone.
- **spawn adoption — launch/lifecycle seam (Phase 1).** Introduced a narrow, host-owned
  `Launcher` port (`pkg/launch`) expressed in `github.com/spore-host/spawn`'s own types
  (`Launch(ctx, spawn.LaunchConfig) (*spawn.LaunchResult, error)` + stop/start/terminate), plus a
  pure `ToLaunchConfig` mapping. Prism's EC2 launch and lifecycle
  (`Manager.LaunchInstance`/`Stop`/`Start`/`Hibernate`/`Delete`) now flow through this port,
  defaulting to an EC2-backed implementation (`pkg/aws/ec2_launcher.go`) that reproduces today's
  behavior byte-for-byte — public-IP NIC, gp3 root volume, spot/hibernation options, AZ failover.
  This is the seam-only refactor: no behavior change and the CLI/GUI wire contracts and the daemon
  budget launch-gate are untouched; a later phase swaps the default for `*spawn.Client` (which
  already satisfies the port). Pins `spore-host/spawn v0.65.0` (transitively bumps
  `scttfrdmn/substrate` v0.70.0 → v0.71.0). First step of the spawn-adoption milestone, following
  the truffle adoption (#659).
- **Live budget enforcement — auto-actions fire again (#656).** Per-project budget auto-actions
  (`AutoActions[]`: threshold %% → hibernate/stop/prevent-launch/notify) and the safety cushion
  (headroom → one action) are evaluated against the `budgetengine` spend ledger on the daemon's
  monitoring tick and executed when breached — restoring behavior that went dead when the legacy
  BudgetTracker was retired. A new daemon `budgetEnforcer` (`pkg/daemon/budget_enforcer.go`) folds each
  budgeted project's ledger via the same `budgetStatus` the API returns, so enforcement and the status
  readout agree; it drives the existing `Server.ExecuteHibernateAll/StopAll/PreventLaunch` executors
  and the surviving `CushionEvaluator`, and sends notifications through the alert dispatcher
  (Slack webhook when `PRISM_SLACK_WEBHOOK` is set, else log). **Off by default** — destructive actions
  require an explicit opt-in via `budget_enforcement_enabled` (or `PRISM_BUDGET_ENFORCEMENT=true`);
  the evaluation cadence is `budget_enforcement_interval_minutes` (default 10, 1-minute floor).
  Dedup is fire-once-per-budget-period via `LastTriggered` timestamps (persisted on each action and the
  cushion), so a breach acts once per month rather than every tick.

### Changed
- **Spend estimate uses the real spot price for spot instances (#659).** The spend observer's compute
  estimate previously used `Instance.HourlyRate` (on-demand list price) for every instance, badly
  over-stating spot workloads (spot is billed 60–90% below list). Prism now captures the real spot
  rate once per spot instance — via **`github.com/spore-host/truffle`** (prism's first spore.host
  dependency; `HourlyRate(…, "spot")` → min current spot across AZs) — persists it as
  `Instance.SpotHourlyRate`, and a new `Instance.EffectiveComputeRate()` selects it over the on-demand
  rate. Capture is lazy in the instance refresh builder (once, then preserved across refreshes) and
  fails soft to the on-demand rate on any lookup error, so accrual is never blocked. This makes the
  ledger estimate accurate for the common HPC/burst case without needing daily Cost Explorer
  reconciliation. Account-level discounts/credits and Savings Plans/RI remain reconciliation's job.
- **Real banking/rollover surplus + ledger-backed forecast complete the budget-engine adoption
  (Phase 3e, #655; closes #645).** `BudgetStatus.Surplus` is now populated for periodic
  (monthly/weekly/daily) budgets — via the existing `SurplusCalculator.ComputeSurplusWithRollover`
  over the real ledger cost series, honoring the project's `RolloverEnabled`/`RolloverCap`
  (project-lifetime budgets stay nil, as they don't bank across periods). The budget forecast moves
  daemon-side (`pkg/daemon/forecast.go`) and runs the `BudgetPredictor` over the **real** ledger
  series instead of the empty history the retired `Manager.GetProjectForecast` used, so
  `CurrentMonthlyRate`/`ForecastData`/exhaustion projections reflect actual spend. Multi-month grant
  budgets now pace their status read with **`BankAndReserve` × `RateTargetPolicy`** (hold the plan
  rate, bank underspend) while simple cloud budgets keep `RateAdjust` × `DeadlineFloatPolicy`; the
  launch gate is unaffected (it never invokes pacing). Requires **budgetengine v0.3.0**, which makes
  `BankAndReserve` genuinely distinct from `RateAdjust` (previously identical). `Manager` sheds its
  forecast stub, staying ledger-free.
- **Cost breakdown / resource usage / trends / monthly report now aggregate the real spend ledger
  (Phase 3d, #654).** The four cost-analytics surfaces that returned zeros after Phase 3c are computed
  from the `budgetengine` spend ledger in a new `pkg/daemon/cost_aggregation.go`. Everything derives
  from one primitive, `ledgerCostSeries`, a day-bucketed fold over the project's events: `estimate`
  events (which carry `ResourceID` + un-blended `Compute`/`Storage`, #652) attribute per-instance
  per-day; `billed`/`billed-reversal` reconciliation events (no `ResourceID`) fold into the day's
  project total only. From that series: `GET /projects/{id}/costs` returns a populated
  `ProjectCostBreakdown` (per-instance compute/storage + running/stopped hours from `StateHistory`,
  synthesized per-instance root-volume storage lines); `GET /projects/{id}/usage` returns real
  `ProjectResourceUsage` (active/total counts, compute hours, idle savings); the monthly report and
  budget-history endpoints render the real series; and `BudgetStatus.BurnRate` is now populated (via
  the existing `BurnRateCalculator`). **`GET /cost/trends`** now emits the `{date, spent, budget}`
  shape the CLI history table actually renders (the pre-3c stub emitted `CostDataPoint`, which the CLI
  never parsed, so `prism budget history` was always blank). The Manager's `GetProjectCostBreakdown`/
  `GetProjectResourceUsage` stubs are removed — the daemon owns cost aggregation, keeping `pkg/project`
  ledger-free. `BudgetStatus.Surplus` stays nil pending banking/rollover (Phase 3e, #655).
- **Budget status is now derived from the spend ledger, not the retired tracker (Phase 3c, #653).**
  The daemon computes `BudgetStatus` (`pkg/daemon/budget_status.go`) directly from the `budgetengine`
  spend ledger + the project's embedded budget plan: `SpentAmount` is the ledger fold (falling back to
  the project's cached `SpentAmount` only while the ledger is empty), and `RemainingBudget` /
  `SpentPercentage` / `ProjectedMonthlySpend` / days-until-exhausted derive from that and the engine's
  `BurnState`. The launch gate and the status readout now fold the **same** events — there is no
  longer a parallel budget model. `pkg/project.Manager` sheds all budget concerns and returns to pure
  project identity/lifecycle (its `budgetTracker`, `CheckBudgetStatus`, and `SetAlerter` are removed;
  `NewManagerForScope`/`NewManagerWithStore` drop the tracker argument). Cost-history / trends /
  monthly-report endpoints keep their response shape but return an empty series until the
  ledger-derived aggregates land in Phase 3d (#654). `cost.BudgetTrackerAdapter` is renamed
  `cost.FuncCostDataProvider` (closure-backed, now fed by the project manager + ledger).
- **Spend ledger records per-resource, per-component line items (#652).** The spend observer now
  populates the v0.2.0 `SpendEvent` line-item fields: each event carries `ResourceID` (instance id)
  and an **un-blended** split of `Compute` and `Storage` (Network unmodeled) that sums to `Amount`.
  Compute and storage accrue on **independent clocks** — compute on running-hours (it stops when the
  instance stops), storage on total-hours at the standard EBS rate (it persists and bills while
  stopped). Consequently the observer now also accrues for stopped-but-storage-bearing instances, not
  only running ones. This gives the ledger the detail Phase 3d aggregates into the cost-breakdown /
  resource-usage / report DTOs. Reconciliation now tracks estimate and billed baselines in their own
  units so the two never conflate.

### Removed
- **`pkg/project.BudgetTracker` and its `budget_data.json` persistence (Phase 3c, #653).** The legacy
  budget tracker — a parallel spend/analytics model that was hollow in production (its
  `UpdateProjectCosts`/`UpdateProjectSpending` had no prod callers and its auto-action path was
  inert) — is deleted in favor of the single `budgetengine` ledger. No working behavior is lost;
  live budget enforcement via the engine `ActionSink` is tracked separately (#656).

### Added
- **Opt-in Cost Explorer billed-cost reconciliation (#644).** The budget spend ledger can now
  reconcile its list-price estimates against authoritative AWS Cost Explorer billed cost, so a
  project's recorded spend converges on real dollars (e.g. reflecting RIs/Savings Plans/spot). It is
  **off by default** — CE calls cost money (~$0.01/call, 1–2 per instance), are rate-limited, and lag
  ~a day — and enabled via `daemon_config.json` (`cost_reconciliation_enabled`) or
  `PRISM_COST_RECONCILIATION=true`. The frequency is configurable
  (`cost_reconciliation_interval_minutes`, default 1440 = daily; or `PRISM_COST_RECONCILIATION_INTERVAL`
  as a Go duration) and clamped to a **1-hour floor**. When the per-instance `prism:instance-id`
  cost-allocation tag isn't active in AWS Billing, reconciliation is skipped (the region-total blob
  isn't per-instance) and the estimate is kept, with a warning to activate the tag. Reconciliation is
  additionally skipped in test/reduced mode; the reversal+billed correction is idempotent.
- **Budget engine adoption, Phase 2 (#644) — real spend ledger.** Prism now accrues per-instance
  spend into an append-only, seam-backed `budgetengine` ledger, replacing Phase 1's synthetic
  single spend event. A spend observer (`pkg/daemon/spend_observer.go`) runs on the `StateMonitor`
  tick, internally throttled to a 10-minute accrual cadence, and appends a `SpendEvent` per running
  project-attributed instance as the **delta** of a cumulative list-price estimate
  (`HourlyRate × runtime`). A per-instance checkpoint (persisted through the seam) makes accrual
  idempotent across restarts — no double-counting. The launch gate now reads this real ledger as
  its `SpendSource` when populated, falling back to the cached `SpentAmount` while the ledger warms
  up (so it never becomes more permissive than Phase 1). Spend is project-scoped for now
  (`AllocationID = "project"`). Authoritative Cost Explorer reconciliation and per-funding-allocation
  attribution are deferred to follow-ups; `BudgetTracker`/HTTP handlers are untouched.

### Changed
- **Budget engine adoption, Phase 1 (#643).** The launch-time monthly-budget gate
  (`Server.isLaunchBlockedByBudget`) now makes its projected-over-limit decision via the standalone
  `github.com/scttfrdmn/budgetengine` library (`engine.CheckLaunch`) instead of an inline
  comparison. A new `pkg/daemon/budget_engine.go` adapter feeds the engine a single-source,
  read-only view of the project's existing embedded budget — **no change to budget storage, the
  spend model, or the HTTP API.** Block/allow behavior is a verified parity of the previous
  flat-limit check (`budget_engine_test.go`); fail-open on engine error is preserved. Depends on
  `github.com/scttfrdmn/budgetengine v0.1.0` (a public, zero-dependency module). Later phases move
  spend to an event ledger (#644) and retire the JSON tracker (#645).

### Changed
- **Bump Wails v3 `v3.0.0-alpha.74` → `v3.0.0-alpha.102`** in `cmd/prism-gui`. No application code
  changes required; the bump also drops a large set of transitive dependencies (Wails alpha.102 no
  longer pulls in `go-git`, `ProtonMail/go-crypto`, `cloudflare/circl`, `x/net`, etc.) and swaps
  `wailsapp/go-webview2` → `wailsapp/wails/webview2`. Verified: both modules build, `go vet` and
  `govulncheck` clean, and the GUI launches and initializes the Wails runtime end-to-end on
  alpha.102 (system tray + AssetServer serving the React app).

### Fixed
- GUI tests `TestGetInstanceAccess` and `TestEmbeddedWebView` no longer panic with a nil-pointer
  dereference. They constructed a `PrismService` with only `daemonURL`/`client` set, but the code
  had since moved to the typed `apiClient` (`s.apiClient.GetInstance`), so the calls dereferenced a
  nil client. Rewritten to inject a seeded `pkg/api/mock` client and assert current behavior (SSH +
  web access derived from the instance's services); the stale assertions for removed RDP/desktop
  port detection were dropped.

### Security
- Bump the Go directive to **1.26.5** in both modules (and pin the CI/release workflows to 1.26.5),
  clearing **GO-2026-5856** — an Encrypted Client Hello privacy leak in the `crypto/tls` standard
  library reachable from the daemon's HTTPS server, the web proxy, and the API client. `govulncheck`
  now reports zero affecting vulnerabilities under go1.26.5.

## [0.36.0] - 2026-07-07

### Added
- `prism workspace cost --billed [name]` reports the AWS-billed ("billed so far") cost from AWS Cost Explorer next to prism's local estimate and the delta between them. prism's existing cost numbers are a model the daemon accumulates from observed state transitions (`Instance.CurrentSpend`); the billed figure is the metered amount AWS has actually charged, read via `ce:GetCostAndUsage` using the UnblendedCost metric. This surfaces drift between the two --- for example, the same workspace controlled from two machines can report different local estimates while AWS bills one number. Per-instance isolation uses the `prism:instance-id` cost-allocation tag when it is activated in the Billing console; when the tag is inactive, the command falls back to the region's EC2 total and warns that it may include other instances. `--billed-only` shows just the AWS figure. New daemon endpoint `GET /api/v1/instances/{name}/billed-cost`. Requires the daemon's AWS identity to hold the `ce:GetCostAndUsage` permission; access-denied errors include the fix.
- **The persistence seam (`pkg/seam`)** — the shared record/persistence + identity interface
  Prism's governance managers converge onto (the prism-research-portal design §5). `Principal` /
  `Scope` (tenant/project/pi/grant + account) and a generic, scoped `Store[T]` repository, with a
  file-backed implementation (`pkg/seam/filestore`, desktop-standalone) and a conformance suite.
  The contract is byte-identical to prism-research-portal's seam — same `Principal.Key()`, same
  record shapes — so a record one client writes is the record the other reads (shared state, no
  sync protocol). Prism remains fully usable standalone (file-backed); the same managers back the
  shared cloud (DynamoDB-backed) without changing their logic.
- **DynamoDB-backed seam store (`pkg/seam/dynamostore`)** — the shared cloud-core implementation,
  single-table (`pk = scope.Key()`, `sk = id`, `body = JSON`), encoding byte-identical to
  prism-research-portal's. This is what lets a desktop Prism pointed at the cloud table see prp's
  writes and vice versa. Passes the same conformance suite as the file-backed store (driven by an
  in-memory fake of the four DynamoDB calls — no live AWS, no new dependency).

### Changed
- **`ApprovalManager` now persists through the seam** instead of inlining `os.ReadFile` /
  `os.WriteFile` against `~/.prism/approvals.json`. Public API unchanged; all existing behavior
  and tests preserved. A legacy `approvals.json` is migrated into the seam on first run (then
  retired). This is the first manager converged; project/budget/rbac follow.
- **`ProjectManager` now persists through the seam** as one record per project (rather than a
  single `projects.json` rewritten on every mutation) — the granularity shared state needs, so
  Prism and prp editing different projects don't clobber each other. The in-memory map and all
  existing mutation call sites are unchanged; `loadProjects`/`saveProjects` are backed by the seam
  (`saveProjects` reconciles: write all in-map, delete records dropped from the map). A legacy
  `projects.json` is migrated on first run (then retired). Public API and tests preserved.
- **`BudgetManager` now persists through the seam** — three seam stores (budgets, allocations,
  reallocations), one record each, replacing the three flat JSON files rewritten wholesale. The
  in-memory maps, lookup indexes, and all call sites are unchanged; the six load/save methods are
  seam-backed (saves reconcile). Legacy `budgets.json` / `budget_allocations.json` /
  `budget_reallocations.json` migrate on first run (then retire). Public API and tests preserved.
- **RBAC `Manager` now persists through the seam** — its single role-binding state is stored as one
  seam record instead of `~/.prism/rbac.json`. Built-in default roles, public API, and tests
  unchanged; a legacy `rbac.json` migrates on first run (then retires). This completes the first
  pass of the §5 manager convergence (approvals, project, budget, rbac).
- **Per-tenant scope plumbed through the converged managers.** Each manager now carries a
  `seam.Scope` field (the zero Scope on the desktop, so behavior is unchanged) with new
  `…ForScope` constructors — `NewApprovalManagerForScope`, `NewManagerForScope`,
  `NewBudgetManagerForScope`, `rbac.NewManagerForScope` — that the multi-tenant cloud uses to
  partition records per `Principal`. No existing public signature changed. A scope-isolation test
  proves two scopes over one shared store never see each other's records — the property that lets a
  single cloud table serve every tenant safely (design §4, §6.2).
- **DynamoDB seam backend wired into the daemon.** The daemon now selects the persistence backend
  for the four converged managers (project, budget, rbac, approval) at startup: file-backed
  `~/.prism` by default (unchanged desktop behavior), or the shared DynamoDB-backed seam when
  `PRISM_SEAM_BACKEND=dynamodb`. Each record type gets its own table (`PRISM_SEAM_TABLE_PREFIX`,
  default `prism` → `prism-projects`, `prism-budgets`, `prism-budget-allocations`,
  `prism-budget-reallocations`, `prism-rbac`, `prism-approvals`) because the dynamostore's only
  type namespace is the table name — mirroring how the file backend separates the same types by
  directory. The cloud path runs under the zero `Scope` for now (single partition); reading a
  per-Principal scope from configuration is the next step. If the DynamoDB backend is requested but
  AWS config is unavailable or setup fails, the daemon logs and falls back to file-backed
  persistence rather than failing to start. `rbac.State` (the persisted role-binding record) is now
  exported so the wiring can name the store's type parameter.
- **General dependency refresh (minor/patch only).** Root Go module: AWS SDK for Go v2 (core +
  ~20 service clients, e.g. `ec2` v1.300→v1.313, `s3` v1.100→v1.105, `dynamodb` v1.57→v1.60),
  `golang.org/x/crypto` v0.52→v0.53, `x/mod`, `x/sys`, `x/text`, `zalando/go-keyring` v0.2.6→v0.2.8,
  `smithy-go`. GUI module: matching AWS SDK / `x/*` bumps plus `go-git` v5.19.0→v5.19.1,
  `coder/websocket`, `ebitengine/purego`, `samber/lo`. Frontend: `npm update` within existing
  semver ranges (radix-ui suite, `react`/`react-dom` 19.2.3→19.2.7, `msw`, `zod`, `framer-motion`,
  `recharts`, `@playwright/test`, `lucide-react`; lockfile-only, no `package.json` changes).
  No major-version jumps; Wails held at alpha.74 and `scttfrdmn/substrate` at v0.48.0 (both defer
  to focused follow-up PRs). Build, `go vet`, `govulncheck`, Trivy, typecheck, Go tests, and all
  286 frontend unit tests pass; `npm audit` reports 0 vulnerabilities.

### Security
- Bump `golang.org/x/crypto` v0.48.0 → v0.52.0 in both the root and `cmd/prism-gui` modules,
  clearing GO-2026-5015 and GO-2026-5013 (SSH server panics reachable via `pkg/web/terminal.go`).
- Bump `github.com/go-git/go-git/v5` v5.16.4 → v5.19.0 (+ `go-billy/v5` → v5.9.0) in `cmd/prism-gui`,
  clearing GO-2026-4910, GO-2026-4909, GO-2026-4473, CVE-2026-45022, and CVE-2026-44973.
- Bump `golang.org/x/net` → v0.55.0 in both modules, clearing CVE-2026-25681, CVE-2026-27136,
  CVE-2026-39821, and CVE-2026-42502 (html/idna). `govulncheck ./...` and the CI Trivy filesystem
  scan (HIGH/CRITICAL) now both report zero findings in every module.
- Clear 6 `npm audit` findings in the `cmd/prism-gui/frontend` dev-dependency tree (vitest
  critical GHSA-5xrq-8626-4rwp, vite/ws high, @babel/core, js-yaml, brace-expansion) via a
  non-breaking `npm audit fix` (lockfile-only; no `package.json` version changes). Together with
  the Go bumps this unblocks the "Vulnerability & SAST Scan" CI gate on all open PRs.

## [0.35.5] - 2026-05-05

### Fixed
- `prism workspace cost` now reports correct lifetime spend and per-minute burn rate. The previous formula treated `instance.CurrentSpend` (lifetime accumulated dollars, per `pkg/types/runtime.go`) as if it were a daily rate, then multiplied it by lifetime fraction --- producing values that differed from `prism workspace list` by orders of magnitude. `TOTAL SPEND` now reads `CurrentSpend` directly; `COST/MIN` derives from `HourlyRate / 60` when running and from the storage-only EBS rate when stopped/hibernated. The institutional-discount path uses the same storage-only rate when stopped, since EBS storage is not subject to compute discounts.
- `prism workspace cost` `RUNNING` column renamed to `AGE`. The displayed value was wall-clock time since launch, not actual running time. Renamed `CostAnalysis.RunningTime` to `Age` and `formatRunningTime` to `formatAge` for consistency.
- **Update checker replays stale `CurrentVersion`**: after a user upgrades, the cached `update_check.json` still held the pre-upgrade version, so `prism` would print `New version available: 0.30.0 -> 0.35.4` even when already on 0.35.4. Cached entries are now re-evaluated against the running binary on read.
- **Update checker version comparison**: `compareVersions` used raw string ordering, which ranked `0.10.0 < 0.9.0`. Replaced with `golang.org/x/mod/semver`, with explicit guards for `dev`/`unknown`/empty/invalid versions so unreleased builds never trigger an update prompt.

## [0.35.4] - 2026-04-20

### Documentation
- **Comprehensive docs overhaul**: 3-pass audit fixing stale content across 120+ files
- **New user guides**: Workspace Lifecycle (TTL/DNS/idle), Courses, Workshops, Approvals
- **New architecture docs**: spored daemon, updated GUI tech stack, idle detection rewrite
- **Release notes**: added v0.31.0–v0.35.3 (11 missing versions)
- **CLI reference**: documented `--ttl`, `--dns`, `--idle-timeout` flags
- **Command fixes**: `prism launch` → `prism workspace launch` across all docs; `--hours` → `--ttl`
- **TUI removal**: purged all remaining TUI references from user and admin guides
- **Version numbers**: updated all stale v0.5.x/v0.7.x references to v0.35.3
- **Cloudscape cleanup**: replaced all Cloudscape UI references with shadcn/ui
- **Navigation**: expanded mkdocs.yml from 6 to 22+ navigation entries
- **USER_SCENARIOS**: all 8 scenarios updated to v0.35.3; shipped features marked ✅
- **Broken links**: fixed 4 broken internal links including TUI_USER_GUIDE reference
- **Docs version badge**: fixed CI to use `mike deploy` so prismcloud.io version always tracks current release

## [0.35.3] - 2026-04-09

### Security
- **Port validation in proxy handlers**: DCV and web proxy endpoints now validate `?port=` parameter with `strconv.Atoi` + 1-65535 range check. (#601)
- **Shell quoting in research provisioner**: all username interpolations in SSH commands (`id`, `getent passwd`, `last`, `ps -u`) wrapped in `shellQuote()` as defense-in-depth against command injection. (#602)
- **Web proxy CSP/CORS fixed**: replaced stripped CSP with `frame-ancestors 'self'`. CORS restricted to localhost. Removed `Allow-Credentials`. AWS proxy and web proxy both fixed. (#603)
- **Federation token validation**: regex validates AWS federation token format before use in redirect URL. (#603)
- **API path encoding**: added `enc()` (encodeURIComponent) wrapper applied to all 133 path parameters in frontend API client. Prevents path traversal via special characters in resource names. (#604)
- **File permissions hardened**: workshop, course, pricing state files changed from 0644 to 0600. (#605)
- **CSP meta tag**: added `Content-Security-Policy` meta tag to `index.html` restricting default-src to 'self' with allowlisted fonts, API connections, and inline styles. (#605)

## [0.35.2] - 2026-04-09

### Security
- **spored checksum verification**: UserData downloads SHA256 checksum alongside binary and verifies before execution. Binary deleted if checksum fails. Gracefully degrades if checksum not yet published. (#591)
- **S3 temp encryption**: PutObject calls for file transfers now specify `ServerSideEncryption: AES256`. (#598)
- **DCV password hardening**: crypto/rand failure now panics instead of generating predictable sequential passwords. (#598)
- **Proxy CSP**: replaced stripped X-Frame-Options/CSP with `SAMEORIGIN` + `frame-ancestors 'self'`. CORS restricted to localhost. (#596)
- **AppleScript injection defense**: escape backslashes and double quotes in SSH commands before AppleScript interpolation. (#598)
- **CORS tightened**: use allowedOrigins map, removed Allow-Credentials and Authorization from allowed headers. (#598)
- **Auth status endpoint hardened**: unauthenticated GET /api/v1/auth returns minimal `{"status":"ok"}` instead of leaking auth configuration. (#598)
- **Security response headers**: all API responses include `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`. (#599)
- **Path traversal defense**: file operations handler validates paths with filepath.Clean() and rejects non-absolute paths. (#598)
- **TLS support**: daemon uses ListenAndServeTLS when cert/key exist at `~/.prism/tls/`. (#594)

## [0.35.1] - 2026-04-09

### Security
- **Shell injection defense**: `validateRemotePath()` rejects shell metacharacters (`;|&` `` ` `` `$( ' \n \r`) and path traversal (`..`) before paths are interpolated into SSM bash scripts. Applied to PushFile, PullFile, ListRemoteFiles. Fixed dirname quoting in PushFile script. (#589)
- **File permissions tightened**: state directory `0755`→`0700`, all state/config/backup file writes `0644`→`0600`. Prevents local users from reading API keys in `~/.prism/state.json`. (#590)
- **Localhost-only binding**: daemon now binds to `127.0.0.1:{port}` instead of `0.0.0.0:{port}`. Prevents external network access to the API. (#593)
- **Request body size limit**: `http.MaxBytesHandler(100MB)` wraps all handlers, preventing memory exhaustion from oversized POST bodies. (#592)
- **Test mode startup warning**: prominent log warnings when `PRISM_TEST_MODE=true` is active. (#595)
- **Auto-generate API key**: daemon generates a 256-bit API key on first start if none configured, preventing the "no key = open access" default. (#597)

### Fixed
- **WCAG 2.1 AA compliance** (ADA Title II): `<main>` landmark with skip link target (1.3.1/2.4.1), muted foreground contrast 43%→33% for 5.5:1 ratio (1.4.3), Link keyboard activation via Enter/Space (2.1.1), Input autocomplete pass-through (1.3.5), table row aria-selected fix + checkbox aria-label (4.1.2). (#600)

## [0.35.0] - 2026-04-09

### Changed
- **Idle detection delegated to spored**: removed daemon-side CloudWatch polling (564 lines deleted — `metrics_collector.go`, `savings.go`, scheduler monitoring loop). Idle detection now runs on-instance via spored with 7 signals (CPU, network, disk I/O, GPU, terminals, users, recent activity). Policy management UI and fleet-level policy templates remain intact as directives to spored. (#588)

### Added
- **TTL countdown column**: instance table shows time remaining with color-coded status (green >2h, yellow <2h, red <1h, "Expired"). (#588)
- **TTL safety valve**: daemon state monitor (10s polling) stops any running instance past its ExpiresAt as a fallback when spored fails to enforce. (#588)
- **Extend Time action**: "Extend Time (+4h)" appears in the instance dropdown for TTL-enabled workspaces. Updates both local state and the `prism:ttl` EC2 tag so spored picks up the change. (#588)
- **ExpiresAt computed at launch**: TTL duration string (e.g., "8h") is converted to an absolute ExpiresAt timestamp and persisted in state. (#588)

## [0.34.0] - 2026-04-09

### Added
- **spored daemon on every instance**: all new Prism workspaces automatically install the spored daemon from `s3://spawn-binaries-{region}/prism/`. Runs as systemd service with `SPORED_TAG_PREFIX=prism` and `SPORED_DNS_DOMAIN=prismcloud.host`. Provides on-instance idle detection (7 signals), DNS registration, TTL enforcement, cost limits, and spot interruption handling. (#588)
- **Dynamic DNS via prismcloud.host**: workspaces automatically register `{name}.{account-base36}.prismcloud.host` A records via the shared spore-host DNS Lambda. Hostnames shown in instance table, used in SSH connect dialog.
- **`--dns` CLI flag**: specify a custom DNS name when launching (`prism workspace launch python-ml my-project --dns my-workspace`). Defaults to sanitized workspace name.
- **`--ttl` CLI flag**: set a time-to-live for the workspace (`--ttl 8h`). spored enforces the limit on-instance and warns users 5 minutes before expiration.
- **Lifecycle tags**: new EC2 tags set at launch: `prism:dns-name`, `prism:ttl`, `prism:idle-timeout`, `prism:hibernate-on-idle`. Read by spored for on-instance lifecycle management.
- **GUI launch form**: DNS Name and Time Limit fields added to the workspace launch modal. DNS name auto-generated from workspace name if not specified.
- **Hostname column**: instance table shows DNS hostname with monospace font. SSH connect prefers hostname over raw IP.

## [0.33.3] - 2026-04-09

### Fixed
- **GovernancePanel quota validation**: `parseInt(input) || -1` silently coerced invalid input to unlimited. Now validates numeric fields before submit — empty defaults to unlimited, non-numeric shows error message, negative spend rejected.
- **Dead route removed**: `project-detail` activeView rendered PlaceholderView but was unreachable from navigation. Removed route, type union entry, and updated `use-app-data.ts` to load budget data on `projects` view instead.

## [0.33.2] - 2026-04-09

### Fixed
- **LogsView real API**: replaced mock log data with `getInstanceLogs()` calling `GET /api/v1/logs/{instance}?type=...`. Added `getDaemonStatus()` API method. Replaced inline `style={{}}` with Tailwind classes; fixed double-scrolling with `max-h-[60vh]`. Moved "About log types" into collapsible disclosure. (#587)
- **Tab arrow-key navigation**: Tabs component now supports Left/Right arrow keys to cycle between tabs, Home/End to jump to first/last. Uses roving tabindex pattern (WAI-ARIA tabs). (#587)
- **Settings version**: replaced hardcoded `v0.5.1` with dynamic import from `package.json`. (#586)
- **Destructive color contrast**: darkened `--destructive` from `hsl(0 84% 60%)` to `hsl(0 84% 50%)` — white-on-red contrast ~3.78:1 → ~5.0:1 (WCAG AA). (#587)

## [0.33.1] - 2026-04-09

### Fixed
- **Tailwind JIT**: replaced 7 dynamic template-literal class names (`grid-cols-${n}`, `gap-${n}`, `col-span-${n}`) with static lookup objects — fixes grid/column layouts that silently rendered as 0-width (#582)
- **Table selection**: selected row background changed from `bg-muted/50` (same as header) to `bg-primary/10`; added `aria-selected` attribute (#582)
- **Modal focus trap**: wrapped custom Modal in Radix `FocusScope` with `trapped` prop; added Escape key handler and `autoFocus` on close button (WCAG 2.4.3) (#582)
- **prefers-reduced-motion**: added global CSS `@media (prefers-reduced-motion: reduce)` rule killing all animation/transition durations; gated Framer Motion view transition on `useReducedMotion()` hook (#583)
- **Theme token colors**: added `--warning`/`--success` CSS custom properties (light + dark); replaced hardcoded Tailwind colors in Alert, StatusIndicator (both shim and standalone), and 7 focus ring `#0d9488` → `var(--ring)` in index.html (#584)
- **E2E flakes**: budget filter test uses `waitFor` + `force:true` (matches project-workflows pattern); backup restore dialog uses `waitFor visible` instead of hard timeout (#580, #581)
- **Optimistic state cache**: daemon sets expected transitional state (`pending`/`stopping`) before AWS refresh in all 4 lifecycle handlers, ensuring state monitor picks up the instance for polling (#579)

### Changed
- **ApprovalsView**: page header now includes description + Refresh action; empty state shows contextual guidance instead of minimal text (#585)
- **DashboardView**: removed duplicate "New Workspace" button from hero section; converted 3 inline `style={{}}` to Tailwind classes (#585)
- **UserManagementView**: removed duplicate "Create User" from container header (#585)
- **Scroll-to-top**: `AppLayout` resets content scroll position on view navigation (#585)
- **Sidebar badge**: workshops badge changed from `variant="default"` to `variant="secondary"` for consistency (#586)
- **Backups heading**: removed emoji from "Backup Storage Summary" (#586)

## [0.33.0] - 2026-04-09

### Fixed
- **WCAG 1.4.3 contrast**: darkened `--primary` from `hsl(174 77% 32%)` to `hsl(174 77% 26%)` — white-on-teal contrast 3.48:1 → ~5.2:1. Updated `--ring`, `--sidebar-primary`, `--sidebar-ring`, `--accent-foreground` to match. Focus ring now 7.3–8.1:1 against backgrounds (WCAG 2.4.11). (#570)
- **Cold-load error flash**: added `!loading &&` guard on 12 error `Alert` renders across `ApprovalsView`, `CapacityBlocksPanel`, `WorkshopsPanel` (×2), and `CoursesPanel` (×7). Prevents red error flash when daemon is slow to respond on first mount.
- **FormField `aria-describedby`**: description, constraint, and error `<p>` elements now have `id` attributes linked to the child input via `aria-describedby`. Error state adds `aria-invalid` and `role="alert"` for live-region screen reader announcement. Applies to all `FormField` usage globally. (#570)

### Changed
- **Dashboard density**: removed misplaced `<Header variant="h1">Dashboard` (rendered below hero), 3-column stats grid showing "0" counters, and Quick Actions bar (duplicated sidebar nav). Replaced with compact inline status row: connection indicator + template count + workspace count + New Workspace CTA.
- **Backups prose**: replaced always-visible info container (heading + 2 paragraphs + 3-column grid) with a `<details>` disclosure defaulting to closed. Data table is immediately visible on load.
- **Settings responsive nav**: at ≤1024px viewport width, the 220px settings sub-nav sidebar is replaced by a horizontal scrollable tab bar. Uses Tailwind `lg:` breakpoint — no JS state needed.

## [0.32.0] - 2026-04-07

### Added
- Framer Motion view transitions: each view slides in from the right (x: 14→0, 160ms ease) keyed on `activeView`. Modal spring animation (scale + y-translate, 150ms). Cards stagger entrance (45ms between items). Framer Motion mock in `src/__mocks__/framer-motion.ts` keeps all 286 unit tests passing in JSDOM.

### Changed
- Font replaced: Inter → Atkinson Hyperlegible Next (UI) + Atkinson Hyperlegible Mono (code blocks), loaded via Google Fonts.
- Color palette replaced: Warm Slate — teal-600 (`hsl 174 77% 32%`) primary, warm white (`hsl 40 33% 98%`) background, stone-500 muted foreground, warm near-black dark mode. All 34 HSL CSS variables updated in `:root` and `.dark`. Hardcoded `bg-blue-*` / `text-blue-*` in `cloudscape-shim.tsx` replaced with semantic `bg-primary` / `text-primary`.
- Sonner `<Toaster>` moved to bottom-right (`position="bottom-right"`, `expand={false}`, `closeButton`).

### Fixed
- Remaining `setState(notifications)` silent drops converted to `toast.*` calls: `onProvision` callback, `EditProjectModal` onSubmit, `EditUserModal` onSubmit, `CreateEFSVolumeModal` (removed `onNotify` prop), `CreateEBSVolumeModal` (same).
- `WorkshopPage.ts` and `CoursePage.ts` E2E nav selectors scoped to `[data-sidebar="menu-button"]` to prevent strict-mode collision with same-named content-area buttons.
- `settings.spec.ts` Templates nav selector scoped to sidebar to prevent collision with SettingsView "Template Marketplace" sub-nav button during parallel test runs.
## [0.31.0] - 2026-04-06

### Changed
- GUI shell migrated from Cloudscape `AppLayout` + `SideNavigation` + `Flashbar` to shadcn/ui `Sidebar` + Sonner toasts. All 23 nav items preserved; badges for pending approvals, courses, workshops, instances, and templates use shadcn `Badge` variants. The Cloudscape-shim compatibility layer bridges remaining views during incremental migration.
- Installed shadcn/ui component library (Radix-based, code-owned), Tailwind CSS v4 (`@tailwindcss/vite` — no config file), Sonner toast library, and Lucide React icons. Removed all `@cloudscape-design/*` packages.
- Extracted 13 views from the monolithic `App.tsx` to top-level module components, fixing React re-mount on every state change (issue #13): `ApprovalsView`, `DashboardView` (+ `RecentWorkspaces`), `LogsView`, `WebViewView`, `TemplateSelectionView`, `PlaceholderView`, `BackupManagementView`, `AMIManagementView`, `MarketplaceView`, `IdleDetectionView`, `ProfileSelectorView`, `InstanceManagementView`, `StorageManagementView`, `ProjectManagementView`, `UserManagementView`. App.tsx reduced from ~11,954 → ~7,330 lines.
- Template helper functions (`getTemplateName`, `getTemplateSlug`, `getTemplateDescription`, `getTemplateTags`) moved to `src/lib/template-utils.ts` — shared across views without closure coupling.
- All `addNotification` / `setNotification` patterns for user-visible events replaced with `toast.success()` / `toast.error()` / `toast.warning()` calls.
- `ApiContext` provider wraps `PrismApp` return; extracted views call `useApi()` from `src/hooks/use-api.ts` instead of `window.__apiClient`. All five component files migrated: `CoursesPanel`, `WorkshopsPanel`, `GovernancePanel`, `ProjectDetailView`, `InvitationManagementView`.
- Extracted three modals from `App.tsx` to `src/modals/`: `DeleteConfirmationModal` (moves `confirmationText` state into modal; `DeleteModalConfig` interface exported for callers), `HibernateConfirmationModal` (uses `useApi()` + Sonner toasts; conditional render pattern prevents ApiProvider requirement in unit tests), `IdlePolicyModal` (same conditional-render pattern; removes `logger.error` dependency).
- Settings inner navigation replaced: Cloudscape `SideNavigation` with `#fragment` hrefs removed; replaced with styled `<button>` list using `data-testid="settings-nav-{section}"`. `SettingsPage.ts` E2E page object updated to use `getByTestId()` selectors.

## [0.30.0] - 2026-04-02

### Removed
- TUI (Terminal User Interface) — retired the BubbleTea `prism tui` command (#578). The interface had fallen behind the CLI and GUI in feature coverage (missing file ops, governance, courses, and all features added after v0.26.0), was excluded from CI, and contributed ~22,000 lines of maintenance overhead. The CLI and GUI cover all use cases. Removes `github.com/charmbracelet/bubbletea`, `bubbles`, and `lipgloss` dependencies.

## [0.29.2] - 2026-04-02

### Fixed
- ARIA/WAI-ARIA accessibility fixes across GUI and landing page (#568, #569, #571, #572, #573, #574, #575, #576, #577)
- Terminal.tsx: added `role="status" aria-live="polite" aria-atomic="true"` to connection status bar; `aria-hidden="true"` on decorative pulsing dot; `role="application"` + `aria-label` on terminal container (#568, #576)
- SSHKeyModal.tsx: added `aria-label` and `aria-readonly="true"` to public key and private key textareas (#569)
- WebView.tsx: added `role="status" aria-live="polite"` + `aria-label` to service loading overlay (#574)
- App.tsx: added skip-navigation link and `tabIndex={-1}` on `#main-content` for keyboard navigation (#573)
- Landing page (`docs/index.md`): added `<main>` landmark, `<section aria-labelledby>` for each section, `<article>` for all cards, fixed heading hierarchy (`h3` → `h2` for feature cards), `aria-hidden="true"` on all decorative emoji (#572, #575)
- Landing page CSS (`docs/overrides/home.html`): added `:focus-visible` outlines to all three button classes (#571)

## [0.29.1] - 2026-04-02

### Fixed
- Project deletion now blocks when running instances belong to the project (#539). `project.Manager.getActiveInstancesForProject` was a permanent stub returning empty — deletion safety was silently bypassed. Added `SetActiveInstancesFunc` callback; daemon wires it against `stateManager.LoadState()` on startup. Added unit test asserting deletion fails with active instances and succeeds after they stop.

## [0.29.0] - 2026-04-02

### Added
- SSM file operations backend: `prism workspace files push/pull/list` — transfers files to/from running instances via S3 relay (`pkg/aws/file_ops.go`, `pkg/daemon/file_ops_handlers.go`, `internal/cli/file_ops_cobra.go`) (#30a)
- `PrismAPI` interface now declares `ListInstanceFiles`, `PushFileToInstance`, `PullFileFromInstance`
- Unit tests for `parseLsOutput` covering all edge cases (`pkg/aws/file_ops_test.go`)
- Substrate v0.48.0 — all 6 Substrate integration tests now pass (`TestSubstrateLaunchInstance`, `TestSubstrateEBSAttachDetach`, `TestSubstrateIAMInstanceProfile`, `TestSubstrateSSMRunCommand`, `TestSubstrateCreateEBSVolume`, `TestSubstrateErrorHandling`) (closes substrate#265, #267)

### Fixed
- All 268 pre-existing TypeScript typecheck errors resolved (`npm run typecheck` now clean)
- `parseLsOutput`: field-count guard reduced from 9 to 7 to match actual `ls -la --time-style` output (name field index corrected from `[8:]` to `[6:]`)

### Changed
- Replaced LocalStack with Substrate for all AWS integration testing (no Docker required)

## [0.28.1] - 2026-04-02

### Added
- E2E tests for AMI Management (`ami-workflows.spec.ts`, 4 tests — Settings > Advanced > AMI Management)
- E2E tests for Template Marketplace (`marketplace-workflows.spec.ts`, 3 tests — Settings > Advanced > Template Marketplace)
- E2E tests for Idle Detection (`idle-workflows.spec.ts`, 4 tests — Settings > Advanced > Idle Detection)
- `AMIPage`, `MarketplacePage`, `IdlePage` page objects

### Fixed
- `WorkshopsPanel.tsx`: added `data-testid="create-workshop-button"` (missing testid caused 17.5s timeout and cascade ECONNREFUSED failures in E2E)
- `course_handlers.go`: EFS creation now skips AWS in `PRISM_TEST_MODE=true`, returning a deterministic fake ID — previously the handler hung indefinitely during E2E tests
- `BasePage.navigateToSettingsAdvanced()`: expansion check computed wrong href (e.g. `#ami-management` instead of `#ami`); now checks link visibility by text
- `CoursePage.waitForCourseList()`: added explicit refresh + `waitForResponse` to eliminate `beforeEach` course-creation race condition
- `global-setup.js`: added `cleanupTestCourses()` pre-run cleanup to prevent stale test courses causing strict-mode violations
- `workshop-workflows.spec.ts`: scoped `getByText(/end workshop/i)` to dialog to fix strict-mode violation (header + button both matched)
- `playwright.config.js`: added `NODE_OPTIONS="--max-old-space-size=4096"` to Vite dev server command to prevent crash after BABEL deoptimisation of App.tsx (>500 KB)

## [0.28.0] - 2026-04-01

### Added
- Cloudscape Design System upgrade to 3.0.1255
- Windows GUID support
- Semver constraint enforcement
- Policy framework fixes
- Research migration

### Changed
- Multiple community PRs (#562–563)

## [0.7.5] - 2026-01-27

### 🎯 Focus: Complete User Management System

**Production-ready user management with comprehensive lifecycle operations**:
- Professional user interface with detailed user views and statistics
- Status management with enable/disable functionality
- Workspace provisioning with consistent UID/GID across instances
- SSH key management and display
- Delete safety with provisioned workspace warnings
- All features fully tested with E2E coverage (9/9 tests passing)

### Added

- **User Detail View with SSH Keys** (#346):
  - Comprehensive user details modal showing username, UID, email, full name
  - SSH keys table displaying all configured keys with fingerprints
  - Creation timestamp and user metadata
  - Professional Cloudscape modal design
  - API integration: GET `/api/v1/users/{username}/ssh-key`

- **User Statistics Dashboard** (#456):
  - Real-time statistics cards showing Total Users, Active Users, SSH Keys, Provisioned Workspaces
  - Automatic calculation from user data
  - Color-coded status indicators
  - Grid layout with responsive design

- **User Status Management** (#348):
  - Status filtering: All Users, Active, Inactive dropdown
  - Dynamic table filtering with automatic counter updates
  - Status detail modal with comprehensive user information
  - Color-coded StatusIndicator components (success/warning/error)
  - API integration: GET `/api/v1/users/{username}/status`

- **User Status Update (Enable/Disable)** (#348):
  - Enable/disable user accounts through UI
  - Dynamic action menu showing "Enable User" or "Disable User" based on current status
  - Status display: "Active" (enabled) or "Suspended" (disabled) in table
  - Optimistic UI updates with immediate feedback
  - Backend endpoints: POST `/api/v1/users/{username}/enable`, POST `/api/v1/users/{username}/disable`
  - New users default to enabled=true
  - Added Enabled bool field to ResearchUserConfig

- **User Provisioning on Workspaces** (#347):
  - "Provision on Workspace" action in user actions dropdown
  - Workspace selection modal showing running instances only
  - Creates user account with consistent UID/GID across instances
  - Installs SSH keys on target workspace
  - API integration: POST `/api/v1/users/{username}/provision`
  - Success/error notifications with detailed messages

- **Provisioned Workspaces Display** (#347):
  - "Provisioned Workspaces" section in user details modal
  - Table showing all workspaces where user is provisioned
  - Empty state for users with no workspaces
  - Automatic refresh on provisioning actions

- **Delete Safety Warnings** (#347):
  - Automatic detection of provisioned workspaces before deletion
  - Enhanced delete confirmation modal with warning alerts
  - Warning message: "This user has N provisioned workspace(s)..."
  - Informs about access removal consequences
  - Prevents accidental deletion of active users

### Changed

- **User Interface Enhancements**:
  - Updated user table to show UID, SSH keys count, provisioned workspaces count
  - Added status column with color-coded indicators
  - Enhanced actions dropdown with conditional menu items
  - Improved modal designs with better spacing and layout

- **Backend User Handlers**:
  - Extended research user handlers with enable/disable operations
  - Added status field to research user configuration
  - Improved error handling and status codes
  - Better API response messages

### Technical Implementation

- **Frontend** (cmd/prism-gui/frontend/src/App.tsx):
  - Added 14 new state variables for user management
  - Implemented 5 new API methods (getUserStatus, provisionUser, enableUser, disableUser, getUserSSHKeys)
  - Created 3 new modals: User Details, User Status, User Provision
  - Added status filter dropdown with dynamic filtering
  - Enhanced delete confirmation with optional warnings
  - ~300 lines of production-ready React/TypeScript code

- **Backend**:
  - pkg/daemon/research_user_handlers.go: Added enable/disable routes and handlers
  - pkg/research/types.go: Added Enabled bool field to ResearchUserConfig
  - pkg/research/manager.go: Set Enabled: true for new users by default

- **E2E Tests** (cmd/prism-gui/frontend/tests/e2e/user-workflows.spec.ts):
  - Activated 9 comprehensive user management tests
  - Fixed Cloudscape Select interaction patterns
  - Added proper wait states and modal selectors
  - All tests passing: 100% success rate (chromium + webkit)

- **API Endpoints Added**:
  - GET `/api/v1/users/{username}/status`: Get detailed user status
  - POST `/api/v1/users/{username}/provision`: Provision user on workspace
  - GET `/api/v1/users/{username}/ssh-key`: Get user SSH keys
  - POST `/api/v1/users/{username}/enable`: Enable user account
  - POST `/api/v1/users/{username}/disable`: Disable user account

### Test Coverage

**9/9 E2E Tests Passing** (100% milestone completion):
- ✅ should display UID for each user
- ✅ should display existing SSH keys
- ✅ should show user statistics
- ✅ should filter users by status
- ✅ should view user status details
- ✅ should provision user on workspace
- ✅ should show provisioned workspaces for user
- ✅ should warn when deleting user with active workspaces
- ✅ should update user status

### Fixed

- Cloudscape Select component interaction patterns in E2E tests
- User status default values for backward compatibility
- Modal test selectors for reliable E2E testing
- Proper error handling in status update operations

## [0.5.11] - 2025-11-09

### 🎯 Focus: Complete User Invitation & Collaboration System

**Production-ready invitation system enabling zero-configuration team onboarding**:
- Individual, bulk, and shared token invitations with full lifecycle management
- Automatic research user provisioning with SSH keys, UID/GID, and EFS home directories
- AWS quota validation prevents bulk invitation failures
- Professional Cloudscape GUI with QR code generation
- End-to-end automation from invitation send to workspace access

### Added

- **Role-Based Permission System** (#102):
  - Users automatically added to project.Members when accepting invitations
  - Role validation enforces allowed values: owner, admin, member, viewer
  - Duplicate member detection prevents re-adding existing users
  - Graceful error handling for edge cases
  - Automatic member addition in pkg/daemon/invitation_handlers.go

- **Research User Auto-Provisioning** (#106):
  - SSH key generation (Ed25519) and storage on invitation acceptance
  - Deterministic UID/GID allocation ensuring consistent IDs across all instances
  - EFS home directory configuration at `/efs/home/{username}`
  - Complete user profile saved to `.prism` directory
  - Graceful failures (provisioning errors don't block invitation acceptance)
  - Integration in pkg/daemon/invitation_handlers.go

- **AWS Quota Validation for Bulk Invitations** (#105):
  - Pre-flight quota checking with detailed capacity analysis
  - Instance type vCPU mapping for 50+ instance types (t3, t4g, c7g, m5, m6g, r5, g4dn, p3)
  - Current usage calculation from running/pending instances
  - AWS Service Quotas API integration for limit retrieval
  - REST endpoint: `POST /api/v1/invitations/quota-check`
  - Validation logic: `required_vcpus = count × instance_type_vcpus`
  - New file: pkg/aws/quota.go (220+ lines)

- **Invitation Management GUI** (#103):
  - Professional Cloudscape-based GUI for invitation lifecycle management
  - Individual invitations: Add by token, accept/decline with confirmation dialogs
  - Bulk invitations: Send to multiple emails (comma/newline separated) with role selection
  - Shared tokens: Create reusable tokens for classrooms/workshops with QR codes
  - Status tracking: View all invitations with status badges (pending/accepted/declined/expired)
  - Time display: Human-readable time remaining (e.g., "5 days remaining")
  - Already implemented in cmd/prism-gui/frontend/src/App.tsx

### Changed

- **Invitation Acceptance Flow**:
  - Now triggers automatic member addition with role validation
  - Auto-provisions research users with complete SSH and filesystem setup
  - Enhanced API response includes provisioning status and research user details

- **Project Member Management**:
  - Added role validation in AddProjectMember method (pkg/project/manager.go)
  - Enforces valid roles: owner, admin, member, viewer
  - Prevents duplicate member addition with graceful error handling

### Technical Implementation

- **Backend**:
  - pkg/daemon/invitation_handlers.go: Member addition and auto-provisioning integration
  - pkg/project/manager.go: Role validation (lines 272-281)
  - pkg/aws/quota.go: Complete quota validation system (220+ lines)
  - pkg/daemon/invitation_handlers.go: Quota check endpoint (lines 601-657)

- **API Endpoints Added**:
  - POST /api/v1/invitations/quota-check: Check AWS quota for bulk invitations

- **Dependencies Added**:
  - github.com/aws/aws-sdk-go-v2/service/servicequotas: AWS Service Quotas API

- **Documentation**:
  - docs/releases/RELEASE_NOTES_v0.5.11.md (500+ lines)
  - docs/user-guides/INVITATION_USER_GUIDE.md (600+ lines)
  - docs/USER_SCENARIOS/: Updated 3 persona walkthroughs with v0.5.11 features

### Use Cases

**University Class (50 Students)**:
- Check quota for 50 students × t3.medium instances
- Send bulk invitations with automatic member addition
- Students auto-provisioned with SSH keys on acceptance
- Zero manual configuration required

**Research Lab (10 Researchers)**:
- Send individual invitations with roles (admin/member)
- Automatic research user provisioning with UID/GID consistency
- SSH keys ready in `~/.prism/ssh_keys/`
- EFS home directories at `/efs/home/{username}`

**Conference Workshop (100 Attendees)**:
- Create shared token for workshop with QR code
- First-come-first-served redemption (limit: 100)
- Attendees scan and redeem instantly
- Auto-provisioning enables immediate workspace access

### Breaking Changes

None. This release is fully backward-compatible.

### Commits

- 58eef9691: Role-Based Permission System (#102)
- 1348ac245: Research User Auto-Provisioning (#106)
- 18618de01: AWS Quota Validation (#105)
- bd78ea89b: Complete v0.5.11 documentation
- bd3e6a8e0: Update persona walkthroughs for v0.5.11

## [0.5.10] - 2025-11-08

### 🎯 Focus: Multi-Project Budget System

**Redesigned budget architecture enabling real-world research funding workflows**:
- Many-to-many relationships: 1 budget → N projects OR N budgets → 1 project
- Real-time spending tracking with automatic enforcement
- Grant-compliant audit trails for institutional reporting
- $12K+/year cost savings through automated zombie resource detection

### Added
- **Shared Budget Pools** (#97):
  - Create budget pools representing funding sources (NSF grants, department budgets, AWS credits)
  - Track total amount, allocated amount, and spent amount independently
  - Support quarterly, annual, and grant-period budget cycles
  - Budget period configuration with start/end dates

- **Project Budget Allocations** (#98):
  - Allocate portions of budget pools to specific projects
  - Per-project spending limits within shared budgets
  - Independent spending tracking per allocation
  - Project-specific alert thresholds
  - DefaultAllocationID for frictionless resource launches

- **Budget Reallocation System** (#99):
  - Move funds between project allocations within same budget
  - Mandatory reason field for grant compliance
  - Immutable audit trail for retrospective analysis
  - Real-time validation of available funds
  - Complete reallocation history tracking

- **Multi-Project Cost Rollup** (#100):
  - Institution-wide budget rollup across all budget pools
  - Per-budget cost summary with project breakdown
  - Per-project cost summary across multiple funding sources
  - Real-time spending aggregation and reporting

- **Funding Source Selection** (#233):
  - Select funding allocation when launching workspaces/volumes
  - `--funding` CLI flag for explicit allocation selection
  - Automatic funding via project's DefaultAllocation
  - Pre-launch validation of allocation availability
  - Instance tagging with `prism:funding-allocation-id`

- **Backup Funding System** (#234):
  - Configure backup allocation for primary exhaustion handling
  - Automatic switch to backup on primary allocation exhaustion
  - Continuity-preserving: resources continue running on backup
  - Email notifications on backup activation
  - Audit trail for all exhaustion events

- **Enhanced Resource Tagging** (#128):
  - 15+ comprehensive `prism:*` namespaced tags
  - AWS Cost Explorer integration (Application, Environment, CostCenter)
  - Lifecycle tracking (prism:launched-at, prism:launched-by)
  - Zombie resource detection script (scripts/cleanup_untagged_resources.sh)
  - ROI: $12K+/year zombie resource prevention

- **Budget Philosophy Documentation** (#236):
  - Comprehensive BUDGET_PHILOSOPHY.md (500+ lines)
  - Two-tier architecture explanation (Budget Pools → Allocations → Projects)
  - Real-world use cases for grants, departments, classes
  - Mental models for students, PIs, and institutional admins
  - Design rationale and comparison to other models

### Changed
- **Budget Data Model**:
  - Migrated from 1:1 (budget → project) to many-to-many (via allocations)
  - Budget pools now independent entities with multiple project allocations
  - Projects can have multiple funding sources simultaneously

- **Cost Tracking**:
  - Real-time spending tracked per allocation (not just per project)
  - Allocation exhaustion detection with backup funding support
  - Enhanced cost attribution for multi-source projects

### Technical Implementation
- **Backend**:
  - pkg/types/project.go: Budget and ProjectBudgetAllocation types
  - pkg/project/budget_manager.go: Budget pool and allocation management (1,133 lines)
  - pkg/daemon/budget_handlers.go: REST API endpoints (613 lines)
  - 20+ API endpoints for budget operations

- **API Endpoints Added**:
  - POST/GET/PUT/DELETE /api/v1/budgets
  - POST/GET/PUT/DELETE /api/v1/allocations
  - POST /api/v1/reallocations
  - GET /api/v1/budgets/{id}/summary
  - GET /api/v1/budgets/{id}/report
  - GET /api/v1/allocations/{id}/status

- **Documentation**:
  - docs/BUDGET_PHILOSOPHY.md (26KB, 500+ lines)
  - docs/RESOURCE_TAGGING.md (16KB, 568 lines)
  - docs/BUDGET_BANKING_PHILOSOPHY.md (updated with cross-references)

### Use Cases Enabled
1. **Single Grant → Multiple Projects**:
   - NSF grant ($50K) funds 3 research projects with independent spending limits

2. **Multi-Source Project Funding**:
   - Large project funded by NSF ($50K) + DOE ($30K) + Institution ($10K)
   - Per-source spending tracking for compliance reporting

3. **Department-Wide Student Budgets**:
   - CS Department Q1 budget ($10K) allocated to 20 student projects ($500 each)
   - Automated per-student spending limits and monitoring

4. **Grant Transition with Backup Funding**:
   - Primary grant exhausts → automatic switch to department bridge funding
   - Research continuity maintained during grant renewals

### Success Metrics Achieved
- ✅ 1 budget → N projects (grant funding multiple research projects)
- ✅ N budgets → 1 project (multi-source funding)
- ✅ Real-time cost tracking and pre-launch validation
- ✅ Grant compliance with complete audit trails
- ✅ $12K+/year zombie resource prevention

### Migration Notes
- Existing budgets remain functional (backward compatible)
- New budget features available immediately via API/CLI
- No breaking changes to existing project workflows
- Budget tagging automatically applied to new resources

### Related Issues
Closed: #97, #98, #99, #100, #128, #232, #233, #234, #236

---

## [0.5.8] - 2025-10-29

### 🎯 Focus: Quick Start Experience

**Dramatically Reduced Time to First Workspace**:
- Time to first workspace: 15 minutes → < 30 seconds
- First-attempt success rate: ~60% → > 90% target
- User confusion rate: ~40% → < 12% target (70% reduction)
- Research-friendly terminology throughout interface

### Added
- **CLI Onboarding Wizard** (#17):
  - `prism init` command for guided setup
  - AWS configuration validation with helpful guidance
  - Research area selection with template recommendations
  - Optional budget configuration for grant-funded research
  - First workspace launch with pre-filled sensible defaults
  - Progress tracking with clear messaging
  - Complete 30-second onboarding experience

### Changed
- **User-Facing Terminology** (#15):
  - Renamed "Instances" → "Workspaces" in all user-facing strings
  - CLI display output now shows research-friendly "workspace" terminology
  - TUI interface updated with "Workspaces" throughout
  - GUI navigation, dialogs, and help text updated
  - Error messages use clear, friendly "workspace" language
  - Documentation reflects new terminology
  - **Backward Compatibility**: CLI commands unchanged (`prism list`, `prism launch`, etc.)

### Technical
- **Impact**: Removes AWS jargon barrier for researchers
- **UX Achievement**: 30-second time-to-first-workspace
- **Compatibility**: 100% backward compatible with existing commands
- **Internal Consistency**: Code still uses "instance" for AWS alignment

### User Journey Improvement

**Before v0.5.8** (15+ minutes, multiple failures):
1. Install Prism
2. Read AWS setup documentation
3. Configure AWS credentials manually
4. Learn region concepts
5. Browse template list (confused by options)
6. Trial-and-error with launch command
7. Wait for provisioning (unsure if working)
8. Finally connect to workspace

**After v0.5.8** (< 30 seconds, zero failures):
1. Install Prism
2. Run: `prism init`
3. Follow guided wizard (4 questions)
4. Confirm launch
5. Connect to workspace

### Example Usage

```bash
$ prism init

Welcome to Prism! Let's get you set up in 30 seconds.

✓ AWS credentials detected (profile: research-lab)
✓ Region: us-west-2
✓ Research area: Machine Learning
  Recommended template: Python Machine Learning

Ready to launch your first workspace? [Y/n]

🚀 Launching "ml-workspace-1"...
✓ Workspace ready in 22 seconds!

Connect: prism connect ml-workspace-1
```

---

## [0.5.7] - 2025-10-26

### 🎉 Major: Template File Provisioning

**S3-Backed File Provisioning System**:
- Complete S3 transfer system for template provisioning
- Multipart transfers supporting files up to 5TB
- MD5 checksum verification for data integrity
- Progress tracking with real-time updates
- Conditional provisioning (architecture-specific files)
- Required vs optional files with graceful fallback
- Auto-cleanup from S3 after download
- Complete documentation: [TEMPLATE_FILE_PROVISIONING.md](docs/TEMPLATE_FILE_PROVISIONING.md)

### Added
- **Template File Provisioning** (#64, #31):
  - S3 transfer system with multipart upload support
  - Template schema extensions for file configuration
  - S3 file download integration in instance launch
  - Documentation and examples for dataset distribution

### Fixed
- **Test Infrastructure Stability** (#83):
  - Fixed Issue #83 regression (tests hitting AWS and timing out)
  - Fixed data race in system_metrics.go (concurrent cache access)
  - Test performance: 206x faster (97.961s → 0.463s)
  - All smoke tests passing (8/8)
  - Zero race conditions detected

### Changed
- **Script Cleanup**:
  - Completed CloudWorkStation → Prism rename across all scripts
  - Updated 19 script files with consistent branding
  - Documentation consistency verification

### Technical
- **Impact**: Enable multi-GB dataset distribution, binary deployment, pre-trained models
- **Performance**: Reliable CI/CD pipeline with fast developer feedback loop
- **Quality**: Production-ready test infrastructure with race-free concurrent operations

---

## [0.5.6] - 2025-10-26

### 🎉 Major: Complete Prism Rebrand

**Project Rename**: CloudWorkStation → Prism
- Complete code rename (29,225 files across 3 PRs)
- GitHub repository rename: `scttfrdmn/prism`
- All binaries renamed: `cws` → `prism`, `prismd` → `prismd`
- Configuration directory: `.prism` → `.prism`
- Go module: `github.com/scttfrdmn/prism`

### Added
- **Feature Issues Created**:
  - Issue #90: Launch Throttling System (rate limiting for cost control)
  - Issue #91: Local System Sleep/Wake Detection with Auto-Hibernation

### Fixed
- **CLI Test Fixes** (#88): Updated string constants for Prism rename
  - Fixed: TestConstants, TestUsageMessages, TestErrorHelperFunctions
  - Updated all command references: `cws` → `prism`
  
- **API Client Test Fixes** (#89): Updated timeout expectations
  - Fixed: TestNewClient, TestDefaultPerformanceOptions  
  - Updated timeout expectations: 30s → 60s

- **Storage Test Fixes** (#87): Complete storage volume type migration
  - Fixed 55 test failures in storage system
  - Completed unified StorageVolume type implementation

### Changed
- **Repository Infrastructure** (Commit c37937e35):
  - Updated 45 files with new repository URLs
  - Package manifests (homebrew, chocolatey, conda, rpm, deb)
  - Build scripts and CI/CD configurations
  - Documentation configs updated

- **CLI Command Structure** (#79):
  - Consistent command hierarchy implementation
  - Improved user experience and discoverability

### Technical
- **3 PRs Merged**:
  - PR #85: Code rename (189 Go files, 55+ scripts)
  - PR #86: Documentation updates (320 markdown files)
  - PR #87: Storage test remediation (55 tests fixed)
  - PR #88: CLI test fixes (3 tests fixed)
  - PR #89: API timeout test fixes (2 tests fixed)

- **Repository Renamed**: GitHub automatically redirects old URLs
- **Backward Compatibility**: All old URLs redirect to new repository

### Benefits
- **Complete Brand Consistency**: Unified naming across all components
- **Professional Identity**: Clean, memorable project name
- **Improved Discoverability**: `prism` command is intuitive
- **Test Stability**: 60 test failures resolved

### Migration Guide
Existing users should:
1. Update git remotes: `git remote set-url origin git@github.com:scttfrdmn/prism.git`
2. Rebuild binaries: `make build`
3. Configuration automatically migrates from `.prism` to `.prism`
4. Old commands still work via shell aliases (optional)

**Note**: GitHub automatically redirects old repository URLs, so existing clones continue to work without changes.

## [0.5.4] - 2025-10-18

### Added
- **Universal Version System**: Dynamic OS version selection at launch time
  - `--version` flag for specifying OS versions (e.g., `--version 24.04`, `--version 22.04`)
  - Support for version aliases: `latest`, `lts`, `previous-lts`
  - 4-level hierarchical AMI structure: distro → version → region → architecture
  - AWS SSM Parameter Store integration for Ubuntu, Amazon Linux, Debian
  - Static fallback AMIs for Rocky Linux, RHEL, Alpine
- **AMI Freshness Checking**: Proactive validation of static AMI IDs
  - `prism ami check-freshness` command to validate AMI mappings
  - Automatic detection of outdated AMIs against latest SSM values
  - Clear reporting with recommended update actions
  - Support for all distributions (SSM-backed and static)
- **Enhanced AMI Discovery**: Intelligent AMI resolution with automatic updates
  - Daemon startup warm-up with bulk AMI discovery
  - Hybrid discovery: SSM Parameter Store with static fallback
  - Regional AMI caching for improved performance

### Enhanced
- **Version Resolution**: 3-tier priority system (User → Template → Default)
- **Template System**: Version constraints in template dependencies
- **Documentation**: Complete VERSION_SYSTEM_IMPLEMENTATION.md guide

### Technical
- Added `pkg/aws/ami_discovery.go` (416 lines) - AMI discovery and freshness checking
- Added `pkg/templates/resolver.go` (267 lines) - Version resolution and aliases
- Added `pkg/templates/dependencies.go` (300 lines) - Dependency resolution
- Enhanced `pkg/templates/parser.go` with hierarchical AMI structure
- Added `Version` field to `LaunchRequest` for version specification
- Integrated AMI discovery into daemon initialization
- Added REST API endpoint `/api/v1/ami/check-freshness`
- Added CLI commands: `prism ami check-freshness`

### Benefits
- **No Template Explosion**: Single template supports multiple OS versions
- **Always Current**: SSM integration provides latest AMIs automatically
- **Version Flexibility**: Choose any supported OS version at launch time
- **Proactive Maintenance**: Monthly freshness checks identify outdated AMIs
- **Clear Communication**: Users know exactly which version they're getting

### Supported Distributions
- Ubuntu: 24.04, 22.04, 20.04 (SSM-backed)
- Rocky Linux: 10, 9 (static fallback)
- Amazon Linux: 2023, 2 (SSM-backed)
- Debian: 12 (SSM-backed)
- RHEL: 9 (static fallback)
- Alpine: 3.20 (static fallback)

## [0.5.3] - 2025-10-17

### Development Workflow
- **Simplified Git Hooks**: Streamlined pre-commit checks to run in < 5 seconds (down from 2-5 minutes)
  - Fast auto-formatting only (gofmt, goimports, go mod tidy)
  - Heavy checks (lint, tests) moved to explicit make targets for pre-push validation
- **Enhanced Makefile**: Go Report Card linting integration with comprehensive quality tools
  - gofmt, goimports, go vet, gocyclo, misspell, staticcheck, golangci-lint
  - Quick Start workflow documentation for new developers
- **Documentation Cleanup**: Organized 20+ historical documents into structured archive
  - Created docs/archive/ with planning/, implementation/, deprecated/ subdirectories
  - Preserved historical context while cleaning main docs/ directory

### Quality Improvements
- **Cost Display Precision**: Enhanced cost output from 3 to 4 decimal places for sub-cent accuracy
- **Version Synchronization**: Fixed Makefile version mismatch (aligned with runtime version)

### Infrastructure
- **Build System**: Maintained zero compilation errors for production binaries
- **Testing**: Core functionality verification with automated smoke tests
- **GoReleaser Integration**: Complete distribution automation with multi-platform support
  - Automated builds for Linux, macOS, Windows (AMD64 + ARM64)
  - Homebrew tap integration (scttfrdmn/homebrew-tap)
  - Scoop bucket support for Windows package management
  - Debian/RPM/APK packages for Linux distributions
  - Docker multi-arch images with manifest support
  - Makefile targets for local testing (snapshot mode)
  - Simplified Homebrew formula with auto-starting daemon messaging

## [0.4.1] - 2025-08-08

### Critical Bug Fixes
- **GUI Content Display**: Fixed blank white areas in Dashboard, Instances, Templates, and Storage sections
- **Version Verification**: Fixed daemon version reporting (was hardcoded "0.1.0", now reports actual version)
- **CLI Version Panic**: Fixed crash when GitCommit string shorter than 8 characters  
- **Storage API Mismatch**: Fixed JSON unmarshaling errors in EFS/EBS volume endpoints
- **GUI Threading**: Eliminated threading warnings and improved stability
- **Daemon Version Checking**: Added proper version verification after daemon startup

### User Experience Improvements
- **System Tray Integration**: Enhanced window management and data refresh when shown from tray
- **Navigation Highlighting**: Fixed sidebar navigation button highlighting without rebuilding
- **Connection Status**: Improved daemon connection status detection with proper timeouts
- **Error Messages**: More helpful and actionable error messages throughout the application

### Documentation
- **Major Cleanup**: Organized 50+ scattered documentation files into clean structure
  - Root: 14 essential project files
  - docs/: 41 current documentation files organized by category  
  - docs/archive/: 42 historical files properly archived
- **Updated Navigation**: Comprehensive documentation index with clear categorization
- **User Guides**: Improved organization of user-facing documentation

### Technical Improvements  
- **API Consistency**: Storage and volume endpoints now return arrays instead of maps
- **Version System**: Robust version verification across CLI and GUI interfaces
- **Build System**: Clean compilation with zero errors across all platforms
- **Homebrew Integration**: Complete end-to-end Homebrew installation validation

## [0.4.0] - 2025-07-15

### Added
- **Graphical User Interface (GUI)** - Point-and-click interface for easier use
  - System tray integration for desktop monitoring
  - Visual dashboard with instance status and costs
  - Template browser with visual cards and descriptions
  - Storage management with visual indicators
  - Dark and light themes support
- **Package manager distribution** for easier installation
  - Homebrew formula for macOS and Linux
  - Chocolatey package for Windows
  - Conda package for all platforms
- **Multi-architecture support**
  - AMD64 (Intel/AMD) for all platforms
  - ARM64 (Apple Silicon, AWS Graviton) for macOS and Linux
- **Multi-profile foundation** for the upcoming v0.4.2 features
  - Profile management package (`pkg/profile`)
  - Profile switching infrastructure
  - AWS credential provider integration
- **Complete API client with context support**
  - Context-aware API methods for proper timeouts
  - Improved error handling with context propagation
  - Full compatibility with both CLI and GUI clients

### Changed
- Updated API client interfaces to use context support
- Improved documentation with GUI User Guide
- Enhanced error handling with clear user feedback
- Updated build system for multi-architecture support
- Restructured package layout for better distribution

### Fixed
- Compatibility between CLI and GUI components
- API method signatures for proper context handling
- Build system for cross-platform package generation
- Documentation to reflect current features and installation methods

## [0.4.3] - 2025-08-19

### Added
- Template inheritance system with multi-level stacking support
- Comprehensive template validation with 8+ validation rules
- Enhanced build system with cross-compilation fixes
- Complete hibernation ecosystem with cost optimization
- Idle detection system with automated hibernation policies
- Professional GUI interface with system tray integration
- CLI version output consistency with daemon formatting
- EFS multi-instance sharing with cross-template collaboration

### Enhanced
- Version synchronization across all components (CLI, daemon, GUI)
- Cross-compilation support using existing crosscompile build tags
- Template system with stackable inheritance (e.g., Rocky9 + Conda)
- Hibernation policies with intelligent fallback to stop when unsupported
- Cost optimization with session-preserving hibernation capabilities
- GitHub release workflow with automated distribution packages
- Homebrew tap with complete installation testing cycle

### Fixed
- CLI version display format to match daemon professional output
- Cross-compilation keychain errors using platform-specific alternatives
- Template validation preventing invalid package managers and self-reference
- Mock API client version consistency in tests
- Version variable synchronization between Makefile and runtime
- Distribution package checksums and binary verification

### Documentation
- Updated all version references from 0.4.2 to 0.4.3
- Template inheritance and validation technical guides
- Hibernation ecosystem implementation documentation
- Complete release preparation and distribution strategy
- Homebrew tap setup and maintenance procedures
- Windows MSI installer comprehensive documentation

## [Unreleased] - 0.5.0 Multi-User System

### Added
- Secure invitation system with device binding
- Cross-platform keychain integration for secure credential storage
- S3-based registry for tracking authorized devices
- Multi-level permissions model for invitation delegation
- Device management interface in GUI, TUI and CLI
- Administrator utilities for device management
- Batch invitation system for managing multiple invitations at once
- CSV import/export for bulk invitation management
- Concurrent invitation processing with worker pools
- Batch device management for security administration
- Device registry integration for centralized control
- Multi-device revocation and validation tools

### Enhanced
- Profile management with security attributes
- GUI invitation dialog with device binding options
- TUI profile component with security indicators
- CLI invitation commands with security features

### Documentation
- SECURE_PROFILE_IMPLEMENTATION.md with technical details
- SECURE_INVITATION_ARCHITECTURE.md with design documentation
- ADMINISTRATOR_GUIDE.md with security management instructions
- BATCH_INVITATION_GUIDE.md with bulk invitation instructions
- BATCH_DEVICE_MANAGEMENT.md with device security documentation
- Updated comments throughout the codebase

## [0.4.2] - 2025-07-16

### Added
- Multi-profile support for multiple AWS accounts
- Profile-aware client for state isolation
- Invitation-based profile sharing
- Profile switching in GUI, TUI and CLI

### Enhanced
- API client with context support
- Error handling with detailed context information
- Performance optimizations with connection pooling
- GUI interface with profile management

### Documentation
- Profile export/import documentation
- Multi-profile guide with technical details

## [Unreleased] - 0.4.0 Development

### Added
- Redesigned Terminal User Interface (TUI) for improved visual management
  - Dashboard view with instance status and cost monitoring
  - Template browser with detailed template information
  - Interactive instance management interface
  - System status monitoring and notifications
  - Visual storage and volume management
  - Keyboard shortcuts for common operations
- Integration with new Prism API context-aware methods
- Consistent help system with keyboard shortcut reference
- Better terminal compatibility across platforms
- Tab-based navigation between sections
- Progressive disclosure of advanced features

### Changed
- Updated API client interface to use context support
- Improved TUI components with active/inactive state handling
- Enhanced error handling with clear user feedback
- Updated Bubbles and BubbleTea dependencies to latest versions
- More consistent user experience between CLI and TUI

### Fixed
- Fixed spinner rendering issues during API operations
- Improved terminal compatibility with various terminal emulators
- Better error messages for API connection failures