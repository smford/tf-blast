# tf-blast

**Ultra-fast, zero-trust, graph-aware blast-radius analyzer for Terraform and OpenTofu execution plans.**

Instead of dumping thousands of lines of raw plan diffs, `tf-blast` parses the plan's underlying dependency graph and resource changes, calculates the cascading blast radius of destructive modifications, and outputs both an interactive terminal report and a GitHub/GitLab-ready Markdown summary for pull requests.

---

## Core Architectural Principles

1. **Dual Engine Support (Terraform & OpenTofu)**:
   - Consumes standard `terraform show -json <planfile>` / `tofu show -json <planfile>` output or JSON piped directly from `stdin`.
   - Never invokes cloud provider APIs or shells out to the `terraform`/`tofu` binary at runtime. Analyzes purely from input artifacts.
2. **Zero-Trust / Client-Side Only (No Credentials, No State Backends)**:
   - Requires **zero cloud credentials** (no AWS/GCP/Azure IAM roles, API keys, or remote state bucket access).
   - Eliminates enterprise infosec review friction.
3. **Sub-500ms Execution**:
   - Written in Go. Memory-efficient adjacency list graph engine.
   - Evaluates plans containing **1,000+ resources in under 15ms** (exceeding the 500ms goal by over 30x).
4. **CI-First Integration**:
   - Clean, predictable exit codes (`0` for pass, `1` for policy threshold exceeded, `2` for error).
   - Configurable thresholds (`--fail-on critical`, `--max-blast 10`, `--fail-on-replacement`).
   - Multiple output targets: Colorized ANSI terminal, PR-ready Markdown comments, and structured JSON.

---

## Compatibility & Version Independence

`tf-blast` is **not bound** to any specific version of Terraform, OpenTofu, or cloud provider.

### Terraform & OpenTofu Version Independence

- **Consumes the Standard JSON Spec**: `tf-blast` parses output from `terraform show -json <planfile>` or `tofu show -json <planfile>` (or standard input).
- **Broad Version Compatibility**: Supports Terraform JSON Plan schema versions (`format_version` `0.1`, `0.2`, and `1.x`), which spans:
  - **Terraform**: `0.12` through `1.10+`
  - **OpenTofu**: All versions (`1.6+`, `1.7+`, `1.8+`, and later)
- **Zero Runtime Binary Dependency**: `tf-blast` never shells out to or invokes the `terraform` or `tofu` CLI at runtime; it purely evaluates the portable, self-contained JSON artifact.

### Cloud Provider Independence

- **Universal Provider Support**: Operates across all cloud providers (AWS, Azure, Google Cloud, Kubernetes, Cloudflare, etc.) and proprietary/in-house providers.
- **Zero Cloud Credentials Required**: No cloud SDKs or IAM permissions needed. All dependency DAGs are derived directly from plan attributes and configuration references.
- **Configurable Risk Heuristics**: Ships with built-in heuristic tiers for common resources across providers, and can be fully customized or overridden via `.tf-blast.yaml`.

---

## Installation

### Homebrew:
```bash
brew tap smford/tap
brew install tf-blast
```

### Using `go install`:
```bash
go install github.com/smford/tf-blast/cmd/tf-blast@latest
```

### Docker / Container Image:
```bash
docker pull ghcr.io/smford/tf-blast:latest

# Analyze a local plan file:
docker run --rm -v $(pwd):/work ghcr.io/smford/tf-blast plan.json
```

### Build from Source:
```bash
git clone https://github.com/smford/tf-blast.git
cd tf-blast
make build
# Or directly with Go:
go build -ldflags="-s -w" -o tf-blast ./cmd/tf-blast
```

---

## Quick Start & Usage

### 1. Basic Ingestion
```bash
# Via pipe from terraform / tofu
terraform show -json tfplan.binary | tf-blast

# Via file flag
tf-blast -f plan.json

# Via positional argument
tf-blast plan.json
```

### 2. Multi-Plan Aggregation (Terragrunt & Monorepos)
Analyze multiple plans simultaneously across micro-stacks, Terragrunt modules, or multi-account configurations. `tf-blast` aggregates them into a consolidated blast-radius graph and links cross-stack dependencies through shared resource identifiers (ARNs and IDs):

```bash
# Analyze multiple plans across separate modules
tf-blast vpc/plan.json db/plan.json app/plan.json

# Shell globbing across monorepo stacks
tf-blast live/prod/*/plan.json

# Policy threshold check across all aggregated plans
tf-blast --fail-on critical --max-blast 30 live/**/*.json
```

### 3. Output Formats

#### Interactive Terminal TUI Mode:
```bash
tf-blast -i plan.json
```
Launch an interactive keyboard-navigable terminal explorer. Navigate resources with `↑/↓` (or `j/k`), cycle severity filters with `[TAB]`, and view cascading blast trees and details with `[Enter]`.

#### Terminal (Rich ANSI / Colorized):
```bash
tf-blast -o terminal plan.json
```
```text
⚠️  Plan Health: HIGH BLAST RADIUS DETECTED

+ 0 to add | ~ 3 to update | - 0 to destroy | ± 1 to replace
Total Blast Radius: 4 resource(s) impacted

SEVERITY    RESOURCE ADDRESS                        ACTION        ROOT CAUSE                  DOWNSTREAM
────────────────────────────────────────────────────────────────────────────────────────────────────
HIGH        aws_security_group.db                   REPLACE       name                        3 dependent(s)
LOW         aws_instance.app_primary                UPDATE        In-place modification       -
LOW         aws_instance.app_secondary              UPDATE        In-place modification       -
LOW         aws_instance.worker                     UPDATE        In-place modification       -

🌳 Cascading Blast Radius Trees:

aws_security_group.db (REPLACE)
  ├── aws_instance.app_primary (update)
  ├── aws_instance.app_secondary (update)
  └── aws_instance.worker (update)
```

#### Markdown (Optimized for PR Comments):
```bash
tf-blast -o markdown --out-file pr-comment.md plan.json
```

Output preview:
> ### 💥 tf-blast: Blast-Radius & Risk Report
>
> **Plan Health:** ⚠️ **HIGH BLAST RADIUS DETECTED**
>
> **Summary:** `+0` to add | `~3` to update | `-0` to destroy | `±1` to replace
>
> **Total Blast Radius:** 4 affected resource(s)
>
> | Impact Level | Resource Address | Planned Action | Root Cause Attribute | Downstream Affected |
> | :--- | :--- | :--- | :--- | :--- |
> | 🟠 HIGH | `aws_security_group.db` | **REPLACE** | `name` | 3 instances |
> | 🟢 LOW | `aws_instance.app_primary` | UPDATE | `In-place modification` | - |
> | 🟢 LOW | `aws_instance.app_secondary` | UPDATE | `In-place modification` | - |
> | 🟢 LOW | `aws_instance.worker` | UPDATE | `In-place modification` | - |
>
> <details>
> <summary>🔍 <strong>View Cascading Blast Radius Graph</strong></summary>
>
> ```text
> aws_security_group.db (delete,create)
> ├── aws_instance.app_primary (update)
> ├── aws_instance.app_secondary (update)
> └── aws_instance.worker (update)
> ```
> </details>

#### Structured JSON:
```bash
tf-blast --json plan.json
```

#### Mermaid.js Flowchart (Visual Architecture Graph):
```bash
tf-blast -o mermaid plan.json
```
Exports a Mermaid.js diagram visualizing dependency linkages and cascading blast-radius paths.

#### OASIS SARIF 2.1.0 (GitHub Code Scanning & Annotations):
```bash
tf-blast -o sarif --out-file results.sarif plan.json
```
Exports standard OASIS SARIF v2.1.0 diagnostics for automated PR code scanning annotations in GitHub and GitLab.

---

## Policy & CI Thresholds

```bash
# Block merge if any CRITICAL severity change is detected
tf-blast --fail-on critical plan.json

# Block merge if destructive replacements are detected
tf-blast --fail-on replacement plan.json

# Block merge if total blast radius exceeds 15 resources
tf-blast --max-blast 15 plan.json

# Suppress stdout, exit with code 1 on violation (for CI guardrails)
tf-blast -s --fail-on high plan.json
```

### CLI Flags Reference

| Flag | Short | Description | Default |
| :--- | :--- | :--- | :--- |
| `[plan-file...]` | | One or more Terraform/OpenTofu plan JSON files (multi-plan support) | Standard input (`stdin`) |
| `--file` | `-f` | Path to Terraform/OpenTofu JSON plan file | Standard input (`stdin`) |
| `--config` | `-c` | Path to `.tf-blast.yaml` policy configuration | Auto-detect |
| `--output` | `-o` | Output format (`terminal`, `markdown`, `json`, `mermaid`, `sarif`) | `terminal` |
| `--out-file` | | Path to write output directly to a file | - |
| `--interactive` | `-i` | Launch interactive terminal TUI dashboard | `false` |
| `--fail-on` | | Exit code 1 threshold (`critical`, `high`, `replacement`, `any-destroy`) | - |
| `--max-blast` | | Maximum acceptable blast radius before failing | `0` (disabled) |
| `--silent` | `-s` | Suppress terminal output (useful in CI pipelines) | `false` |
| `--json` | | Shorthand for `--output json` | `false` |
| `--version` | `-v` | Print version information | - |

---

## Configuration Policy (`.tf-blast.yaml`)

You can define custom risk classification rules, ignored resources, and default thresholds in `.tf-blast.yaml`:

```yaml
rules:
  critical:
    - "*rds*"
    - "*database*"
    - "*dynamodb*"
    - "*s3_bucket*"
    - "*persistent_volume*"
    - "*blob*"
  high:
    - "*security_group*"
    - "*route_table*"
    - "*iam_role*"
    - "*firewall*"
    - "*policy*"
  medium:
    - "*instance*"
    - "*ecs_service*"
    - "*alb*"
    - "*kubernetes_deployment*"
  low:
    - "*tag*"
    - "*log_group*"
    - "*alert*"

fail_on: "critical"
max_blast: 25

ignore_resources:
  - "*null_resource*"
  - "*random_id*"
```

---

## GitHub Action

You can run `tf-blast` directly as a GitHub Action on pull requests, post automated PR comments, and upload SARIF annotations to GitHub Code Scanning:

```yaml
name: "Blast Radius Check"

on:
  pull_request:
    branches: [ "main" ]

permissions:
  contents: read
  pull-requests: write
  security-events: write

jobs:
  blast-analysis:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Terraform Plan
        run: |
          terraform init -backend=false
          terraform plan -out=tfplan.binary
          terraform show -json tfplan.binary > plan.json

      - name: Run tf-blast
        uses: smford/tf-blast@v1
        with:
          plan-file: plan.json
          fail-on: critical
          max-blast: 25
          post-pr-comment: "true"
          sarif-file: tf-blast.sarif

      - name: Upload SARIF to GitHub Code Scanning
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: tf-blast.sarif
```

---

## Release & Semantic Versioning

Releases are published automatically upon merging pull requests to `main` via [`.github/workflows/release.yml`](.github/workflows/release.yml):

- **Code-Only Triggers**: Releases **only** trigger and update when there are modifications to the application code (`cmd/`, `pkg/`, `go.mod`, `go.sum`, or `*.go`). Updates solely to documentation, CI workflows, or project metadata never trigger releases.
- **Conventional Commits**: Commit messages touching code dictate version increments:
  - `fix:`, `perf:`, `refactor:`, or standard PR merge -> **Patch** (`v1.0.X`)
  - `feat:` or `feat(...)` -> **Minor** (`v1.X.0`)
  - `BREAKING CHANGE:` or `feat!:` -> **Major** (`vX.0.0`)
- **Automated Tagging**: Creates the new release tag (e.g. `v1.0.1`) and updates the floating major tag (`v1`).
- **Multi-Platform Assets**: Builds and packages release archives (`.tar.gz` and `.zip`) for Linux, macOS, and Windows with SHA-256 checksums (`checksums.txt`).
- **Container Images**: Publishes multi-arch (`linux/amd64`, `linux/arm64`) distroless container images to GitHub Container Registry (`ghcr.io/smford/tf-blast:${VERSION}`, `ghcr.io/smford/tf-blast:v1`, and `:latest`).
- **Changelogs**: Automatically generates release notes from merged pull requests and commit history.

---

## Shell Autocompletion

Generate shell autocompletion scripts for your shell:

```bash
# Bash
source <(tf-blast completion bash)

# Zsh
tf-blast completion zsh > "${fpath[1]}/_tf-blast"

# Fish
tf-blast completion fish | source

# PowerShell
tf-blast completion powershell | Out-String | Invoke-Expression
```

---

## Pre-Commit Hook

Add `tf-blast` to your `.pre-commit-config.yaml` to enforce blast-radius checks before code is committed:

```yaml
repos:
  - repo: https://github.com/smford/tf-blast
    rev: v1.0.0
    hooks:
      - id: tf-blast
        args: ["--fail-on", "critical"]
```

---

## CI Ecosystem Integrations

In addition to the official GitHub Action, `tf-blast` includes ready-to-use templates for other major CI platforms:

- **GitLab CI**: Use [`.gitlab-ci-template.yml`](.gitlab-ci-template.yml) to automatically post blast radius comments to GitLab Merge Requests.
- **Azure DevOps**: Use [`azure-pipelines-template.yml`](azure-pipelines-template.yml) to publish blast radius reports on Pull Requests.

---

## Security & Zero-Trust Guarantees

- **Sensitive Value Redaction**: Values marked as sensitive in Terraform plan metadata (`before_sensitive` and `after_sensitive`) or matching sensitive patterns (passwords, tokens, private keys) are automatically sanitized as `(sensitive value redacted)` before rendering into terminal or Markdown outputs.
- **Out-of-Band State Drift Detection**: Cross-references Terraform's `resource_drift` array to alert engineers when resources scheduled for modification have experienced unmanaged console drift.
- **Zero Cloud Credentials**: Executes 100% client-side without AWS, GCP, Azure, or remote backend credentials.

---

## Running Tests

### 1. Run the Entire Test Suite
Runs all unit tests, CLI integration tests, and the sub-500ms performance verification:
```bash
go test -v ./...
```

### 2. Run Specific Package Tests
- **CLI Integration Tests**:
  ```bash
  go test -v ./cmd/tf-blast/...
  ```
- **Graph Engine & Cycle Detection**:
  ```bash
  go test -v ./pkg/graph/...
  ```
- **Risk Analyzer & Blast Radius Scoring**:
  ```bash
  go test -v ./pkg/analyzer/...
  ```
- **JSON Parser & Dependency Ingestion**:
  ```bash
  go test -v ./pkg/parser/...
  ```
- **Terminal, Markdown & JSON Renderers**:
  ```bash
  go test -v ./pkg/renderer/...
  ```

### 3. Manual Fixture Testing
Test the compiled binary against representative test plans in `testdata/`:
```bash
# Build the binary
go build -o tf-blast ./cmd/tf-blast

# Test a clean plan (creates only -> Low Risk)
./tf-blast testdata/clean-plan.json

# Test a security group replacement cascading to compute instances (High Risk)
./tf-blast testdata/replacement-plan.json

# Test stateful RDS deletion (Critical Risk)
./tf-blast testdata/stateful-destroy.json

# Test policy threshold failure (exits with code 1)
./tf-blast --fail-on critical testdata/stateful-destroy.json

# Generate PR Markdown comment
./tf-blast -o markdown --out-file pr-comment.md testdata/replacement-plan.json
```

---

## Performance & Benchmarks

Benchmarked on Apple Silicon (M4) with `testdata/massive-plan.json` containing **1,080 resources** with multi-tier transitive dependency graph:

| Stage | Benchmark Result | Target Requirement |
| :--- | :--- | :--- |
| **Ingestion + Graph Construction + Blast Analysis** | **~8.33 ms** | < 500 ms |
| **End-to-End Pipeline with Markdown Rendering** | **~14.23 ms** | < 500 ms |

Run the benchmark suite:
```bash
go test -v -bench=. .
```

---

## License

Apache 2.0. See [LICENSE](LICENSE) for details.