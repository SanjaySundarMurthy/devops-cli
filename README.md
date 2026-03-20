# 🛠️ dops — DevOps Swiss Army Knife

[![CI](https://github.com/sanjaysundarmurthy/devops-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/sanjaysundarmurthy/devops-cli/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/sanjaysundarmurthy/devops-cli)](https://goreportcard.com/report/github.com/sanjaysundarmurthy/devops-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/sanjaysundarmurthy/devops-cli)](https://github.com/sanjaysundarmurthy/devops-cli/releases)

A fast, opinionated CLI tool that validates, audits, and generates DevOps configurations from a single binary. Built in Go for speed and portability.

```
$ dops validate ./k8s/
  ⚠ WARN   [K8S-005] k8s/deployment.yaml
           Container 'api' missing resource limits/requests
  ⚠ WARN   [K8S-006] k8s/deployment.yaml
           Container 'api' missing livenessProbe
  ✖ ERROR  [K8S-008] k8s/deployment.yaml
           Container 'api' is running as privileged — security risk

  Summary: 1 errors, 2 warnings, 0 info
```

## ✨ Features

| Command | Description | Rules |
|---------|-------------|-------|
| `validate` | Validate K8s manifests, Dockerfiles, Helm charts, CI configs | 25+ rules |
| `audit` | Security & best-practice audits (K8s, Docker, Terraform) | 20+ rules |
| `generate` | Generate production-ready configs from templates | 4 generators |
| `check` | Pre-deploy health checks across your stack | 8 checks |

### Supported File Types

- **Kubernetes** — Deployments, Services, StatefulSets, DaemonSets, CronJobs
- **Dockerfiles** — Multi-stage builds, security, efficiency
- **Helm Charts** — Chart.yaml, values.yaml, templates structure
- **GitHub Actions** — Workflow syntax, action pinning, timeouts
- **Docker Compose** — Service configs, health checks, restart policies
- **Terraform** — Security patterns, hardcoded secrets, tags

## 🚀 Installation

### From Source
```bash
go install github.com/sanjaysundarmurthy/devops-cli/cmd/dops@latest
```

### Binary Downloads
Download pre-built binaries from [Releases](https://github.com/sanjaysundarmurthy/devops-cli/releases):

```bash
# Linux (amd64)
curl -sL https://github.com/sanjaysundarmurthy/devops-cli/releases/latest/download/dops_linux_amd64.tar.gz | tar xz
sudo mv dops /usr/local/bin/

# macOS (Apple Silicon)
curl -sL https://github.com/sanjaysundarmurthy/devops-cli/releases/latest/download/dops_darwin_arm64.tar.gz | tar xz
sudo mv dops /usr/local/bin/
```

## 📖 Usage

### Validate Configurations

```bash
# Validate a single file
dops validate Dockerfile
dops validate k8s/deployment.yaml

# Validate an entire directory recursively
dops validate ./infrastructure/

# JSON output for CI/CD integration
dops validate ./k8s/ -o json
```

### Security Audit

```bash
# Audit a Kubernetes manifest
dops audit k8s/deployment.yaml

# Audit entire infrastructure directory
dops audit ./infrastructure/

# Example output
$ dops audit deployment.yaml
  🔴 CRIT  [SEC-K8S-006] deployment.yaml
           Container 'app' running in privileged mode
           💡 Remove privileged: true — this grants full host access

  🟡 MED   [SEC-K8S-005] deployment.yaml
           Container 'app' missing securityContext
           💡 Add securityContext with runAsNonRoot, readOnlyRootFilesystem
```

### Generate Configs

```bash
# Generate a production Dockerfile
dops generate dockerfile --lang go
dops generate dockerfile --lang python --file Dockerfile

# Generate GitHub Actions CI/CD pipeline
dops generate github-actions --lang go --file .github/workflows/ci.yml

# Generate Kubernetes manifests
dops generate k8s-deploy --name myapi --image myapi:v1.0

# Generate Docker Compose stacks
dops generate docker-compose --stack web
dops generate docker-compose --stack monitoring
```

### Pre-Deploy Checks

```bash
# Run pre-deploy checks on a project
dops check ./my-project/

# Check output
$ dops check ./my-project/
  ✅ [required-files] (README.md) README.md found
  ❌ [required-files] (.gitignore) Repository should have .gitignore
  ✅ [yaml-syntax] All YAML files have valid syntax
  ❌ [secret-files] (.env) File '.env' found — ensure it's in .gitignore

  📊 Results: 2 passed, 2 failed out of 4 checks
```

## 🏗️ Architecture

```
devops-cli/
├── cmd/dops/              # CLI entrypoint
│   └── main.go
├── internal/
│   ├── cli/               # Cobra command definitions
│   │   ├── root.go        # Root command + flags
│   │   ├── validate.go    # validate command
│   │   ├── audit.go       # audit command
│   │   ├── generate.go    # generate subcommands
│   │   ├── check.go       # check command
│   │   └── version.go     # version command
│   └── core/              # Business logic
│       ├── validators/    # File validation engine (25+ rules)
│       ├── auditor/       # Security audit engine (20+ rules)
│       ├── generator/     # Config generation templates
│       └── checker/       # Pre-deploy health checks
├── .github/workflows/     # CI/CD pipeline
├── .goreleaser.yml        # Cross-platform release config
├── go.mod
└── README.md
```

## 📋 Validation Rules

### Dockerfile Rules (DF-001 → DF-009)
| Rule | Severity | Description |
|------|----------|-------------|
| DF-001 | Warning | Avoid `:latest` tag |
| DF-002 | Info | Prefer alpine/distroless base images |
| DF-003 | Warning | Use `--no-install-recommends` |
| DF-004 | Error | Don't pipe curl to shell |
| DF-005 | Warning | Use COPY instead of ADD |
| DF-006 | Error | No hardcoded secrets in ENV |
| DF-007 | Error | Must have FROM instruction |
| DF-008 | Warning | Add USER instruction (non-root) |
| DF-009 | Info | Add HEALTHCHECK |

### Kubernetes Rules (K8S-001 → K8S-009)
| Rule | Severity | Description |
|------|----------|-------------|
| K8S-001 | Error | Missing `kind` field |
| K8S-003 | Warning | Missing labels |
| K8S-004 | Warning | No namespace specified |
| K8S-005 | Warning | Missing resource limits/requests |
| K8S-006 | Warning | Missing livenessProbe |
| K8S-007 | Warning | Missing readinessProbe |
| K8S-008 | Error | Privileged container |
| K8S-009 | Warning | Using `:latest` or untagged image |

### Security Audit Rules (SEC-*)
| Rule | Severity | Description |
|------|----------|-------------|
| SEC-K8S-001 | Critical | hostNetwork enabled |
| SEC-K8S-002 | Critical | hostPID enabled |
| SEC-K8S-006 | Critical | Privileged mode |
| SEC-DF-003 | High | SSH port exposed |
| SEC-TF-001 | High | Publicly accessible resource |
| SEC-TF-002 | High | Encryption disabled |

## 🔌 CI/CD Integration

### GitHub Actions
```yaml
- name: Validate Infrastructure
  run: |
    dops validate ./k8s/ -o json > validation.json
    dops audit ./k8s/ -o json > audit.json
```

### Pre-commit Hook
```bash
#!/bin/sh
dops validate . || exit 1
dops audit . || exit 1
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feat/amazing-feature`)
3. Run tests (`go test ./...`)
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feat/amazing-feature`)
6. Open a Pull Request

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

## 🔗 Related Tools

Part of the **DevOps Principal Mastery** toolkit:
- [terraform-modules](https://github.com/sanjaysundarmurthy/terraform-modules) — Production-ready Terraform module library
- [docker-compose-templates](https://github.com/sanjaysundarmurthy/docker-compose-templates) — Ready-to-use dev environments
- [helm-charts](https://github.com/sanjaysundarmurthy/helm-charts) — Kubernetes deployment templates
