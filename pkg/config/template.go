package config

// DefaultTemplate is the starter .tf-blast.yaml policy configuration template.
const DefaultTemplate = `# yaml-language-server: $schema=https://raw.githubusercontent.com/smford/tf-blast/main/schema/tf-blast.schema.json
$schema: "https://raw.githubusercontent.com/smford/tf-blast/main/schema/tf-blast.schema.json"

# CI/CD Policy Failure Gates (Exit Code 1)
# Options: "critical", "high", "medium", "replacement", "any-destroy"
fail_on: "critical"

# Maximum acceptable total blast radius count (0 disables limit)
max_blast: 25

# Maximum acceptable weighted blast score (0 disables limit)
max_score: 50

# Custom Severity Classification Rules (overrides built-in heuristic tiers)
rules:
  critical:
    - "*rds*"
    - "*database*"
    - "*dynamodb*"
    - "*s3_bucket*"
    - "*persistent_volume*"
    - "*blob*"
    - "*spanner*"
    - "*bigtable*"
  high:
    - "*security_group*"
    - "*route_table*"
    - "*iam_role*"
    - "*firewall*"
    - "*policy*"
    - "*acl*"
  medium:
    - "*instance*"
    - "*ecs_service*"
    - "*alb*"
    - "*elb*"
    - "*kubernetes_deployment*"
  low:
    - "*tag*"
    - "*log_group*"
    - "*metric_alarm*"
    - "*alert*"
    - "*sns_topic*"

# Resource addresses or types excluded from blast analysis
ignore_resources:
  - "*null_resource*"
  - "*random_id*"
  - "*random_password*"
  - "*time_sleep*"
`
