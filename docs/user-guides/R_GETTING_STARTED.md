# Getting Started with R on Prism

This tutorial walks you through Prism's R template family — from a minimal scripting environment
all the way to a published Shiny app — with practical R workflows at each step. By the end
you'll have a reproducible research environment and a custom AMI that launches in under 2 minutes.

**Prerequisites**: Prism installed, AWS credentials configured, `prism init` completed.

---

## The R Template Family

Prism's R templates form two inheritance trees. Choose your path before launching:

```
R Base (Ubuntu 24.04)          ← command-line R, no IDE
 ├── R + RStudio Server         ← web IDE, interactive work       ⭐ start here
 │    └── R Research Publishing Stack  ← + Quarto, LaTeX, Python
 └── R Shiny Server             ← share analyses as web apps

R Research Full Stack           ← monolithic: RStudio + Quarto + Python, independent
```

| Template | Best for | Launch time | Disk | Instance |
|----------|----------|-------------|------|----------|
| R Base | Scripts, CI, building blocks | ~5 min | 20 GB | any |
| R + RStudio Server | Interactive analysis, teaching | ~8 min | 25 GB | t3.medium |
| R Research Full Stack | Complete lab environment | ~15 min | 80 GB | m7i.xlarge |
| R Research Publishing Stack | Papers, reproducible reports | ~18 min | 80 GB | m7i.xlarge |
| R Shiny Server | Sharing dashboards with others | ~10 min | 30 GB | t3.medium |

---

## Part 1: Your First R Workspace

**R + RStudio Server** is the right starting point for most researchers. It gives you a full
web-based IDE without the weight of a complete publishing stack.

### Launch

```bash
prism workspace launch r-rstudio-server my-analysis
```

Prism provisions the instance, installs R 4.4+, RStudio Server, and all base packages. Watch the
output — it takes about 8 minutes. When it completes you'll see:

```
✅ Instance ready: my-analysis
   Public IP: 54.123.45.67
   Services:
     RStudio Server: http://54.123.45.67:8787
```

### Connect

**SSH** (works immediately — uses your SSH key):

```bash
prism workspace connect my-analysis          # opens SSH terminal
```

**RStudio Server** requires a system password for the `researcher` user. Templates that
include RStudio generate a random password during provisioning and print it in the launch
log. Look for lines like:

```
=============================================
  RStudio Server Login Credentials
  Username: researcher
  Password: <random-password>
=============================================
```

If you missed the password or need to reset it, SSH in and run:

```bash
sudo passwd researcher
```

Then open `http://<public-ip>:8787` and log in with username `researcher` and the password
you set.

### Verify your environment

In the RStudio console:

```r
R.version.string     # should show R 4.4.x
installed.packages()[, "Package"]  # tidyverse, rmarkdown, knitr, devtools already here
```

---

## Part 2: Installing Packages Efficiently

The templates use **Posit Package Manager** — a binary package repository for Ubuntu 24.04.
Packages install in seconds instead of minutes because they come pre-compiled.

### Set Posit Package Manager as your default

In RStudio, run once per session or add to `~/.Rprofile`:

```r
options(repos = c(CRAN = "https://packagemanager.posit.co/cran/__linux__/noble/latest"))
```

This is already configured for the initial install — packages you add later will also use it.

### Install packages the fast way

```r
# Examples: pre-built binaries, no compilation
install.packages("lme4")       # ~3 seconds vs ~4 minutes from source
install.packages("brms")       # Bayesian modeling
install.packages("targets")    # Pipeline toolkit
```

Compare: on a standard CRAN mirror without binaries, `lme4` compiles C++ code and takes several
minutes. With Posit Package Manager it's a simple download.

### Bioconductor packages

```r
if (!require("BiocManager")) install.packages("BiocManager")
BiocManager::install("DESeq2")
```

---

## Part 3: Reproducible Environments with renv

`renv` locks your package versions so collaborators and future-you get the same environment.

### Initialize renv in a project

```r
# In RStudio: File > New Project > New Directory, or in the console:
setwd("~/projects/my-study")
renv::init()
```

`renv` creates:
- `renv.lock` — the exact version of every package
- `.Rprofile` — loads renv automatically when you open the project
- `renv/` — local package library (not shared with other projects)

### Snapshot after installing packages

```r
install.packages(c("lme4", "emmeans", "ggplot2"))
renv::snapshot()    # writes versions to renv.lock
```

### Share with collaborators

Send them your project (including `renv.lock`). They run:

```r
renv::restore()     # installs exactly your versions
```

### Recommended workflow

```r
# Start of session
renv::status()      # check if lock file is current

# After adding packages
renv::snapshot()

# Before sharing / archiving
renv::status()
git add renv.lock
git commit -m "lock package versions"
```

---

## Part 4: Cost-Effective Workflows — Hibernate and Resume

Cloud instances cost money while idle. Prism's hibernation feature preserves your work and
stops billing (you pay only for EBS storage, ~$0.10/GB/month).

### Check your instance cost

```bash
prism workspace list                 # shows instance type and estimated hourly cost
```

A t3.medium (default for RStudio) costs ~$0.042/hour. Left running for a week = ~$7.

### Hibernate when you're done for the day

From your local terminal:

```bash
prism workspace hibernate my-analysis
```

RStudio Server saves its state. The instance stops. Your files, installed packages, and running R
sessions (if you save `.RData`) are preserved on the EBS volume.

### Resume the next day

```bash
prism workspace resume my-analysis
```

In ~2 minutes, your instance is back at the same IP with all files intact.

### Save your R session before hibernating

In RStudio:

```r
save.image("~/projects/my-study/session.RData")
```

Or use RStudio's built-in "Save workspace" option. On resume:

```r
load("~/projects/my-study/session.RData")
```

### Auto-hibernation

The RStudio template auto-hibernates after **60 minutes** of idle (no active R processes,
no RStudio sessions). Adjust this in the instance settings if you run long background jobs.

---

## Part 5: Scaling Up — Full Research Environments

When your analysis outgrows the base RStudio environment, move to one of the full stacks.

### When to upgrade

| Situation | Template to use |
|-----------|-----------------|
| Need Quarto for papers/reports | R Research Publishing Stack |
| Need Python + R in same environment | R Research Full Stack or Publishing Stack |
| Need LaTeX for PDF output | R Research Publishing Stack |
| Need Jupyter notebooks | R Research Full Stack |
| Heavy computation (>4 GB data in memory) | Either full stack (m7i.xlarge = 16 GB RAM) |

### R Research Full Stack — the monolithic approach

A standalone environment with everything pre-installed:

```bash
prism workspace launch r-research-full-stack my-big-study
```

Launches on **m7i.xlarge** (4 vCPU, 16 GB RAM) by default — a non-burstable instance suitable
for sustained computation. Available at both ports 8787 (RStudio) and 8888 (Jupyter Lab).

```bash
# Access RStudio
open http://<ip>:8787

# Access Jupyter Lab
open http://<ip>:8888
```

Includes 80 GB disk — enough for medium-sized datasets and LaTeX.

### R Research Publishing Stack — the layered approach

Builds on top of R + RStudio Server, adding Quarto, full TeX Live, Python, and Jupyter:

```bash
prism workspace launch r-publishing-stack my-paper
```

Also launches on m7i.xlarge with 80 GB disk. This is the recommended template for
writing papers — Quarto handles both HTML and PDF output from the same `.qmd` source.

**Quick Quarto example** (in RStudio terminal):

```bash
# Create a new article
quarto create article my-paper

# Preview in browser
quarto preview my-paper.qmd

# Render to PDF (requires LaTeX — included)
quarto render my-paper.qmd --to pdf
```

### Mixed R + Python workflows

Both full stacks include Python 3.12 and Jupyter. Switch between languages in a single
Quarto document:

```r
# my-analysis.qmd
# ```{r}
library(reticulate)
use_python("/usr/bin/python3")
# ```

# ```{python}
import pandas as pd
df = pd.read_csv("data.csv")
# ```

# ```{r}
py$df |> as_tibble()   # access Python's df from R
# ```
```

---

## Part 6: Sharing Your Work with Shiny

Once your analysis is complete, use the **R Shiny Server** template to share it as an
interactive web app with colleagues or students.

### Launch a Shiny server

```bash
prism workspace launch r-shiny my-shiny-dashboard
```

Installs R, Shiny Server, and a sample demo app. Accessible at port 3838.

### Deploy your app

SSH in and copy your app directory:

```bash
prism workspace connect my-shiny-dashboard

# On the instance:
sudo mkdir -p /srv/shiny-server/my-app
sudo cp -r ~/projects/my-app/* /srv/shiny-server/my-app/
sudo chown -R shiny:shiny /srv/shiny-server/my-app
sudo systemctl restart shiny-server
```

Access at: `http://<ip>:3838/my-app/`

### Install additional R packages for your app

```bash
prism workspace connect my-shiny-dashboard
```

```r
# On the instance, as researcher
options(repos = c(CRAN = "https://packagemanager.posit.co/cran/__linux__/noble/latest"))
install.packages(c("shinydashboard", "plotly", "leaflet"))
sudo systemctl restart shiny-server
```

### Transfer an app from your analysis instance

If you developed the app on `my-analysis` (RStudio), get both IPs then transfer:

```bash
# Get IP addresses
prism workspace list

# Copy app from analysis instance to shiny instance
ANALYSIS_IP=<analysis-instance-ip>
SHINY_IP=<shiny-instance-ip>

ssh researcher@${ANALYSIS_IP} "tar czf /tmp/my-app.tar.gz ~/projects/my-app"
scp researcher@${ANALYSIS_IP}:/tmp/my-app.tar.gz /tmp/
scp /tmp/my-app.tar.gz researcher@${SHINY_IP}:/tmp/
ssh researcher@${SHINY_IP} "
  sudo mkdir -p /srv/shiny-server/my-app
  sudo tar xzf /tmp/my-app.tar.gz -C /srv/shiny-server/my-app --strip-components=2
  sudo chown -R shiny:shiny /srv/shiny-server/my-app
  sudo systemctl restart shiny-server
"
```

---

## Part 7: Lock In Your Environment with an AMI

After spending 15-90 minutes configuring an environment, save it as an AMI (Amazon Machine Image).
Future launches from that AMI take **under 2 minutes** instead of 15-90 minutes.

### When to create an AMI

- After installing your domain-specific packages (e.g., a full bioinformatics stack)
- After configuring your `.Rprofile`, SSH keys, and project structure
- Before a course or workshop (saves time for all participants)
- Any time you want to share a ready-to-use environment with collaborators

### Create an AMI

```bash
prism ami save my-analysis "R 4.4 Genomics - March 2026"
```

This takes ~5 minutes. The instance keeps running while the snapshot is taken.

### Launch from your AMI

```bash
prism workspace launch --ami "R 4.4 Genomics - March 2026" my-new-instance
```

Or in the GUI, select your saved AMI from the launch dialog.

### AMI naming tips

Include the date and key packages in the name:
- `"R 4.4 + Seurat 5 + DESeq2 - 2026-03"` — genomics stack
- `"R + Quarto + tinytex - Stats 510 Fall 2026"` — course environment
- `"R Shiny + leaflet + DT - Lab Dashboard"` — shared dashboard base

### List and manage AMIs

```bash
prism ami list                                  # all saved AMIs
prism ami status <ami-id>                       # details for a specific AMI
prism ami delete <ami-id>                       # remove when no longer needed
```

---

## Quick Reference

### Common commands

```bash
# Launch templates
prism workspace launch r-base-ubuntu24 my-scripts          # minimal, SSH only
prism workspace launch r-rstudio-server my-analysis        # RStudio web IDE
prism workspace launch r-research-full-stack my-lab        # full stack, m7i.xlarge
prism workspace launch r-publishing-stack my-paper         # + Quarto + LaTeX
prism workspace launch r-shiny my-dashboard                # Shiny Server

# Instance management
prism workspace list                                       # all instances + IPs
prism workspace connect my-analysis                        # SSH into instance
prism workspace hibernate my-analysis                      # stop + preserve
prism workspace resume my-analysis                         # restart from hibernate
prism workspace delete my-analysis                         # destroy (irreversible)

# AMI management
prism ami save my-analysis "name"                          # save environment
prism ami list                                             # list saved AMIs
prism workspace launch --ami "name" new-instance           # launch from AMI
prism ami delete <ami-id>                                  # remove AMI
```

### Access URLs

| Template | URL |
|----------|-----|
| R + RStudio Server | `http://<ip>:8787` |
| R Research Full Stack | `http://<ip>:8787` (RStudio) / `http://<ip>:8888` (Jupyter) |
| R Research Publishing Stack | `http://<ip>:8787` (RStudio) / `http://<ip>:8888` (Jupyter) |
| R Shiny Server | `http://<ip>:3838` |

**RStudio login**: Username `researcher`, password from provisioning log (or reset via `sudo passwd researcher` over SSH)

### Posit Package Manager URL

```r
options(repos = c(CRAN = "https://packagemanager.posit.co/cran/__linux__/noble/latest"))
```

Add to `~/.Rprofile` on your instance to make it permanent.

---

## Next Steps

- **Reference guide**: [R Research Template Guide](R_GETTING_STARTED.md) — deep dive into
  the Full Stack template with worked examples (Quarto documents, R+Python, database connections)
- **Shared storage**: `prism volume create shared-data --size 100` — attach an EFS volume to share
  data between your analysis and Shiny instances
- **Collaboration**: Add colleagues as users with `sudo adduser colleague` on the instance; they
  log in at `http://<ip>:8787` with their own credentials
- **Cost visibility**: `prism budget` — see what your R instances are spending per day

---

# Full-stack R template reference


## Overview

The **R Research Full Stack** template provides a complete, production-ready R research environment designed for collaborative data analysis. It includes everything needed for modern R-based research: RStudio Server (web-based IDE), Quarto for publishing, LaTeX for documents, Python integration, and essential data science tools.

**Perfect for:**
- Collaborative research projects with remote team members
- Publishing research papers with R Markdown/Quarto
- Mixed R and Python data science workflows
- Teaching and coursework (web-based access)
- Multi-user research environments

## Quick Start (⏱️ 5 minutes)

### 1. Launch the Environment

```bash
# Launch R research environment
prism workspace launch r-research-full-stack my-r-project

# Wait for installation (this takes 45-90 minutes first time)
# The template installs R, RStudio Server, Quarto, LaTeX, and 40+ R packages
```

**Installation Components:**
- R 4.5.2 with base packages (2-3 min)
- RStudio Server 2026.01.1 (1-2 min)
- Quarto 1.6.33 (1 min)
- TeX Live 2024 full distribution (20-40 min — large download, ~8 GB installed)
- System packages (numpy, pandas, scipy, etc.) (3-5 min, from Ubuntu apt)
- R packages via Posit Package Manager (20-40 min — compiles from source if binary not available)
- Python 3.12 + Jupyter Lab in venv (2-3 min)
- Database clients and utilities (1-2 min)

### 2. Access RStudio Server

```bash
# Get connection info
prism workspace connect my-r-project

# Output shows:
# RStudio Server: http://54.123.45.67:8787
# Username: researcher
# Password: [your instance password]
```

### 3. Open RStudio in Your Browser

1. Navigate to the RStudio Server URL (port 8787)
2. Login with your credentials
3. Start analyzing data in R!

**You now have access to:**
- Full RStudio IDE in your browser
- All tidyverse packages pre-installed
- Quarto for document publishing
- LaTeX for PDF generation
- Git integration for version control

## What's Included

### Core R Environment
- **R 4.5.2**: Latest stable R release (from CRAN noble-cran40 repository)
- **RStudio Server 2026.01.1**: Web-based IDE on port 8787
- **40+ R packages pre-installed**:
  - **Data manipulation**: dplyr, tidyr, purrr, stringr
  - **Visualization**: ggplot2, plotly, viridis, scales
  - **Publishing**: rmarkdown, knitr, bookdown, blogdown, xaringan
  - **Tables**: gt, gtsummary
  - **Database**: DBI, RSQLite, RPostgres, RMySQL
  - **Web**: httr, jsonlite, xml2, rvest, shiny
  - **Development**: devtools, usethis, testthat, pkgdown
  - **Python integration**: reticulate
  - **Utilities**: here, fs, glue, lubridate, forcats

### Publishing Tools
- **Quarto 1.6.33**: Modern scientific publishing system
- **Pandoc 3.5**: Universal document converter
- **TeX Live 2024**: Full LaTeX distribution with all packages
  - pdflatex, xelatex, lualatex
  - All fonts and packages for academic publishing
- **Document tools**: ghostscript, pdftk-java, ImageMagick

### Python Integration
- **Python 3.12**: Latest Python for mixed workflows
- **Jupyter Lab**: Interactive notebooks (port 8888)
- **Scientific packages**: numpy, pandas, matplotlib, seaborn, scikit-learn, scipy
- **Reticulate**: Seamless R-Python integration in RStudio

### Database Support
- **PostgreSQL client**: Connect to PostgreSQL databases
- **MySQL client**: Connect to MySQL/MariaDB databases
- **SQLite**: Embedded database for local data
- **R database packages**: DBI, RSQLite, RPostgres, RMySQL

### Development Tools
- **Git 2.43+**: Version control with LFS support for large files
- **Text editors**: vim, nano, emacs-nox
- **Terminal multiplexers**: tmux, screen for persistent sessions
- **System monitoring**: htop, tree, ncdu

### Data Processing Utilities
- **csvkit**: Command-line CSV tools
- **jq**: JSON processor
- **xmlstarlet**: XML toolkit
- **Compression tools**: zip, unzip, bzip2, p7zip
- **File transfer**: rsync, wget, curl

## Usage Examples

### Example 1: Create and Render Quarto Document

```bash
# SSH into your instance
prism workspace connect my-r-project

# Create new Quarto project
cd ~/documents
quarto create-project my-analysis --type manuscript

# Edit the document
cd my-analysis
nano index.qmd

# Render to PDF
quarto render

# The PDF is now in _output/my-analysis.pdf
```

### Example 2: Mixed R and Python Workflow

In RStudio Server (http://your-ip:8787):

```r
# Install reticulate if not already installed
# library(reticulate)

# Use Python from R
library(reticulate)
use_python("/usr/bin/python3")

# Import Python libraries
pd <- import("pandas")
np <- import("numpy")

# Create DataFrame in Python, use in R
py_data <- pd$DataFrame(list(
  x = np$array(c(1, 2, 3, 4, 5)),
  y = np$array(c(2, 4, 6, 8, 10))
))

# Convert to R data frame
r_data <- py_to_r(py_data)

# Use ggplot2 for visualization
library(ggplot2)
ggplot(r_data, aes(x = x, y = y)) +
  geom_point() +
  geom_smooth(method = "lm")
```

### Example 3: Connect to PostgreSQL Database

```r
# Load database packages
library(DBI)
library(RPostgres)

# Connect to database
con <- dbConnect(
  RPostgres::Postgres(),
  host = "your-db-host.amazonaws.com",
  port = 5432,
  dbname = "research_data",
  user = "researcher",
  password = Sys.getenv("DB_PASSWORD")
)

# Query data
data <- dbGetQuery(con, "
  SELECT *
  FROM experiments
  WHERE experiment_date > '2024-01-01'
")

# Analyze with tidyverse
library(dplyr)
summary_stats <- data %>%
  group_by(treatment) %>%
  summarise(
    mean_response = mean(response),
    sd_response = sd(response),
    n = n()
  )

# Disconnect
dbDisconnect(con)
```

### Example 4: Create Interactive Shiny Dashboard

```r
# Create new Shiny app
library(shiny)
library(ggplot2)
library(dplyr)

# app.R
ui <- fluidPage(
  titlePanel("Research Data Explorer"),

  sidebarLayout(
    sidebarPanel(
      selectInput("variable", "Variable:",
                 choices = c("Sepal.Length", "Sepal.Width",
                           "Petal.Length", "Petal.Width")),
      sliderInput("bins", "Number of bins:",
                 min = 5, max = 50, value = 30)
    ),

    mainPanel(
      plotOutput("distPlot")
    )
  )
)

server <- function(input, output) {
  output$distPlot <- renderPlot({
    ggplot(iris, aes_string(x = input$variable)) +
      geom_histogram(bins = input$bins, fill = "steelblue") +
      theme_minimal() +
      labs(title = paste("Distribution of", input$variable))
  })
}

shinyApp(ui = ui, server = server)

# Run the app
# Access at http://your-ip:3838
```

### Example 5: Generate Research Paper with Quarto

Create `paper.qmd`:

```markdown
---
title: "My Research Paper"
author: "Researcher Name"
date: today
format:
  pdf:
    toc: true
    number-sections: true
    colorlinks: true
bibliography: references.bib
---

## Introduction

This paper analyzes...

## Methods

```{r}
#| label: setup
#| include: false
library(tidyverse)
library(knitr)
library(gt)
```

## Results

```{r}
#| label: fig-analysis
#| fig-cap: "Distribution of experimental results"

data <- read_csv("data/results.csv")

ggplot(data, aes(x = treatment, y = response)) +
  geom_boxplot() +
  theme_minimal()
```

## Conclusion

Our findings show...

## References
```

Render:
```bash
quarto render paper.qmd
```

## Collaboration Setup

### Add a Collaborator

```bash
# SSH into your instance
prism workspace connect my-r-project

# Create user account
sudo adduser colleague
sudo usermod -aG sudo colleague

# Set RStudio Server password (same as Linux password)
# User can now login at http://your-ip:8787
```

### Share Project Access

```r
# In RStudio Server, set project permissions
# File > New Project > Existing Directory
# Select ~/projects/shared-analysis

# Set directory permissions for collaboration
system("chmod -R 775 ~/projects/shared-analysis")
system("chgrp -R sudo ~/projects/shared-analysis")
```

### Concurrent Work

Multiple users can:
- Work simultaneously in RStudio Server (separate sessions)
- Share R projects in `/home/shared/` or specific project directories
- Use Git for version control and collaboration
- Access the same data files in shared directories

## Performance Optimization

### Create Custom AMI for Faster Launch

After first launch and full installation (15-20 minutes):

```bash
# Create AMI from configured instance
prism ami create my-r-project --name "R Research Full Stack AMI"

# Future launches from AMI: < 2 minutes!
prism workspace launch --ami ami-abc123def456 quick-r-instance
```

**Benefits:**
- Launch time: 15-20 minutes → < 2 minutes
- All packages pre-installed and cached
- Custom configurations preserved
- Share AMI with colleagues or students

See [Custom AMI Workflow Guide](CUSTOM_AMI_WORKFLOW.md) for details.

### Instance Sizing Recommendations

> **Note**: The fullstack and publishing templates default to `m7i.xlarge` (4 vCPU, 16 GB).
> Burstable `t3.*` instances are **not recommended** — texlive configuration and R package
> compilation exhaust CPU burst credits quickly, extending install time 2-3x.

**Standard Research Work (recommended default)**:
```bash
prism workspace launch r-research-full-stack my-project
# Instance: m7i.xlarge (4 vCPU, 16 GB RAM) — template default, Intel Sapphire Rapids
# Cost: ~$0.21/hour
# Install time: ~60-90 min (first launch, R packages compiled from source)
```

**Large Datasets / Complex Models**:
```bash
prism workspace launch r-research-full-stack my-project --size XL
# Instance: m7i.2xlarge (8 vCPU, 32 GB RAM)
# Cost: ~$0.42/hour
```

**Memory-Intensive R Work (large in-memory datasets)**:
```bash
prism workspace launch r-research-full-stack my-project --instance-type r7i.xlarge
# Instance: r7i.xlarge (4 vCPU, 32 GB RAM) — memory optimized
# Cost: ~$0.30/hour
```

### Cost Optimization with Hibernation

```bash
# Hibernate when not in use (preserves all state)
prism workspace hibernate my-r-project

# Resume when needed (< 2 minutes)
prism workspace resume my-r-project

# Savings: ~90% reduction in compute costs
```

## Troubleshooting

### RStudio Server Not Accessible

**Check if service is running:**
```bash
prism workspace connect my-r-project
sudo systemctl status rstudio-server

# If not running, start it
sudo systemctl start rstudio-server
```

**Check firewall (security group):**
```bash
# Verify port 8787 is open
prism workspace list my-r-project
# Look for "Ports: [22, 8787, 8888]"
```

**Can't login to RStudio Server:**
- Username is the system username (default: `researcher`)
- Password is the Linux user password
- Set/reset password: `sudo passwd researcher`

### R Package Installation Fails

**Insufficient memory:**
```bash
# Launch larger instance
prism workspace launch r-research-full-stack my-project --size L
```

**Missing system dependencies:**
```bash
# Install additional dev libraries
sudo apt-get install -y libgdal-dev libproj-dev
```

**Install R package with dependencies:**
```r
# In RStudio Server or R console
install.packages("sf", dependencies = TRUE)
```

### Quarto Render Fails

**LaTeX errors:**
```bash
# Verify TeX Live installation
which pdflatex
pdflatex --version

# If needed, reinstall
sudo apt-get install --reinstall texlive-full
```

**Missing Quarto:**
```bash
# Verify installation
quarto --version

# If needed, reinstall
wget https://github.com/quarto-dev/quarto-cli/releases/download/v1.6.33/quarto-1.6.33-linux-amd64.deb
sudo apt-get install -y ./quarto-1.6.33-linux-amd64.deb
```

### Python/Jupyter Integration Issues

**Jupyter not found:**
```bash
# Verify installation
which jupyter
jupyter --version

# If needed, reinstall (Ubuntu 24.04 uses apt for system packages)
sudo apt-get install -y jupyter-notebook
pip3 install --break-system-packages jupyterlab
```

**Reticulate can't find Python:**
```r
library(reticulate)
use_python("/usr/bin/python3", required = TRUE)
py_config()
```

### Database Connection Issues

**PostgreSQL connection fails:**
```bash
# Test connection from command line
psql -h your-db-host -U username -d database_name

# Check security groups allow outbound connections
# Check database firewall allows your instance IP
```

## Advanced Features

### Using Git for Version Control

```bash
# Configure Git
git config --global user.name "Your Name"
git config --global user.email "you@example.com"

# Initialize repository
cd ~/projects/my-analysis
git init
git add .
git commit -m "Initial commit"

# Connect to GitHub (or GitLab, Bitbucket)
git remote add origin https://github.com/yourusername/my-analysis.git
git push -u origin main
```

### Large File Support with Git LFS

```bash
# Track large data files with LFS
cd ~/projects/my-analysis
git lfs track "*.csv"
git lfs track "*.rds"
git lfs track "*.RData"

# Add and commit
git add .gitattributes
git add data/large-file.csv
git commit -m "Add large data file"
git push
```

### Running R Scripts in Background

```bash
# Run R script in background
nohup Rscript my_analysis.R > output.log 2>&1 &

# Check progress
tail -f output.log

# Find process
ps aux | grep Rscript
```

### Scheduling R Scripts with Cron

```bash
# Edit crontab
crontab -e

# Run script daily at 2 AM
0 2 * * * /usr/bin/Rscript /home/researcher/scripts/daily_analysis.R >> /home/researcher/logs/daily.log 2>&1
```

## Best Practices

### Project Organization

```
~/projects/my-research/
├── data/
│   ├── raw/           # Original, immutable data
│   ├── processed/     # Cleaned, processed data
│   └── external/      # External reference data
├── R/                 # R scripts and functions
├── notebooks/         # Jupyter/R notebooks for exploration
├── reports/           # Quarto/RMarkdown reports
├── figures/           # Generated figures and plots
├── results/           # Analysis results
├── docs/              # Documentation
├── .gitignore
├── README.md
└── my-research.Rproj  # RStudio project file
```

### Data Management

1. **Keep raw data immutable**: Never modify original data files
2. **Document data processing**: Use R Markdown/Quarto notebooks
3. **Use relative paths**: Use the `here` package for portable code
4. **Version control code, not data**: Use Git LFS for large data files
5. **Back up regularly**: Use `rsync` or cloud storage

### Reproducible Research

```r
# Use renv for package management
install.packages("renv")
renv::init()           # Initialize project environment
renv::snapshot()       # Save package versions
renv::restore()        # Restore exact package versions

# Document session info
sessionInfo()
```

## Resources

### Official Documentation
- **RStudio**: https://www.rstudio.com/
- **Quarto**: https://quarto.org/
- **Tidyverse**: https://www.tidyverse.org/
- **R for Data Science**: https://r4ds.had.co.nz/
- **Jupyter**: https://jupyter.org/

### Prism Documentation
- [Multi-User Instance Setup](MULTI_USER_INSTANCE_SETUP.md) - Detailed collaboration guide
- [Custom AMI Workflow](CUSTOM_AMI_WORKFLOW.md) - Create reusable AMIs for faster launches
- [Template Format](TEMPLATE_FORMAT.md) - Customize templates
- [Community Template Guide](../development/COMMUNITY_TEMPLATE_GUIDE.md) - Contribute templates

### Getting Help
- **Prism Issues**: https://github.com/scttfrdmn/prism/issues
- **RStudio Community**: https://community.rstudio.com/
- **Stack Overflow**: Tag questions with `[r]`, `[rstudio]`, `[quarto]`

## FAQ

**Q: How long does the initial launch take?**
A: First launch: 45-90 minutes (TeX Live ~8 GB + R packages compiled from source). Create an AMI after first launch, then future launches take < 2 minutes.

**Q: Can multiple users work simultaneously?**
A: Yes! Each user gets their own RStudio Server session. Add users with `sudo adduser username`.

**Q: What's the difference between R Markdown and Quarto?**
A: Quarto is the next-generation of R Markdown with better multi-language support, consistent syntax, and enhanced features. Both are included.

**Q: Can I use R packages not pre-installed?**
A: Yes! Install any CRAN, Bioconductor, or GitHub package: `install.packages("package")` or `devtools::install_github("user/package")`.

**Q: How do I share my environment with colleagues?**
A: Create an AMI, share the AMI ID, or use template inheritance to create a customized version.

**Q: Does this work on ARM instances?**
A: Currently x86_64 only. ARM support planned for future versions.

**Q: Can I customize the template?**
A: Yes! Copy the template to `~/.prism/templates/my-r-env.yml` and modify as needed. See [Template Format](TEMPLATE_FORMAT.md).

## Related Templates

- **R Research Minimal**: Lightweight R environment (R + RStudio only)
- **Python ML**: Python-focused data science environment
- **Ultimate Research Workstation**: Multi-language research platform

## Version History

- **v1.0.0** (January 2026): Initial release
  - R 4.5.2 + RStudio Server 2026.01.1
  - Quarto 1.6.33 + TeX Live 2024
  - Python 3.12 + Jupyter Lab
  - 40+ pre-installed R packages (from Ubuntu apt, no compilation needed)
  - Full collaboration support

---

**Template**: `r-research-full-stack`
**Last Updated**: February 28, 2026
**Version**: 1.1.0 (R 4.5.2, RStudio Server 2026.01.1, Ubuntu 24.04 Noble)
