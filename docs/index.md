---
template: home.html
title: Prism — Cloud Research Workstations
hide:
  - navigation
  - toc
---

<main>

<div class="hero">
  <div class="hero-inner">
    <img src="images/prism-icon-transparent.png" alt="Prism" class="hero-icon">
    <h1>Cloud research workstations<br><span class="hero-gradient">in seconds, not hours</span></h1>
    <p class="hero-sub">Pre-configured AWS environments for researchers, labs, and classrooms — launch with one command, stop paying when you're done.</p>
    <div class="hero-buttons">
      <a href="user-guides/QUICK_START/" class="btn-hero-primary">Get Started</a>
      <a href="https://github.com/scttfrdmn/prism" class="btn-hero-secondary" target="_blank">View on GitHub</a>
    </div>
    <div class="install-block">
      <code>brew install scttfrdmn/tap/prism</code>
    </div>
  </div>
</div>

<section aria-labelledby="features-heading" class="features-section">
  <div class="features-grid">
    <article class="feature-card">
      <span class="feature-icon" aria-hidden="true">⚡</span>
      <h2>Launch in seconds</h2>
      <p>One command and you're running. Prism picks the right instance, configures storage, and installs your tools automatically.</p>
    </article>
    <article class="feature-card">
      <span class="feature-icon" aria-hidden="true">💰</span>
      <h2>Cost control built in</h2>
      <p>Budget limits per project, idle detection, and automatic hibernation prevent surprise AWS bills.</p>
    </article>
    <article class="feature-card">
      <span class="feature-icon" aria-hidden="true">🧪</span>
      <h2>Research-ready templates</h2>
      <p>Python ML, R/RStudio, genomics, bioinformatics — pre-configured with the tools your field actually uses.</p>
    </article>
    <article class="feature-card">
      <span class="feature-icon" aria-hidden="true">🖥️</span>
      <h2>CLI and GUI</h2>
      <p>Script with the CLI or use the desktop app for visual management. Both share the same backend daemon.</p>
    </article>
    <article class="feature-card">
      <span class="feature-icon" aria-hidden="true">👥</span>
      <h2>Built for labs and classes</h2>
      <p>Provision workstations for an entire class via invitation codes. Budget pools, project isolation, and usage tracking included.</p>
    </article>
    <article class="feature-card">
      <span class="feature-icon" aria-hidden="true">🔒</span>
      <h2>Secure by default</h2>
      <p>SSH key management, per-project IAM scoping, and encrypted storage. Your data stays in your own AWS account.</p>
    </article>
  </div>
</section>

<section aria-labelledby="quick-start-heading" class="quick-start-section">
  <h2 id="quick-start-heading">From zero to running in minutes</h2>

```bash
# Install
brew install scttfrdmn/tap/prism

# Authenticate with AWS (browser-based, no keys needed)
aws login

# Add a Prism profile (interactive)
prism profiles setup

# Launch a research environment
prism workspace launch python-ml my-project

# Connect via SSH — ready in under 2 minutes
prism workspace connect my-project
```

  <div class="quick-start-links">
    <a href="user-guides/QUICK_START/">Full installation guide →</a>
    <a href="admin-guides/AWS_IAM_PERMISSIONS/">IAM permissions →</a>
  </div>
</section>

<section aria-labelledby="templates-heading" class="templates-section">
  <h2 id="templates-heading">Environment templates</h2>
  <div class="templates-grid">
    <article class="template-card"><span aria-hidden="true">🐍</span><div><strong>python-ml</strong><br>PyTorch, TensorFlow, Jupyter</div></article>
    <article class="template-card"><span aria-hidden="true">📊</span><div><strong>r-research</strong><br>R, RStudio Server, Bioconductor</div></article>
    <article class="template-card"><span aria-hidden="true">🧬</span><div><strong>genomics</strong><br>GATK, BWA, samtools, STAR</div></article>
    <article class="template-card"><span aria-hidden="true">🔬</span><div><strong>bioinformatics</strong><br>Conda, Snakemake, BioPython</div></article>
    <article class="template-card"><span aria-hidden="true">🤖</span><div><strong>deep-learning</strong><br>CUDA, cuDNN, GPU-ready PyTorch</div></article>
    <article class="template-card"><span aria-hidden="true">📡</span><div><strong>data-science</strong><br>Pandas, scikit-learn, DuckDB</div></article>
    <article class="template-card"><span aria-hidden="true">⚙️</span><div><strong>hpc-base</strong><br>MPI, OpenMP, GCC, CMake</div></article>
    <article class="template-card"><span aria-hidden="true">🌐</span><div><strong>+ community</strong><br><a href="user-guides/TEMPLATE_MARKETPLACE_USER_GUIDE/">Browse marketplace</a></div></article>
  </div>
</section>

<section aria-labelledby="downloads-heading" class="downloads-section">
  <h2 id="downloads-heading">Downloads</h2>
  <div class="downloads-grid">
    <article class="download-card">
      <strong><span aria-hidden="true">🍎</span> macOS</strong>
      <span>Homebrew: <code>brew install scttfrdmn/tap/prism</code>, or the <code>.dmg</code> from Releases.</span>
    </article>
    <article class="download-card">
      <strong><span aria-hidden="true">🪟</span> Windows</strong>
      <span>Scoop: <code>scoop install prism</code>, or the <code>.msi</code> from Releases.</span>
    </article>
    <article class="download-card">
      <strong><span aria-hidden="true">🐧</span> Linux</strong>
      <span>Download the release archive and add the binaries to your <code>PATH</code>.</span>
    </article>
  </div>
  <p class="downloads-note">Get all builds on the <a href="https://github.com/scttfrdmn/prism/releases/latest" target="_blank">latest release page</a>. See the <a href="user-guides/INSTALLATION/">Installation guide</a> for full instructions.</p>
</section>

</main>
