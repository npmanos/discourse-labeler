# Spec: Versioning, GitHub Actions CI, and Dependabot

## Goal
The goal of this task is to establish robust, professional CI/CD pipelines, automated semantic versioning, code linting, multi-architecture container builds, and dependency management. 

We will configure:
1. **GitHub Actions Test & Lint**: Go tests and `golangci-lint` running on every push or pull request targeting `develop` and `main`.
2. **Release Please (Git-Flow Production Release)**: Automated semantic versioning and changelog tracking targeting the `main` branch. Release Please maintains a rolling release pull request on `main` and mints a GitHub Release and git tag (e.g., `v1.0.0`) when merged.
3. **Automated Sync-Back**: Upon merging a Release PR on `main`, the CI workflow automatically merges the newly created release tag/commit back into `develop` to prevent branch drift.
4. **Multi-Arch Docker Builder**: Triggers on tags (`v*`), performing high-speed parallel native builds (`amd64` and `arm64`) using runner matrixing, publishing the final unified manifest to GitHub Container Registry (GHCR).
5. **Dependabot**: Automatically checking for actions, Go modules, and Docker parent image updates weekly.

---

## Proposed Changes

### 1. Versioning Manifests & Code Files

#### [NEW] [release-please-config.json](file:///home/nmanos/Documents/Code/discourse-labeler/release-please-config.json)
Create the configuration mapping the root component with custom updaters:
```json
{
  "$schema": "https://raw.githubusercontent.com/googleapis/release-please/main/schemas/config.json",
  "packages": {
    ".": {
      "release-type": "go",
      "extra-files": [
        "internal/config/version.go",
        "VERSION"
      ]
    }
  }
}
```

#### [NEW] [.release-please-manifest.json](file:///home/nmanos/Documents/Code/discourse-labeler/.release-please-manifest.json)
Define the initial starting semantic version:
```json
{
  ".": "0.1.0"
```

#### [NEW] [VERSION](file:///home/nmanos/Documents/Code/discourse-labeler/VERSION)
Create the plain-text VERSION file for generic tracking:
```text
0.1.0
```

#### [NEW] [version.go](file:///home/nmanos/Documents/Code/discourse-labeler/internal/config/version.go)
Create a file that injects the current semver compile-time string:
```go
package config

// Version is the current semantic version of the labeler compiled binary.
// It is automatically bumped by Release Please on merge to main.
const Version = "0.1.0" // x-release-please-version
```

#### [NEW] [CHANGELOG.md](file:///home/nmanos/Documents/Code/discourse-labeler/CHANGELOG.md)
Add an initial `CHANGELOG.md` file following the Keep a Changelog standard.

---

### 2. GitHub Workflows (`.github/workflows/`)

#### [NEW] [test.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/workflows/test.yml)
Create a workflow that executes tests and lints on pushes and pull requests targeting `develop` and `main`.
- **Lint Step**: Uses `golangci/golangci-lint-action` with `.golangci.yml` configurations.
- **Test Step**: Runs `go test -v ./...` using Go 1.21.

#### [NEW] [release.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/workflows/release.yml)
Create a workflow that runs Release Please targeting the `main` branch.
- Uses `google-github-actions/release-please-action` with configuration point.
- **Automated Sync-Back Step**: After a release is created (i.e. `releases_created` is `true`), checks out `develop`, merges the tag/commit, and pushes it back to origin automatically.

#### [NEW] [build-image.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/workflows/build-image.yml)
Create a multi-arch container compiler workflow triggered on tags `v*`.
- **Build Job**: Builds amd64 (`ubuntu-latest`) and arm64 (`ubuntu-24.04-arm`) in parallel using native builders. Pushes by digest using Buildx.
- **Publish Job**: Assembles and pushes the final multi-architecture manifest.

---

### 3. Static Analysis & Dependency Configs

#### [NEW] [.golangci.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.golangci.yml)
Add a `.golangci.yml` file to configure static analysis tools like `errcheck`, `gosimple`, `govet`, `staticcheck`, `unused`, and `gofmt`.

#### [NEW] [dependabot.yml](file:///home/nmanos/Documents/Code/discourse-labeler/.github/dependabot.yml)
Configure weekly security and version updates for Go dependencies, Docker images, and GitHub Actions.

---

## Verification Plan

### Automated Verification
1. **GitHub Actions Syntax Validation**: We will validate YAML syntax and check formatting.
2. **Make Harness Validation**: Run `make verify-harness` to ensure level 3 agent readiness compliance.
3. **Local Linting / Test Execution**: Run Go tests locally to ensure there are no regressions.
