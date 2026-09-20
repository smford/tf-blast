# tf-blast GitHub Action

The official GitHub Action for [tf-blast](https://github.com/smford/tf-blast) provides zero-trust, graph-aware blast radius analysis and semantic risk scoring for Terraform and OpenTofu execution plans directly within GitHub Pull Requests.

---

## Key Features

- **Automated Sticky PR Comments**: Posts an interactive, collapsible Markdown report directly onto Pull Requests. Subsequent commits update the existing comment in-place to avoid PR noise.
- **Inline PR Code Annotations**: Emits standard OASIS SARIF v2.1.0 diagnostics and uploads them to GitHub Code Scanning, highlighting high-risk and destructive changes on the exact lines modified.
- **CI Policy Gates**: Automatically blocks PR merges if destructive resource replacements, stateful database drops, or excessive blast radius limits are detected.
- **Terragrunt & Monorepo Support**: Seamlessly aggregates multiple plan files across micro-stacks or directories into a unified dependency graph.
- **Zero Cloud Credentials**: Runs client-side against the self-contained JSON plan artifact. Never needs AWS, Azure, GCP, or backend access.

---

## Quick Start

Add this workflow to `.github/workflows/blast-radius.yml`:

```yaml
name: "Terraform Blast Radius"

on:
  pull_request:
    branches: [ "main" ]

permissions:
  contents: read
  pull-requests: write
  security-events: write

jobs:
  plan-and-analyze:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Set up Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_wrapper: false

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
          sarif-file: results.sarif
          upload-sarif: "true"
```

---

## Action Inputs Reference

| Input | Description | Required | Default |
| :--- | :--- | :--- | :--- |
| `plan-file` | Path to the Terraform/OpenTofu JSON plan file. Supports single files (e.g. `plan.json`) or space-separated paths (e.g. `vpc/plan.json db/plan.json`). | **Yes** | - |
| `fail-on` | Failure threshold that causes the action step to exit with code 1: `critical`, `high`, `replacement`, or `any-destroy`. | No | `""` (disabled) |
| `max-blast` | Maximum acceptable blast radius before failing the workflow. | No | `0` (disabled) |
| `max-score` | Maximum acceptable weighted blast score before failing the workflow. | No | `0` (disabled) |
| `config` | Optional custom path to `.tf-blast.yaml` policy configuration file. | No | Auto-detect |
| `out-file` | Path to write the Markdown report summary to. | No | `pr-comment.md` |
| `sarif-file` | Path to write the OASIS SARIF v2.1.0 report to for GitHub Code Scanning. | No | `""` |
| `upload-sarif` | Automatically upload the SARIF report to GitHub Code Scanning via CodeQL Action. | No | `"false"` |
| `post-pr-comment` | Whether to post or update a sticky PR comment with the analysis report. | No | `"true"` |
| `github-token` | GitHub token used to post PR comments. | No | `${{ github.token }}` |

---

## Action Outputs Reference

| Output | Description |
| :--- | :--- |
| `report-file` | File path where the Markdown summary report was saved. |
| `sarif-file` | File path where the SARIF report was saved (if `sarif-file` was set). |
| `exit-code` | Exit code of the `tf-blast` execution (`0` = pass, `1` = policy violation). |

---

## Required Workflow Permissions

Depending on which features you enable, your workflow job needs the following GitHub token permissions:

```yaml
permissions:
  contents: read          # Required: To checkout the repository code
  pull-requests: write    # Required if post-pr-comment is 'true'
  security-events: write  # Required if upload-sarif is 'true'
```

---

## Common Recipes & Examples

### 1. Basic Pull Request Comment
Post a rich summary report with collapsible blast-radius trees to every pull request:

```yaml
- name: Run tf-blast
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json
```

### 2. Strict CI Policy Gates
Block merge if any destructive resource replacement or critical data store deletion occurs:

```yaml
- name: Run tf-blast Gate
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json
    fail-on: replacement
    max-blast: 15
```

If Terraform plans to delete and recreate a resource (such as a database or security group), the action exits with code 1, rendering a policy failure banner in the PR comment and failing the check run.

### 3. Inline Annotations via GitHub Code Scanning (SARIF)
Generate SARIF diagnostics and upload them so GitHub Code Scanning decorates the PR diff with inline warnings:

```yaml
- name: Run tf-blast with Code Scanning
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json
    sarif-file: tf-blast.sarif
    upload-sarif: "true"
```

If you prefer to run the upload step manually:

```yaml
- name: Run tf-blast
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json
    sarif-file: tf-blast.sarif

- name: Upload SARIF Manually
  uses: github/codeql-action/upload-sarif@v3
  if: always()
  with:
    sarif_file: tf-blast.sarif
```

### 4. Terragrunt & Monorepo Multi-Plan Aggregation
In environments where separate plans are generated across multiple stacks or directories, pass all plan files to `plan-file`:

```yaml
- name: Generate Terragrunt Plans
  run: |
    terragrunt run-all plan -out=plan.binary
    # Convert each binary plan to JSON
    for f in $(find . -name plan.binary); do
      dir=$(dirname "$f")
      terragrunt show -json "$f" > "${dir}/plan.json"
    done

- name: Multi-Stack Blast Analysis
  uses: smford/tf-blast@v1
  with:
    plan-file: "environments/prod/vpc/plan.json environments/prod/db/plan.json environments/prod/app/plan.json"
    fail-on: critical
    max-blast: 50
```

`tf-blast` labels each resource with its source directory (e.g. `[environments/prod/vpc] aws_security_group.db`) and automatically connects cross-stack dependencies linked through IDs and ARNs.

### 5. OpenTofu Pipeline
`tf-blast` natively parses OpenTofu plan files without any binary dependency:

```yaml
- name: Set up OpenTofu
  uses: opentofu/setup-opentofu@v1

- name: OpenTofu Plan
  run: |
    tofu init -backend=false
    tofu plan -out=tofuplan.binary
    tofu show -json tofuplan.binary > plan.json

- name: Analyze OpenTofu Plan
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json
    fail-on: critical
```

### 6. Custom Risk Rules via `.tf-blast.yaml`
Point the action to a custom risk classification policy in your repository:

```yaml
- name: Run tf-blast with Custom Policy
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json
    config: ".github/policies/blast-rules.yaml"
```

---

## Troubleshooting & FAQ

### 1. `Resource not accessible by integration` error when posting PR comment
Ensure your workflow specifies:
```yaml
permissions:
  pull-requests: write
```
If your repository is an enterprise organization, ensure that **Repository Settings -> Actions -> General -> Workflow permissions** is set to "Read and write permissions".

### 2. Forked Pull Requests
When running on `pull_request` from a fork, GitHub restricts the default token to read-only permissions for security reasons. To post comments on forked PRs:
- Use the `pull_request_target` event with appropriate caution, OR
- Set `post-pr-comment: "false"` and inspect the output in workflow logs.

### 3. How do I access the Markdown report in subsequent steps?
The action provides the output `report-file`:
```yaml
- name: Run tf-blast
  id: blast
  uses: smford/tf-blast@v1
  with:
    plan-file: plan.json

- name: Archive Report
  uses: actions/upload-artifact@v4
  with:
    name: blast-radius-report
    path: ${{ steps.blast.outputs.report-file }}
```
