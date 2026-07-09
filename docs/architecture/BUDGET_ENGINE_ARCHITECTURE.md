# Budget Engine Architecture

**Document purpose**: Specify a standalone, embeddable budget engine — the conceptual model and
the library architecture — that Prism (and other tools) can adopt. This is a design record, not an
implementation. No code has been written against it yet.

**Relationship to existing docs**:
- [BUDGET_PHILOSOPHY.md](../BUDGET_PHILOSOPHY.md) — the user-facing *organizational* model
  (Pools → Allocations → Projects, v0.5.10). Still valid; this doc generalizes it.
- [BUDGET_BANKING_PHILOSOPHY.md](../BUDGET_BANKING_PHILOSOPHY.md) — the user-facing *temporal*
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
2. **Convergence.** `prism-research-portal` (prp) already persists budget and spend as separate
   seam-backed record types (`BudgetStore`/`SpendStore`). A shared engine over shared record shapes
   lets a desktop tool and the web portal fold the *same* records into the *same* answer — the
   whole point of the persistence seam.

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
  policy/
    sourcing.go   # SourcingPolicy interface + built-ins
    pacing.go     # PacingPolicy interface + built-ins
    projection.go # ProjectionPolicy interface + built-ins
  ports.go        # EventStore, Clock, ActionSink interfaces (host-supplied)
```

### 5.3 The ports (host-supplied seams)

```go
// Persistence: the host owns storage. A seam-backed adapter satisfies this trivially.
type EventStore interface {
    AppendSpend(ctx, scope, SpendEvent) error
    AppendPlan(ctx, scope, PlanEvent) error
    Spend(ctx, scope) ([]SpendEvent, error)   // ordered
    Plan(ctx, scope)  ([]PlanEvent, error)     // ordered
}

// Enforcement: the engine emits an intent; the host performs the effect.
// Mirrors today's already-decoupled ActionExecutor (budget never learns how to hibernate).
type ActionSink interface {
    OnState(ctx, scope, BurnState) error       // alerts, hibernate, throttle — host's call
}

type Clock interface { Now() time.Time }        // injectable for determinism/tests
```

`scope` is the seam `Scope` (tenant/pi/grant/project + account). The engine is scope-parameterized
but scope-agnostic — see §6.

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

### 5.5 How a host adopts it (Prism as the reference consumer)
- Implement `EventStore` over the persistence seam (`seam.Store[SpendEvent]`,
  `seam.Store[PlanEvent]`) — Prism already has the seam.
- Implement `ActionSink` to fan out to Prism's existing alerting + the `ActionExecutor`
  (hibernate-all, throttle).
- Provide a real `Clock`.
- Everything else — pools, allocations, sources, the three policies — is engine-internal.

Project identity stays in Prism. The engine only ever sees `ProjectID` / `AllocationID` strings.

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

---

## 7. Migration & convergence (informational)

- **From Prism today**: `BudgetManager` is already seam-backed and nearly liftable; `BudgetTracker`
  uses a separate JSON persistence and is owned by `project.Manager` at ~10 call sites. Adopting
  this engine means (a) unifying Tracking onto the event log, (b) relocating budget types out of the
  shared `pkg/types`, (c) converting the two concrete back-references (`Manager`↔tracker,
  budget→project) to interfaces, (d) moving webhook delivery behind the existing `alerting`
  dispatcher. Only `pkg/daemon` consumes budget, so the blast radius is one package.
- **prp convergence**: adopt prp's `CostLineItem` as the `SpendEvent` shape and align the budget
  record shapes so both clients fold identical logs. Then Budget and Spend become two more
  byte-identical shared-state domains alongside the ones already on the seam.

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

### Still open
3. **Checkpointing** — do hosts persist a folded-state checkpoint for performance, and if so how is
   it invalidated on out-of-order event arrival? (Log stays authoritative regardless.)
4. **prp record parity now vs. later** — adopt prp's `CostLineItem`/budget shapes as the target from
   day one, or refactor Prism-internally and reconcile as a second step.

---

## 9. Summary

One conserved quantity (`remaining_principal ≥ 0`), moved in time, never negative. State
event-sourced from two logs (actuals + plan mutations) so it is derived and reproducible. A small
memoryless controller at the core; all richness in **three orthogonal policy axes** (Sourcing,
Pacing, Projection) plus a signed pace accumulator for banking/borrowing. Packaged as a
host-agnostic library with three injected ports (`EventStore`, `ActionSink`, `Clock`), consumed by
Prism over its persistence seam and converging with prp on shared record shapes.
