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

```bash
AWS_PROFILE=spore-host-infra ./landing/deploy.sh
```

`deploy.sh` uploads `landing/` to the `prismcloud-host-website` bucket and
invalidates the CloudFront cache. Pass the profile explicitly if your shell
already sets `AWS_PROFILE` (e.g. `aws` for Prism's test account).

### Infrastructure

Created once (#686) in the **spore-host-infra** account (966362334030), which
also holds the `prismcloud.host` Route 53 zone (`Z05117751LI9W0I0HUXM5`) and
spore.host:

| Resource | Value |
|----------|-------|
| S3 bucket | `prismcloud-host-website` (us-east-1, private; all public access blocked) |
| Origin Access Control | `E3HI7CC3HQFLMZ`; the bucket policy allows reads only from the distribution |
| ACM certificate | `prismcloud.host` + `www.prismcloud.host` (us-east-1, DNS-validated) |
| CloudFront | `E5FM6YPIFHGR9` (`d3bxqhnf4uvbjh.cloudfront.net`): HTTPS redirect, managed CachingOptimized + SecurityHeaders policies, 403/404 → `/index.html` with status 404 |
| Route 53 | `prismcloud.host` and `www.prismcloud.host`: A + AAAA aliases to the distribution |

Unlike spore.host's bucket, this one isn't an S3 static website: CloudFront
reads it over the S3 REST endpoint through Origin Access Control, so the
bucket is never public.

### Domain topology

| Host | Serves |
|------|--------|
| `prismcloud.host`, `www.prismcloud.host` | this landing page (CloudFront) |
| `docs.prismcloud.host` | the MkDocs docs on GitHub Pages: a CNAME to `scttfrdmn.github.io`; `docs/CNAME` and `site_url` in `mkdocs.yml` match |
| `<name>.<account-base36>.prismcloud.host` | workspace DNS; spored self-registers each instance |
| `prismcloud.io` | redirects to `https://prismcloud.host` (forwarding at Porkbun, its registrar) |

Workspace names are registered under an account label, so they don't collide
with `www` or `docs`.

## Editing

All copy lives directly in `index.html`. Keep any `prism ...` commands accurate
against the current CLI (`prism workspace launch|list|...`, `prism profiles`,
`aws login`).
