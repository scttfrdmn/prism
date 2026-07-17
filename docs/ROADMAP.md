# Prism Roadmap

Prism is under active development. The **authoritative, always-current** view of
planned work lives on GitHub:

- **[Milestones](https://github.com/scttfrdmn/prism/milestones)** — grouped, dated goals
- **[Issues](https://github.com/scttfrdmn/prism/issues)** — individual features and fixes
- **[CHANGELOG.md](https://github.com/scttfrdmn/prism/blob/main/CHANGELOG.md)** — what has already shipped, per release

This page gives a high-level sense of direction; it intentionally does **not**
duplicate the issue tracker (which drifts out of date the moment it's copied).

## Current direction

- **spore.host foundation** — Prism's instance launch, lifecycle, and pricing are
  built on the [spore.host](https://spore.host) toolchain (spawn for
  launch/lifecycle, truffle for pricing) rather than hand-rolled AWS calls. This
  includes multi-instance workloads: **job arrays** and **parameter sweeps**.
- **Cost control** — budget engine with live enforcement, per-project limits,
  idle detection and hibernation, spot pricing.
- **Collaboration** — projects, research users, invitations, courses, and
  workshops for labs and classrooms.
- **Two interfaces** — a scriptable CLI and a desktop GUI over one daemon.

## Proposing or tracking work

Open an [issue](https://github.com/scttfrdmn/prism/issues/new) to request a
feature or report a bug. Larger efforts are organized under
[milestones](https://github.com/scttfrdmn/prism/milestones).
