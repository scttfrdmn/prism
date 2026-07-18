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

Prism follows the **same AWS pattern spore.host uses** (S3 static website +
CloudFront + ACM + Route 53), run from the `prism-infra` AWS profile:

```bash
AWS_PROFILE=prism-infra ./landing/deploy.sh
```

`deploy.sh` creates/uses the `prismcloud-host-website` S3 bucket, enables static
hosting, sets a public-read policy, uploads `landing/`, and (once
`PRISM_CLOUDFRONT_ID` is set) invalidates the CloudFront cache. It mirrors
`~/src/spore-host/spore-host/web/deploy.sh`.

**One-time infra setup** (see the deploy tracking issue for exact commands):
1. ACM certificate for `prismcloud.host` in **us-east-1**.
2. CloudFront distribution: origin = the S3 website endpoint, alternate domain
   name = `prismcloud.host`, viewer policy = redirect-HTTP-to-HTTPS, cert from
   step 1. Note its distribution ID → set `PRISM_CLOUDFRONT_ID`.
3. Route 53 apex A/ALIAS record `prismcloud.host` → the CloudFront distribution.

Docs (`docs.prismcloud.host`) deploy separately — the existing MkDocs `gh-pages`
site, retargeted to the `docs.` subdomain (or its own S3+CloudFront like
`docs.spore.host`). See the tracking issue for the coordinated cutover.

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
