# Spec: Versioning, GitHub Actions CI, and Dependabot

## Goal
The goal of this task is to establish robust, professional CI/CD pipelines, automated semantic versioning, code linting, multi-architecture container builds, and dependency management. 

We will configure:
1. **GitHub Actions Test & Lint**: Go tests and `golangci-lint` running on every push or pull request targeting `develop` and `main`.
2. **Release Please (Git-Flow Production Release)**: Automated semantic versioning and changelog tracking targeting the `main` branch. Release Please maintains a rolling release pull request on `main` and mints a GitHub Release and git tag (e.g., `v1.0.0`) when merged.
3. **Multi-Arch Docker Builder**: Triggers on tags (`v*`), performing high-speed parallel native builds (`amd64` and `arm64`) using runner matrixing, publishing the final unified manifest to GitHub Container Registry (GHCR).
4. **Dependabot**: Automatically checking for actions, Go modules, and Docker parent image updates weekly.

---

## Proposed Changes

### 1. GitHub Workflows (`.github/workflows/`)

#### [NEW] [test.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/workflows/test.yml)
Create a workflow that executes tests and lints on pushes and pull requests targeting `develop` and `main`.
- **Lint Step**: Uses `golangci/golangci-lint-action` with static analysis configurations.
- **Test Step**: Runs `go test -v ./...` using Go 1.21.

#### [NEW] [release.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/workflows/release.yml)
Create a workflow that runs Release Please targeting the `main` branch.
- Uses `google-github-actions/release-please-action` with `release-type: go`.

#### [NEW] [build-image.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/workflows/build-image.yml)
Create a multi-arch container compiler workflow triggered on tags `v*`.
- **Build Job**: Builds amd64 (`ubuntu-latest`) and arm64 (`ubuntu-24.04-arm`) in parallel. Compiles with Go 1.21 (or Go 1.24/Dockerfile setup) and pushes by digest using Buildx.
- **Publish Job**: Assembles and pushes the final multi-architecture manifest.

### 2. Linting Configuration

#### [NEW] [.golangci.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.golangci.yml)
Add a `.golangci.yml` file to configure static analysis tools like `errcheck`, `gosimple`, `govet`, `staticcheck`, `unused`, and `gofmt`.

### 3. Dependency Management

#### [NEW] [dependabot.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/dependabot.yml)
Configure weekly security and version updates for Go dependencies, Docker images, and GitHub Actions.

### 4. Versioning Files

#### [NEW] [CHANGELOG.md](file:///home/nmanos/Documents/Code/discourse-labeler/CHANGELOG.md)
Add an initial `CHANGELOG.md` file following the Keep a Changelog standard.

---

## Verification Plan

### Automated Verification
1. **GitHub Actions Syntax Validation**: We will validate the YAML structures using local linters if available or verify their syntax.
2. **Make Harness Validation**: Run `make verify-harness` to ensure level 3 agent readiness compliance.
3. **Local Linting / Test Execution**: Run tests and verify Go formatting is clean.
