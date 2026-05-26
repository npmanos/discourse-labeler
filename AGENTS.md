# Agent Collaboration & Git Guidelines

Welcome to the **Bluesky Meta-Discourse Labeler** project. This repository enforces strict guidelines for AI agents and human collaborators to ensure clean development history, code isolation, and synchronization.

---

## Repo Structure

- `cmd/` — Application entrypoints (e.g., [cmd/labeler/main.go](cmd/labeler/main.go))
- `internal/` — Core implementation packages (config, pipeline, services)
- `docs/` — Documentation (see [ARCHITECTURE.md](docs/ARCHITECTURE.md))
- `scripts/` — Helper scripts (including [verify-harness.sh](scripts/verify-harness.sh))
- `.github/workflows/` — CI/CD workflows, including the multi-arch builder [build-image.yml](.github/workflows/build-image.yml) and [harness-verify.yml](.github/workflows/harness-verify.yml)
- `.golangci.yml` — Code linting rules configuration


---

## Commands

```bash
# Build the labeler daemon
go build -o bin/labeler ./cmd/labeler

# Run the test suite
go test -v ./...

# Verify agent harness consistency
make verify-harness
```

---

## Rules

### 1. Branching Model: Git-Flow

We adhere strictly to the **git-flow** branching model for all project modifications. Do not commit directly to primary branches.

#### Primary Branches
* **`main`:** Production-ready code. Matches the latest stable release.
* **`develop`:** Integration branch for the latest development changes. Features are merged here to prepare for release.

#### Supporting Branches
* **`feature/<name>`:** Used to build specific features or tasks.
  * *Workflow:* Branch off `develop`, build the feature, test it, push changes, and open a Pull Request back into `develop`.
* **`release/<version>`:** Prepares a release from `develop` to `main`.
* **`hotfix/<name>`:** Emergency fixes targeting `main` directly.

#### Agent Workflow Commands
Before starting a new task, always pull the latest changes from `develop` and create a dedicated feature branch:

```bash
# Ensure you are on develop and fully updated
git checkout develop
git pull origin develop

# Create your dedicated feature branch
git checkout -b feature/your-feature-name
```

### 2. Synchronization Requirement: Push All Changes

All AI agents and collaborators **must push their commits** to the remote origin. This guarantees that your changes are backed up, visible to peers, and integrated into CI/CD pipelines.

When committing your work:
1. Write clear, structured commit messages adhering to standard conventions (e.g. Conventional Commits: `feat: add ...`, `fix: resolve ...`).
2. Run your verification tests locally first.
3. Push your branch to the remote repository:

```bash
# Stage and commit your changes
git add .
git commit -m "feat: implement feature xyz"

# Push to the remote origin
git push -u origin feature/your-feature-name
```

### 3. Pull Requests & Code Review

Once a task is complete and thoroughly tested:
1. Ensure your branch is fully pushed.
2. Open a Pull Request targeting `develop`.
3. Provide a clear walkthrough of the changes, verification outputs, and execution logs inside the PR description.

### 4. CI/CD & Linter Compatibility

To ensure linting runs smoothly in the GitHub Actions pipeline:
1. **Go Version Alignment**: The targeted Go language version in `go.mod` MUST be compatible with the compiler used to build `golangci-lint` (e.g. Go `1.24.0` for linter `v1.64.8`). If the linter version is older than your Go target, downgrade `go.mod` and CI configs to match.
2. **Exclude Pedantic Linters**: When configuring `govet` with `enable-all: true` in `.golangci.yml`, always explicitly disable the pedantic `fieldalignment` (struct packing) and `shadow` (variable shadowing) analyzers. These micro-optimizations harm code readability and cause late-stage CI failures.
3. **Format Imports**: Always execute `go run golang.org/x/tools/cmd/goimports@latest -w .` alongside `gofmt` to prevent `goimports` ordering failures in CI.

### 5. Harness Drift Prevention

When modifying CI/CD workflows (`.github/workflows/*`), Makefiles, or build manifests, a minor non-functional update (such as a comment, timestamp, or note) MUST be made to `AGENTS.md` in the exact same commit to satisfy the Level 3 drift verification checks of `make verify-harness`.

---

## References

- [Architecture Plan](docs/ARCHITECTURE.md)
- [Design Specs](docs/superpowers/specs/)
- [Implementation Plans](docs/superpowers/plans/)

<!-- Last updated: May 2026 to resolve Release Please deprecation and llama model filenames -->

