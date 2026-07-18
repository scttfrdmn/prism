# Prism landing page

A self-contained static marketing landing page for **prismcloud.host**, in the
style of [spore.host](https://spore.host): dark-mode hero, feature-pill badges,
feature-card grid, tabbed install, and a steps section — reusing Prism's indigo
palette and prism-icon.

## Files

- `index.html` — the entire page (inline CSS + vanilla JS; no build step, no deps)
- `assets/prism-icon.png` — logo / favicon

## Preview locally

Just open it — no server needed:

```bash
open landing/index.html          # macOS
# or serve it:
python3 -m http.server -d landing 8000   # then visit http://localhost:8000
```

## Deploying to prismcloud.host

The page is a single static file, so it works on any static host (Netlify,
Cloudflare Pages, S3 + CloudFront, GitHub Pages, etc.). Point the apex domain at
your chosen host and upload `landing/`.

### Domain topology

`prismcloud.host` does triple duty (same pattern as spore.host):

- **apex `prismcloud.host`** → this landing page (static host)
- **`docs.prismcloud.host`** → the MkDocs documentation site (`mkdocs.yml`)
- **`*.prismcloud.host`** → live workspace DNS, managed by Route 53; spored
  self-registers each instance as `<name>.<account-base36>.prismcloud.host`

Because the apex serves the landing and `*` serves workspace DNS, keep the apex
A/ALIAS record pointed at the static host and let the wildcard resolve through
Route 53 — they don't collide.

The docs site currently ships its own `docs/CNAME` (`prismcloud.host`); when the
landing takes the apex, move the docs to the `docs.` subdomain and update
`site_url` in `mkdocs.yml` accordingly.

## Editing

All copy lives directly in `index.html`. Keep any `prism ...` commands accurate
against the current CLI (`prism workspace launch|list|...`, `prism profiles`,
`aws login`).
