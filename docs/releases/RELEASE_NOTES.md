# Release Notes

For the complete changelog with all details, see [CHANGELOG.md on GitHub](https://github.com/scttfrdmn/prism/blob/main/CHANGELOG.md).

---

## Version history

| Version | Date | Highlights |
|---------|------|------------|
| [v0.37.0](#v0370) | 2026-10-13 | Security hardening release; job arrays, parameter sweeps, completion signaling; spend-ledger fixes |
| [v0.36.1](#v0361) | 2026-07-08 | Security patch: Go 1.26.5 (GO-2026-5856); Wails v3 alpha.102 |
| [v0.36.0](#v0360) | 2026-07-07 | `workspace cost --billed`, persistence seam with DynamoDB backend |
| [v0.35.5](#v0355) | 2026-05-05 | Correct lifetime spend and burn rate in `workspace cost`; update-checker fixes |
| [v0.35.4](#v0354) | 2026-04-20 | Documentation overhaul |
| [v0.35.3](#v0353) | 2026-04-09 | Security hardening: proxy validation, shell quoting, CSP/CORS, API path encoding |
| [v0.35.2](#v0352) | 2026-04-09 | Security: spored checksum, S3 encryption, TLS daemon support, proxy CSP |
| [v0.35.1](#v0351) | 2026-04-09 | Security: shell injection defense, file permissions, localhost binding, auto API key |
| [v0.35.0](#v0350) | 2026-04-09 | spored idle detection, TTL countdown/safety valve/extend action |
| [v0.34.0](#v0340) | 2026-04-09 | spored daemon on all instances, prismcloud.host DNS, `--dns`/`--ttl` flags |
| [v0.33.3](#v0333) | 2026-04-09 | Governance quota validation, dead route cleanup |
| [v0.33.2](#v0332) | 2026-04-09 | LogsView real API, tab arrow keys, settings version, contrast fix |
| [v0.33.1](#v0331) | 2026-04-09 | Tailwind grid fix, modal focus trap, reduced motion, theme tokens |
| [v0.33.0](#v0330) | 2026-04-09 | WCAG 1.4.3 contrast, error flash guard, FormField aria-describedby |
| [v0.32.0](#v0320) | 2026-04-07 | Atkinson Hyperlegible font, Warm Slate palette, Framer Motion animations |
| [v0.31.0](#v0310) | 2026-04-06 | shadcn/ui migration, sidebar rebuilt, Sonner toasts, Cloudscape removed |
| [v0.30.0](#v0300) | 2026-04-02 | Removed TUI; CLI + GUI cover all use cases |
| [v0.29.2](#v0292) | 2026-04-02 | WAI-ARIA accessibility fixes across GUI and landing page |
| [v0.29.1](#v0291) | 2026-04-02 | Project deletion now blocks when active instances exist |
| [v0.29.0](#v0290) | 2026-04-02 | File operations (`workspace files push/pull/list`), Substrate integration tests |
| [v0.28.1](#v0281) | 2026-04-02 | AMI, Marketplace, Idle Detection E2E tests |
| [v0.28.0](#v0280) | 2026-04-01 | Cloudscape 3.0.1255 upgrade, Windows GUID, semver constraints |

---

## v0.37.0

**Released**: 2026-10-13

### ⚠️ Action required on headless machines
- **Set `PRISM_CREDENTIALS_PASSPHRASE` to save profile credentials.** On a host with no OS keychain (for example a Linux server without a Secret Service), saving profile credentials now fails unless `PRISM_CREDENTIALS_PASSPHRASE` is set; the passphrase encrypts the `~/.prism/credentials/*.creds` file. Credentials saved before this release remain readable, and are re-encrypted once the variable is set. Machines with a keychain (macOS, Windows, desktop Linux) need no change. (#738)

### Security
- **Cross-site WebSocket hijacking closed**: the GUI terminal server (no other authentication) accepted any origin, so any web page could open a shell on your workstations; the daemon's SSH proxy accepted lookalike origins such as `localhost.evil.com`. Only local origins and the GUI webview are accepted now. (#737)
- **Credential files really encrypted**: the fallback `.creds` file key was derivable from public data; it's now a keychain-held random key or your passphrase. (#738)
- **SSH host keys verified**: ssh no longer disables host-key checking; keys are recorded on first use, checked afterwards, and bound to the instance ID (EC2 reuses IPs). (#739)
- **`workspace connect` runs no shell**: ssh is invoked with an argument list; the GUI's open-terminal validates and quotes. (#740)
- **SSRF in the AWS console proxy**, **zip slip and decompression bomb on profile import**, **path traversal** through invitation tokens and research usernames, **TLS pinning skipped on resumed sessions**, and several **command-injection** paths fixed. Policy evaluation fails closed on malformed policies, and files under `~/.prism` are created owner-only. (#737)
- Go 1.26.9 and patched x/crypto, x/net, x/mod, grpc and otel clear all govulncheck and Trivy findings; every security scanner now runs on every build. (#698, #719)

### Added
- **Job arrays** (`--count`) and **parameter sweeps** (`--param-file`): launch many workspaces from one request.
- **Completion signaling** (`--on-complete`, `--completion-file`, `--completion-delay`): workspaces can stop, hibernate or terminate themselves when work finishes; also in the GUI launch form.
- **`prism security`** command tree (status, health, compliance reports) is now reachable.

### Fixed
- **Spend accuracy**: the budget ledger and cost estimate no longer bill stopped or hibernated workspaces as running when a stop went unrecorded; stops and starts made outside Prism are now detected; `workspace cost --billed` labels region-wide figures. (#715, #716, #718, #720, #722)
- **Refreshes no longer drop workspace settings** (region, volumes, idle/always-on, TTL and more). (#721)
- **`--dry-run`** no longer claims a launch, leaves a phantom workspace, or uses up launch rate limits — in the CLI or the GUI. (#703, #704, #725)
- **`prism storage upload/download`** and **applying templates over SSH** work again. (#729)
- **`prism ami`** commands check mapped AMIs with AWS and can no longer crash the daemon. (#730)

### Changed
- S3 transfers use AWS's `feature/s3/transfermanager`; AWS SDK and other dependencies updated.

### Removed
- About 1,350 lines of unused code, including the never-registered `prism uninstall` command (the uninstall scripts remain).

---

## v0.36.1

**Released**: 2026-07-08

### Security
- **Go 1.26.5** clears GO-2026-5856, an Encrypted Client Hello privacy leak in `crypto/tls` reachable from the daemon HTTPS server, web proxy and API client.

### Changed
- Wails v3 `alpha.74` → `alpha.102` for the GUI (no application changes; fewer transitive dependencies).

---

## v0.36.0

**Released**: 2026-07-07

### Added
- **`prism workspace cost --billed`**: AWS-billed cost from Cost Explorer next to Prism's estimate.
- **Persistence seam** (`pkg/seam`) with a **DynamoDB backend**; approvals, projects, budgets and RBAC persist through it.

### Security
- x/crypto, x/net and go-git updates; `npm audit` findings cleared.

---

## v0.35.5

**Released**: 2026-05-05

### Fixed
- `prism workspace cost` reports correct lifetime spend and per-minute burn rate; the `RUNNING` column is renamed `AGE`.
- The update checker no longer replays a stale current version after an upgrade, and compares versions numerically.

---

## v0.35.4

**Released**: 2026-04-20

### Documentation
- Documentation overhaul: new user guides (workspace lifecycle, courses, workshops, approvals), architecture docs, CLI reference updates, and stale command and version references fixed across 120+ files.

---

## v0.35.3

**Released**: 2026-04-09

### Security
- **Port validation in proxy handlers**: DCV and web proxy `?port=` parameter now validated with numeric range check (1–65535). Rejects non-numeric and out-of-range values with 400. (#601)
- **Shell quoting in research provisioner**: all username interpolations in SSH commands wrapped in `shellQuote()` using POSIX single-quote escaping. Defense-in-depth against command injection. (#602)
- **Web proxy CSP/CORS fixed**: replaced stripped CSP headers with `frame-ancestors 'self'`. CORS restricted to localhost. Removed `Allow-Credentials`. (#603)
- **Federation token validation**: regex validates AWS federation token format before use in redirect URL. (#603)
- **API path encoding**: `encodeURIComponent()` applied to all 133 path parameters in frontend API client. Prevents path traversal via special characters in resource names. (#604)
- **File permissions hardened**: workshop, course, pricing state files changed from 0644 to 0600. (#605)
- **Content-Security-Policy meta tag**: added to `index.html`, restricting default-src to self. (#605)

---

## v0.35.2

**Released**: 2026-04-09

### Security
- **spored binary checksum verification**: UserData now downloads SHA256 checksum alongside binary and verifies with `sha256sum -c` before execution. Binary deleted if verification fails. (#591)
- **S3 temp file encryption**: PutObject calls for SSM file transfers now specify `ServerSideEncryption: AES256`. (#598)
- **Proxy CSP/CORS**: replaced stripped X-Frame-Options/CSP with `SAMEORIGIN` + `frame-ancestors 'self'` in all proxy handlers. (#596)
- **AppleScript injection defense**: backslashes and double quotes escaped in SSH commands before AppleScript interpolation on macOS. (#598)
- **Auth status endpoint hardened**: unauthenticated `GET /api/v1/auth` returns minimal `{"status":"ok"}` instead of leaking auth configuration. (#598)
- **Security response headers**: all API responses include `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`. (#599)
- **Path traversal defense**: file operations handler normalizes paths with `filepath.Clean()` and rejects non-absolute paths. (#598)
- **TLS support**: daemon uses `ListenAndServeTLS` when `~/.prism/tls/cert.pem` and `key.pem` exist. (#594)

---

## v0.35.1

**Released**: 2026-04-09

### Security
- **Shell injection defense**: `validateRemotePath()` rejects shell metacharacters and path traversal sequences before interpolating paths into SSM bash scripts. (#589)
- **File permissions tightened**: state directory `0755`→`0700`; all state/config/backup file writes `0644`→`0600`. (#590)
- **Localhost-only binding**: daemon binds to `127.0.0.1` instead of `0.0.0.0` — prevents external network access. (#593)
- **Request body size limit**: `http.MaxBytesHandler(100MB)` prevents memory exhaustion from oversized POST bodies. (#592)
- **Auto-generate API key**: daemon generates a 256-bit API key on first start if none configured. (#597)
- **Test mode startup warning**: prominent log warning when `PRISM_TEST_MODE=true` is active. (#595)

### Fixed
- **WCAG 2.1 AA compliance** (ADA Title II): `<main>` landmark, muted foreground contrast 5.5:1, Link keyboard activation (Enter/Space), Input autocomplete, table row ARIA fixes. (#600)

---

## v0.35.0

**Released**: 2026-04-09

### Changed
- **Idle detection delegated to spored**: daemon-side CloudWatch polling removed. Idle detection now runs on-instance via spored with 7 signals (CPU, network, disk I/O, GPU, terminals, users, recent activity). Fleet-level policy templates remain as directives to spored. (#588)

### Added
- **TTL countdown column**: instance table shows time remaining with color-coded status (green >2h, yellow <2h, red <1h). (#588)
- **TTL safety valve**: daemon state monitor stops running instances past their ExpiresAt as a fallback. (#588)
- **Extend Time action**: "Extend Time (+4h)" in instance dropdown for TTL-enabled workspaces. (#588)

---

## v0.34.0

**Released**: 2026-04-09

### Added
- **spored daemon on every instance**: all new workspaces install the spored daemon from S3. Runs as a systemd service providing on-instance idle detection, DNS registration, TTL enforcement, and spot interruption handling. (#588)
- **Dynamic DNS — prismcloud.host**: workspaces automatically register `{name}.{account}.prismcloud.host` A records. Hostname shown in instance table and used in SSH connect dialog.
- **`--dns` flag**: `prism workspace launch ... --dns my-name` — custom DNS record name (defaults to sanitized workspace name).
- **`--ttl` flag**: `prism workspace launch ... --ttl 8h` — auto-stop after duration. spored warns users 5 minutes before expiration.
- **GUI launch form**: DNS Name and Time Limit fields in workspace launch modal.
- **Hostname column**: instance table shows `{name}.{account}.prismcloud.host`.

---

## v0.33.3

**Released**: 2026-04-09

### Fixed
- **GovernancePanel quota validation**: numeric fields now validated before submit; non-numeric input shows error instead of silently coercing to unlimited.
- **Dead route removed**: `project-detail` activeView rendered a placeholder but was unreachable from navigation. Removed.

---

## v0.33.2

**Released**: 2026-04-09

### Fixed
- **LogsView**: replaced mock log data with real `GET /api/v1/logs/{instance}` API calls.
- **Tab arrow-key navigation**: Left/Right arrow keys cycle tabs; Home/End jump to first/last. WAI-ARIA roving tabindex pattern.
- **Settings version**: replaced hardcoded `v0.5.1` with dynamic import from `package.json`.
- **Destructive color contrast**: `hsl(0 84% 60%)` → `hsl(0 84% 50%)` — WCAG AA compliant.

---

## v0.33.1

**Released**: 2026-04-09

### Fixed
- **Tailwind JIT**: replaced 7 dynamic template-literal class names with static lookup objects — fixes grid/column layouts that silently rendered as 0-width.
- **Modal focus trap**: Radix `FocusScope` with Escape key handler and `autoFocus` on close button (WCAG 2.4.3).
- **prefers-reduced-motion**: global CSS media query kills all animation/transition durations for users with motion sensitivity.
- **Theme token colors**: `--warning`/`--success` CSS custom properties added; hardcoded Tailwind colors replaced; focus rings use `var(--ring)`.

---

## v0.33.0

**Released**: 2026-04-09

### Fixed
- **WCAG 1.4.3 contrast**: `--primary` darkened to `hsl(174 77% 26%)` — white-on-teal 3.48:1 → ~5.2:1. Focus ring now 7.3–8.1:1.
- **Error flash guard**: `!loading &&` guard on 12 error Alerts prevents red flash when daemon is slow to respond on first mount.
- **FormField accessibility**: description, constraint, and error text linked via `aria-describedby`; `aria-invalid` and `role="alert"` on errors.

### Changed
- **Dashboard density**: compact inline status bar replaces 3-column stats grid and duplicate Quick Actions.
- **Settings responsive nav**: horizontal tab bar below 1024px viewport width.

---

## v0.32.0

**Released**: 2026-04-07

### Added
- **Framer Motion animations**: view slide transitions, modal spring animation, card stagger entrance.

### Changed
- **Font**: Inter → Atkinson Hyperlegible Next (UI) + Atkinson Hyperlegible Mono (code) — designed for accessibility and legibility.
- **Color palette**: Warm Slate — teal primary, warm white background, stone neutrals. All 34 CSS custom properties updated.
- **Toast position**: Sonner toaster moved to bottom-right.

---

## v0.31.0

**Released**: 2026-04-06

### Changed
- **GUI framework migration**: Cloudscape removed. Rebuilt with shadcn/ui (Radix UI primitives + Tailwind CSS v4), Sonner toasts, and Lucide icons.
- **Sidebar rebuilt**: all 23 nav items preserved with shadcn `Sidebar` component; badges for pending approvals, courses, workshops, instances, templates.
- **13 views extracted** from monolithic `App.tsx` to top-level components, fixing React re-mount performance issue.
- **Toast system**: all `addNotification` patterns replaced with `toast.success/error/warning`.

---

## v0.30.0

**Released**: 2026-04-02

### Removed
- **TUI removed** — the `prism tui` command and `internal/tui/` package (~22,000 lines) have been retired. The TUI had fallen behind the CLI and GUI in feature coverage (missing file operations, governance, courses, and all features added after v0.26.0), was excluded from CI, and added maintenance overhead. All use cases are covered by the CLI and GUI. Removes `charmbracelet/bubbletea`, `bubbles`, and `lipgloss` dependencies.

---

## v0.29.2

**Released**: 2026-04-02

### Fixed
- WAI-ARIA accessibility across GUI and landing page (#568–577)
- `Terminal.tsx`: `role="status"`, `aria-live`, `aria-hidden` on status bar and pulsing dot; `role="application"` + `aria-label` on terminal container
- `SSHKeyModal.tsx`: `aria-label` and `aria-readonly` on key textareas
- `WebView.tsx`: `role="status"` + `aria-label` on loading overlay
- `App.tsx`: skip-navigation link; `tabIndex={-1}` on `#main-content`
- Landing page: `<main>` landmark, `<section aria-labelledby>`, `<article>` cards, corrected heading hierarchy, `aria-hidden` on decorative emoji
- Landing page CSS: `:focus-visible` outlines on all hero buttons

---

## v0.29.1

**Released**: 2026-04-02

### Fixed
- Project deletion now correctly blocks when running instances belong to the project (#539). Previously `getActiveInstancesForProject` was a permanent stub returning empty — deletion safety was silently bypassed.

---

## v0.29.0

**Released**: 2026-04-02

### Added
- **File operations**: `prism workspace files push/pull/list` — transfers files to/from running instances via S3 relay (#30a)
- Substrate v0.48.0 — all 6 Substrate integration tests pass (replaces LocalStack, no Docker required)

### Fixed
- All 268 pre-existing TypeScript typecheck errors resolved (`npm run typecheck` now clean)

---

## v0.28.1

**Released**: 2026-04-02

### Added
- E2E test coverage for AMI Management, Template Marketplace, and Idle Detection
- `AMIPage`, `MarketplacePage`, `IdlePage` Playwright page objects

### Fixed
- `WorkshopsPanel.tsx`: added missing `data-testid` that was causing E2E timeout cascade
- `course_handlers.go`: EFS creation now returns deterministic fake ID in test mode (was hanging indefinitely)
- `playwright.config.js`: added `NODE_OPTIONS="--max-old-space-size=4096"` to prevent Vite crash on large App.tsx

---

## v0.28.0

**Released**: 2026-04-01

### Added / Changed
- Cloudscape Design System upgrade to 3.0.1255
- Windows GUID support
- Semver constraint enforcement
- Policy framework fixes
- Research user migration improvements
