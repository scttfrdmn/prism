# Budget Engine Architecture

**Document purpose**: Specify a standalone, embeddable budget engine — the conceptual model and
the library architecture — that Prism (and other tools) can adopt. This is a design record, not an
implementation. No code has been written against it yet.

**Relationship to existing docs**:
- [BUDGET_PHILOSOPHY.md](../BUDGET_PHILOSOPHY.md) — the user-facing *organizational* model
  (Pools → Allocations → Projects, v0.5.10). Still valid; this doc generalizes it.
- [BUDGET_PHILOSOPHY.md](../BUDGET_PHILOSOPHY.md) — the user-facing *temporal*
  model (surplus/banking). Still valid; this doc gives it a formal engine and adds borrowing,
  multi-source dated funding, and the projection choice.

This doc supersedes neither; it is the architecture beneath both, factored so it can live outside
Prism.

---

## 1. Why a separate engine

Budget logic in Prism today lives inside `pkg/project` and is entangled with project identity and
with a daemon-owned spend tracker (see the coupling map in the extraction analysis). Two forces make
a clean, standalone engine worthwhile:

1. **Reuse.** The model below is not Prism-specific. Any tool that meters cloud spend against
   time-boxed funding wants it. It should be adoptable as a library with no dependency on Prism.
2. **Convergence.** `prism-research-portal` (prp) is a second host — a web portal with its own early
   budget sketch (`BudgetStore`/`SpendStore`/`CheckLaunch`), not yet shipping. Rather than maintain
   two budget brains, both Prism and prp adopt **this** engine and its record shapes, so a desktop
   tool and the web portal fold the *same* records into the *same* answer — the whole point of the
   persistence seam. The engine defines the records; prp's sketch is replaced (§7).

---

## 2. Three elements, by their essential nature

The domain is not "budget stuff." It is three elements with different shapes, lifecycles, and
writers. Keeping them distinct is the central design decision.

| Element | Question it answers | Writer | Shape |
|---|---|---|---|
| **Project** (identity) | *What work, by whom?* | People, occasionally | A record. Out of scope for this engine — referenced only by ID. |
| **Budget Management** (the plan) | *How much is authorized, to whom, over what time?* | Admins/PIs, rarely | Window + dated funding sources + allocations. **Transactional.** |
| **Tracking** (the actuals) | *How much spent, at what rate, and what do we do?* | Machines, constantly | An append-only ledger of spend events. **Observational + reactive.** |

**Management is the *plan*; Tracking is the *ledger*; comparing them is *variance*.** That is the
oldest distinction in accounting, and it dictates the boundaries:

- A **budget pool spans multiple projects** (one pool → many allocations → many projects), so a
  pool cannot live *inside* the Project aggregate. The current "Project owns budget" ownership is
  inverted; the correct shape is three **peers under a shared scope**, related by ID, composed by a
  thin facade above them. This matches prp's handler, which is *given* both a spend source and a
  budget source and joins them — neither owns the other.
- **Spend is owned by Tracking, period.** Management must not cache `SpentAmount`; it reads spend
  through an interface and computes variance at read time. Two caches of "how much was spent" drift;
  one ledger does not.

---

## 3. The conceptual model

### 3.1 One conserved quantity, moved in time

The core invariant:

```
remaining_principal(t) = Σ funding_sources_active(t) − cumulative_spend(t)  ≥ 0
```

`remaining_principal` is **real money**, hard-floored at **zero**. Everything else is a *view* over
it. The instantaneous sustainable pace is a derived quantity, never stored:

```
sustainable_rate(t) = remaining_principal(t) / remaining_time(t)
```

**Banking and borrowing both move spend in *time*, never in *amount*.** Banking pushes spend later;
borrowing pulls it earlier. You never spend money you don't have — borrowing is pulling *your own*
remaining budget forward, not going negative. Solvency is structural: the ledger fold cannot produce
a spend that takes `remaining_principal` below zero, so `$0` is a wall, not a checked limit. Once you
hit zero you are **done** — only adding a funding source (which lifts `remaining_principal`) revives
the rate.

### 3.2 Memory is required — the model is not memoryless

A pure `remaining/time_left` controller is *forgetful*: it silently absorbs every deviation into a
new floating rate, so it can never (a) return you to your *original* pace after a detour, nor
(b) let you reserve an accrued amount for a planned burst. Both require remembering two things:

1. a **nominal reference** — the pace you're supposed to be at (a time-varying curve; see §3.3), and
2. a **signed pace accumulator** — how far *ahead* (banked) or *behind* (borrowed) that reference
   you currently are.

These are **two independent scalars with different floors**:

- `remaining_principal` — dollars left; floor **0**; answers *"am I out of money?"*
- `pace_deviation` — signed dollars ahead-of / behind-schedule vs. nominal; answers *"am I pacing
  right?"* Positive = banked (burst room earned by idling). Negative = borrowed (spent ahead; repaid
  by idling, which restores you toward nominal).

You can be behind pace (borrowed) yet solvent (money remains). That is legitimate borrowing, and it
self-limits because it draws `remaining_principal`, which floors at zero.

### 3.3 The nominal is a mutable, time-varying, possibly-disjoint curve

Multi-source funding means the reference pace is not a line. It is a curve that:

- **steps up** when a source activates, **steps down** when one expires;
- can be **disjoint** — a gap between a source ending in June and one starting in September drives
  nominal to **zero** in July–August (→ to run through a gap you *must* have banked; this is why
  banking is load-bearing, not cosmetic);
- is **editable in flight** — add a source, extend the window, amend an allocation.

Because the plan mutates, answering "am I ahead or behind" requires the nominal *as it was at each
moment*. Therefore the plan itself is **event-sourced history**, not a static record.

### 3.4 Event sourcing (decided)

State is folded from **two logs**:

1. **Actuals** — spend line items (converges with prp's `CostLineItem`).
2. **Plan mutations** — `SourceAdded`, `SourceExpired`, `WindowExtended`, `AllocationChanged`, …

```
pace_deviation(t) = ∫ (nominal_from_plan_history(τ) − spend_from_actuals(τ)) dτ,  0 ≤ τ ≤ t
remaining_principal(t) = Σ sources_from_plan_history(t) − Σ spend_from_actuals(t)
```

The bank/borrow state is **derived and reproducible** from shared records — not a checkpointed
scalar that can drift between two clients. This is the same discipline that makes seam records "dumb
and shareable": Prism and prp fold the same two logs → identical state. (A checkpoint may be cached
as an optimization, but the log is authoritative.)

**Plan-edit semantics (decided): causal, go-forward only.** Adding a source or extending the window
reshapes the *future* nominal and lifts `remaining_principal`; it does **not** retroactively
recompute past nominal or forgive already-accrued bank/borrow. Money already banked and pace already
borrowed ride through a plan edit untouched. (The alternative — retroactive "true-up," e.g.
"we always had until December, recompute as if" — is explicitly *not* the default.)

---

## 4. The three policy axes (the flexibility surface)

All variation lives in three **orthogonal, composable** policy axes over the engine core. New
funding models become new implementations on one axis; nothing else changes.

### Axis 1 — Sourcing: how multiple dated sources combine and drain
Sources are **dumb, dated capacity** (`amount`, `[start, end]`). All expiry/priority logic lives in
the policy: *expiry-first* (drain a source before it lapses), *priority order*, *proportional*, …
Keeping sources dumb keeps the persisted records boring and shareable.

### Axis 2 — Pacing: how idle allowance is treated
When nothing is running (cloud spend only accrues while something runs), the allowance you *could*
have spent goes somewhere:
- **rate-adjust** — nothing is reserved; idling simply lets the recomputed rate float up later
  (memoryless behavior).
- **bank-and-reserve** — hold the nominal rate, accrue the unspent allowance as banked
  `pace_deviation` for a later deliberate burst. This is the only stateful pacing mode, and it is
  *mandatory* when funding is disjoint (§3.3).

### Axis 3 — Projection: which variable is pinned when you deviate
`rate = remaining / time_left` — when you deviate, one variable is free and one is pinned:
- **fixed-date / floating-rate** (`RateTargetPolicy`) — the window end is sacred (a grant that must
  last to its end date). Bursting lowers the go-forward rate. Setpoint = "land at zero *on* the
  date." Alert = "sustainable rate dropped to $X."
- **fixed-rate / floating-date** (`DeadlineFloatPolicy`) — the chosen rate is sacred (a cloud budget:
  "spend how I want, tell me when I'm broke"). Bursting pulls the zero-date earlier. Zero is an
  *outcome to forecast*, not a target. Alert = "projected broke by Oct 3."

The math produces both readouts from the same state for free; **Projection is the per-budget policy
choice of which one enforcement/alerting acts on.**

### Worked compositions
- **Multi-month grant**: `expiry-first × bank-and-reserve × fixed-date`.
- **Monthly cloud budget**: `single-source × rate-adjust × floating-date`.
- **Disjoint two-source grant with a summer gap + planned fall GPU burst**:
  `expiry-first × bank-and-reserve × fixed-date` — banking across the gap is what makes the fall
  burst possible.

---

## 5. Library architecture

### 5.1 Goals
- **Zero host coupling.** No dependency on Prism, AWS SDKs, HTTP, or a specific datastore.
- **Bring-your-own persistence.** The host supplies storage behind a small interface.
- **Bring-your-own enforcement.** The engine decides *whether* to act; the host decides *how*.
- **Deterministic + reproducible.** Same event logs → same state, on any host.

### 5.2 Proposed shape (`budgetengine`, its own module)

```
budgetengine/
  event.go        # SpendEvent, PlanEvent (the two logs) — plain data
  plan.go         # Window, FundingSource, Allocation — plain data, IDs only
  state.go        # Fold: (planLog, spendLog, now) -> EngineState
  engine.go       # Engine.Evaluate(now) -> BurnState   (the pure controller)
  checklaunch.go  # Engine.CheckLaunch(scope, cost) -> Decision   (pre-flight query)
  policy/
    sourcing.go   # SourcingPolicy interface + built-ins
    pacing.go     # PacingPolicy interface + built-ins
    projection.go # ProjectionPolicy interface + built-ins
  ports.go        # read/write persistence, Clock, ActionSink interfaces (host-supplied)
```

### 5.2a Two interaction shapes: query (pull) and react (push)

The engine is used two independent ways; a host may use either or both:

- **Query — pre-flight, synchronous.** `CheckLaunch(scope, estimatedCost) → Decision` answers
  *"may this action proceed, and how close to the ceiling is it?"* **before** the host acts. This is
  prp's entire budget model, and it also replaces Prism's daemon "projected > limit → 403" gate.
- **React — post-state, push.** `ActionSink.OnState(scope, BurnState)` fires **after** spend has
  moved. This drives Prism's alerts and hibernate/stop. A read-mostly host (prp) may leave it a
  no-op.

`CheckLaunch` is **budget-only.** It does not absorb non-budget preconditions — prp today bundles an
auto-stop-policy requirement into its launch gate (`engine.GatedLaunch`); that stays host-side. The
engine answers the budget question; the host composes any other gates around it.

### 5.3 The ports (host-supplied seams)

**Persistence is split into read and write**, because one host writes spend and the other does not.
Prism's daemon observes state transitions and *appends* spend; prp's spend is written by an external
collector/CUR pipeline (not prp's code), and prp only *reads* it. A single mandatory-append port
would make the read-only host a degenerate/error case; splitting makes it first-class.

```go
// READ — both hosts implement. The engine only ever needs to read to evaluate/check.
type SpendSource interface { Spend(ctx, scope) ([]SpendEvent, error) } // ordered
type PlanSource  interface { Plan(ctx, scope)  ([]PlanEvent, error)  } // ordered

// WRITE — the appending host implements (Prism). A read-only host (prp) omits these;
// its writes come from the external collector, not from engine calls.
type SpendWriter interface { AppendSpend(ctx, scope, SpendEvent) error }
type PlanWriter  interface { AppendPlan(ctx, scope, PlanEvent) error }

// Enforcement (push): the engine emits state; the host performs any effect.
// Mirrors today's already-decoupled ActionExecutor (budget never learns how to hibernate).
type ActionSink interface {
    OnState(ctx, scope, BurnState) error       // alerts, hibernate, throttle — host's call
}

type Clock interface { Now() time.Time }        // injectable for determinism/tests
```

The engine's evaluation/query path depends only on the **read** ports (`SpendSource`/`PlanSource`)
+ `Clock`. `CheckLaunch` and `Evaluate` need nothing more. Writing and reacting are separate
capabilities a host opts into.

A seam-backed adapter satisfies the read ports trivially: `SpendSource` over
`seam.Store[SpendEvent].List(scope)` — which is exactly prp's existing `SpendStore.Rollup`
(list-by-scope + fold).

`scope` is the seam `Scope` (tenant/pi/grant/project + account) — **byte-identical** in Prism and
prp today (same `Principal`/`Store[T]` contract, verified). The engine is scope-parameterized but
scope-agnostic: it evaluates against whatever single `Scope` it is handed and never climbs the
scope hierarchy itself (§6).

### 5.4 The output

```go
type BurnState struct {
    RemainingPrincipal float64      // ≥ 0, real money
    PaceDeviation      float64      // signed: + banked, − borrowed
    SustainableRate    float64      // fixed-date view
    ProjectedZeroDate  time.Time    // fixed-rate view
    Solvent            bool         // remaining > 0
    OnTrack            bool         // per the active ProjectionPolicy
    // ... per-source breakdown, next nominal step, etc.
}
```

Both projection readouts are always computed; the `ProjectionPolicy` decides which drives `OnTrack`
and what `ActionSink` acts on.

`CheckLaunch` returns a `Decision` — a three-valued verdict plus the numbers that explain it. This
shape is adopted from prp's `budget.Decision` (which the engine replaces), generalized onto the
engine's model:

```go
type Verdict int   // Allow | Warn | Block

type Decision struct {
    Verdict          Verdict
    EstimatedCost    float64   // the action's estimated cost, as checked
    EffectiveBalance float64   // the spendable ceiling for this scope right now
    Spent            float64   // authoritative spend (from SpendSource, never client-supplied)
    Projected        float64   // Spent + EstimatedCost
    Remaining        float64   // headroom before the action (≥ 0)
    Reason           string    // human-facing explanation
}
```

Verdict rules (ordered), mirroring prp so the two hosts decide identically:
1. **Block** — scope is frozen (`LaunchPrevented`, e.g. a grant period ended). Unconditional.
2. **Block** — `Projected > EffectiveBalance` (would exceed the ceiling).
3. **Warn** — `Projected ≥ EffectiveBalance × warnThreshold` (default 0.80): allowed, surfaces headroom.
4. **Allow** — within budget.

The mapping to the engine's model, stated explicitly so it's not lost:

```
EffectiveBalance    = available_to_date          // the paced ceiling right now
headroom            = available_to_date − spent   // grows when you under-spend = banking
remaining_principal = landed − spent  ≥ 0         // hard solvency floor (real money)
pace_deviation      = available_to_date − spent   // signed: + banked / − borrowed
```

> **Reconciliation (build finding, #647).** An earlier draft of this section wrote
> `EffectiveBalance = remaining_principal + min(banked_deviation, cap)`. Implementing it against §9's
> own numbers showed that formula **double-counts the −spent term**: `banked_deviation` is
> `(available_to_date − spent)` and `remaining_principal` is `(landed − spent)`, so summing them
> subtracts spend twice. In a *continuous, never-resetting* budget there is no separate "carried bank"
> pot — that was a discrete-period artifact of prp's per-month model; under-spending simply makes
> `(available_to_date − spent)` larger, and that **is** your banked headroom. So the engine uses
> `EffectiveBalance = available_to_date`, which is what §9's `$100k` reflects. This still
> **generalizes** prp's `PeriodAllocation + min(bankedSurplus, TotalBudget × surplusCapPercent)`: prp's
> single-period allocation is the degenerate one-source, one-window case of the engine's capacity
> curve, where `available_to_date` collapses to that period's allocation. (The optional
> `BankCap`/`BankFraction` knob that would bound how much banked headroom counts toward the ceiling
> is declared on the `Allocation` record but not yet wired into the fold — deferred to a later engine
> version.)

`Spent` is supplied by `SpendSource` (server-authoritative); the engine never accepts a client-supplied
spend. "Frozen scope" is a first-class engine input (a plan-level freeze), distinct from "over budget".

### 5.5 How two very different hosts adopt it

The design is validated by fitting **both** consumers. Prism is a long-lived local daemon that
*writes* spend from observed state; prp is a stateless Lambda behind API-Gateway with cross-account
assume-role, *read-mostly* on spend. If one port set fits both, the boundary is right. It does:

| Port | Prism (daemon) | prp (Lambda) |
|---|---|---|
| `SpendSource` / `PlanSource` (read) | local seam store | DynamoDB seam over Lambda; = prp's existing `SpendStore.Rollup` (list-by-scope + fold) |
| `SpendWriter` / `PlanWriter` (append) | daemon **writes** spend from observed state transitions | **omitted** — spend is written by the external collector/CUR, not prp; prp wires read ports only |
| `CheckLaunch` (query, budget-only) | daemon launch gate — replaces today's `projected > limit → 403` | replaces prp's `budget.CheckLaunch`; host still wraps its own non-budget gate (auto-stop policy) |
| `ActionSink.OnState` (react) | alerts + `ActionExecutor` (hibernate/stop/prevent-launch) | no-op, or a warning in the API response (read-mostly) |
| `Clock` | real — earns its keep for idle/scheduled work | near no-op — prp is clockless; a real wall-clock injected per request |
| `Scope` | zero → per-`Principal` (multi-tenant cloud) | `seam.Principal` from the verified JWT; host resolves most-specific (grant→PI→tenant) *before* calling |

Reading the table by row is the proof: no port needs a host-specific shape it can't express. The
only asymmetry — spend append — is handled by the read/write split (§5.3), not by bending a port.

**Both hosts keep their own non-engine concerns.** Project identity stays in the host; the engine
only ever sees `ProjectID` / `AllocationID` strings. prp keeps cross-account `Directory`/assume-role
and chargeback attribution (§7E). Prism keeps its `ActionExecutor` effects. The engine is the shared
budget brain; the hosts are the bodies.

---

## 6. Scope granularity (a real modeling point)

The three elements do not all live at the same scope level:
- **Project** → scoped by tenant (or tenant/pi).
- **Budget pool** → scoped by the pool's *owner* (pi/grant/tenant) — a notch **above** project,
  because a pool spans projects.
- **Allocation** → lives under the pool's scope; references a project by ID.
- **Spend line item** → scoped **at** project (+ account); references an allocation by ID.

So the three are a **family sharing the seam `Scope` key and infrastructure**, but pools sit higher
in the scope hierarchy than projects and spend sits at/below project+account. They are siblings in a
governance bounded-context, not children of Project.

### Decision — bank/borrow scope: **per-allocation**
The signed `pace_deviation` (bank/borrow) is scoped **per-allocation**: a project banks/borrows
against its *own* allocation, isolated — one lab cannot affect another, and the blast radius of any
over-bursting is a single allocation.

Per-allocation is chosen because it is the **more general** of the two: pool-level behavior is
recoverable as a *composition* of per-allocation state (a pool view sums its allocations' deviations),
whereas the reverse is not — you cannot recover per-allocation isolation from a single shared pool
accumulator. So per-allocation is the primitive; any pool-level "shared burst room" feature is built
on top as an explicit policy that reallocates capacity between allocations, never as a change to
where the accumulator lives.

(There is no cross-project *insolvency* risk in either case, since borrowing never goes below the
pool's real $0 — this is purely a utilization/isolation choice, not a liability one. Per-allocation
simply keeps the isolation guarantee available by default.)

### Hierarchy resolution is the host's job, not the engine's
prp resolves a "most-specific" scope (grant → PI → tenant) in two places: cross-account binding
(`crossaccount.Directory.Resolve`) and spend attribution (`spentForScope`). The engine does **not**
climb this hierarchy. It is scope-*parameterized*: it evaluates against exactly the one `Scope` it is
handed, per call. The host decides which scope that is — prp picks the most-specific bound dimension
before calling; Prism passes its per-Principal (or zero) scope. Keeping hierarchy resolution
host-side keeps the engine's contract a single clean `(scope) → answer`, and means prp's
grant-beats-PI-beats-tenant logic never leaks into engine surface.

---

## 7. Migration & convergence (informational)

- **From Prism today**: `BudgetManager` is already seam-backed and nearly liftable; `BudgetTracker`
  uses a separate JSON persistence and is owned by `project.Manager` at ~10 call sites. Adopting
  this engine means (a) unifying Tracking onto the event log, (b) relocating budget types out of the
  shared `pkg/types`, (c) converting the two concrete back-references (`Manager`↔tracker,
  budget→project) to interfaces, (d) moving webhook delivery behind the existing `alerting`
  dispatcher. Only `pkg/daemon` consumes budget, so the blast radius is one package.
- **prp convergence (direction corrected)**: prp is **not shipping and has no users**, so it is
  *downstream* of this engine, not a contract to converge toward. The engine **defines** the record
  shapes (`SpendEvent`, `PlanEvent`, and the plan/allocation types); prp and Prism both adopt them.
  prp's current `pkg/budget` (`Snapshot`/`CheckLaunch`/`Decision`), `BudgetRecord`, and
  `CostLineItem` are **replaced** by the engine — they were a first sketch, not a foundation. What
  the engine keeps *from* that sketch is the good part: the three-valued `Verdict`, the warn/block
  thresholds, and the `EffectiveBalance` formula (which the engine generalizes to a capacity curve).

### 7E. What stays in each host (the engine does NOT absorb)
- **prp**: cross-account `Directory` / assume-role, JWT→`Principal` federation, and chargeback
  attribution (CUR/collector tagging by `prp:` dimension). The engine consumes an *already-attributed*
  `SpendEvent`; **how** a host attributes a raw cost line to a scope is out of scope. prp's spend
  **writer** is the external collector — prp adopts the engine's read ports only.
- **Prism**: the `ActionExecutor` effects (hibernate/stop/prevent-launch), alerting transport, and
  project identity/lifecycle. Prism supplies the write ports (it observes and appends spend).
- **Both**: they keep their own launch orchestration and any non-budget preconditions; the engine
  answers only the budget question via `CheckLaunch`.

---

## 8. Decisions

### Decided
1. **Bank/borrow scope — per-allocation.** The more general primitive; pool-level behavior composes
   on top. §6.
2. **Engine module home — standalone module from day one** (`github.com/scttfrdmn/budgetengine`),
   not an internal `pkg/budget` extracted later. The engine takes no dependency on Prism; Prism
   consumes it like any third party via the injected ports (§5.3). Building it standalone from the
   start forces the host-agnostic boundary to be real rather than aspirational, and makes adoption
   by other tools (and by prp) a matter of importing the module, not extracting from Prism.
3. **Checkpointing — log is authoritative; a folded-state checkpoint is an optional host-side cache.**
   The engine always *can* recompute state by folding the two logs. A host **may** cache a folded
   checkpoint for performance; if it does, appending a spend/plan event with a timestamp ≤ the
   checkpoint's high-water mark invalidates it (out-of-order arrival → recompute from log). The
   engine's correctness never depends on a checkpoint, so the two hosts can differ freely: prp
   (stateless Lambda) folds per request over a bounded window and caches nothing; Prism (daemon) may
   hold a checkpoint. State stays reproducible either way — the checkpoint is never a second source
   of truth.
4. **prp record parity — engine defines, both hosts adopt.** prp is not shipping; the engine is the
   source of truth for the record shapes, and prp's `pkg/budget`/`BudgetRecord`/`CostLineItem` are
   replaced (§7). Not "converge toward prp"; prp converges onto the engine, same as Prism.

*(All four decisions resolved. Interaction shape — query + react — and the read/write port split
were added in this pass; see §5.2a and §5.3.)*

---

## 9. Worked two-host scenario

One scenario folded through both hosts, to show they produce the **same Decision from the same
records**. A grant with two dated sources, banked pacing, per-allocation, `fixed-date` projection:

- **Plan (event log):** `SourceAdded{A: $60k, Jan 1–Jun 30}`, `SourceAdded{B: $120k, Apr 1–Dec 31}`,
  `Window{Jan 1–Dec 31}`, `Allocation{alloc-1 → project-X}`.
- **Spend (event log):** line items accrued Jan–Sep, folded to `Spent = $95k` for `alloc-1`'s scope.
- **State at Oct 1** (folded): both sources are active (B since April), so far `$100k` of the grant's
  `$180k` is available-to-date under the pacing policy; `Spent = $95k`; the effective ceiling for a
  launch check is `EffectiveBalance = available_to_date = $100k` (the paced ceiling — banking is
  already reflected in it, not added on top; see §5.4). `Remaining = $5k`.

Now a launch estimated at `$9k` arrives. `Projected = Spent + est = $95k + $9k = $104k > $100k`:

- **prp (Lambda):** JWT → `Principal{tenant, pi, grant}`; host resolves the most-specific scope, then
  calls `engine.CheckLaunch(scope, 9_000)` with `SpendSource` = its DynamoDB seam read. Gets
  `Decision{Verdict: Block, Projected: 104_000, EffectiveBalance: 100_000, Remaining: 5_000}`. prp
  maps Block → HTTP 402 with the Decision attached. (Had the estimate been `$4k`, `Projected = $99k`
  would land in the warn band → `Warn`, allowed with a headroom message.) prp then applies its own
  non-budget gate (is an auto-stop policy present?) — separate from the engine.
- **Prism (daemon):** same `engine.CheckLaunch(scope, 9_000)`, `SpendSource` = local seam read. Same
  fold → **identical `Decision{Block, 104_000, 100_000, 5_000}`**. Prism maps Block → refuse the
  launch (replacing today's ad-hoc 403). Separately, its `ActionSink.OnState` fires as spend accrues,
  driving alerts/hibernate.

The point: identical engine call, identical folded inputs, identical `Decision`. The hosts differ
only in *transport* (HTTP 402 vs. daemon refusal), *who writes spend* (external collector vs. daemon),
and *what they do with the verdict* — never in the budget math. That is the boundary working.

---

## 10. Summary

One conserved quantity (`remaining_principal ≥ 0`), moved in time, never negative. State
event-sourced from two logs (actuals + plan mutations) so it is derived and reproducible. A small
memoryless controller at the core; all richness in **three orthogonal policy axes** (Sourcing,
Pacing, Projection) plus a signed pace accumulator for banking/borrowing. Two interaction shapes: a
synchronous **`CheckLaunch`** query (budget-only, three-valued Allow/Warn/Block) and a reactive
**`ActionSink.OnState`** push. Persistence is **split read/write** so a writing host (Prism) and a
read-only host (prp) both fit. Packaged as a host-agnostic module; **the engine defines the records,
and both Prism and prp adopt them** — prp's prior budget code is replaced, not converged toward. All
four design decisions are resolved.
