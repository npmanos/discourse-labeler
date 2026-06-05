# Spec: Versioning, GitHub Actions CI, and Dependabot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish proper versioning tooling, CI testing & linting workflows, high-speed multi-arch Docker builds, and Dependabot package management.

**Architecture:** We configure Release Please targeting `main` with automated tag merge-back. We utilize a Go runtime version file `version.go` with inline annotations for automatic version bumping. Code quality is guarded on `develop` and `main` branches by running parallel Go tests and `golangci-lint` in CI. Multi-arch images are built on tag releases using a parallel native runner matrix, then assembled and published to GHCR.

**Tech Stack:** GitHub Actions, golangci-lint, Release Please, Docker Compose, Go.

---

### Task 1: Versioning Manifests and Code Files

**Files:**
- Create: `release-please-config.json`
- Create: `.release-please-manifest.json`
- Create: `VERSION`
- Create: `internal/config/version.go`
- Create: `CHANGELOG.md`

- [ ] **Step 1: Write `release-please-config.json`**
  Write the manifest configuration mapping the root component with custom Go updaters.
  
  Code for `release-please-config.json`:
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

- [ ] **Step 2: Write `.release-please-manifest.json`**
  Write the initial starting semantic version mapping.
  
  Code for `.release-please-manifest.json`:
  ```json
  {
    ".": "0.1.0"
  }
  ```

- [ ] **Step 3: Write `VERSION`**
  Write the plain-text VERSION file for generic tracking.
  
  Code for `VERSION`:
  ```text
  0.1.0
  ```

- [ ] **Step 4: Write `internal/config/version.go`**
  Create a file that injects the current semver compile-time string to expose Version to the binary.
  
  Code for `internal/config/version.go`:
  ```go
  package config

  // Version is the current semantic version of the labeler compiled binary.
  // It is automatically bumped by Release Please on merge to main.
  const Version = "0.1.0" // x-release-please-version
  ```

- [ ] **Step 5: Write `CHANGELOG.md`**
  Create an initial `CHANGELOG.md` following Keep a Changelog.
  
  Code for `CHANGELOG.md`:
  ```markdown
  # Changelog

  All notable changes to this project will be documented in this file.

  The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
  and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

  ## [Unreleased]
  ```

- [ ] **Step 6: Commit versioning files**
  Run:
  ```bash
  git add release-please-config.json .release-please-manifest.json VERSION internal/config/version.go CHANGELOG.md
  git commit -m "feat: add release-please manifest configurations, version.go, and initial changelog"
  ```

---

### Task 2: Test & Lint Workflow and Lint Configuration

**Files:**
- Create: `.github/workflows/test.yml`
- Create: `.golangci.yml`

- [ ] **Step 1: Write `.golangci.yml`**
  Write the golangci-lint configuration.
  
  Code for `.golangci.yml`:
  ```yaml
  run:
    timeout: 5m
    tests: true

  linters:
    disable-all: true
    enable:
      - errcheck
      - gosimple
      - govet
      - ineffassign
      - staticcheck
      - unused
      - gofmt
      - goimports
      - misspell

  linters-settings:
    govet:
      enable-all: true
    gofmt:
      simplify: true
  ```

- [ ] **Step 2: Write `.github/workflows/test.yml`**
  Write the Actions workflow for testing and linting.
  
  Code for `.github/workflows/test.yml`:
  ```yaml
  name: Test and Lint

  on:
    push:
      branches: [develop, main]
    pull_request:
      branches: [develop, main]

  permissions:
    contents: read

  jobs:
    lint:
      name: Lint Codebase
      runs-on: ubuntu-latest
      steps:
        - name: Checkout
          uses: actions/checkout@v4

        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version: "1.21"
            cache: true

        - name: Run golangci-lint
          uses: golangci/golangci-lint-action@v6
          with:
            version: v1.54

    test:
      name: Run Go Tests
      runs-on: ubuntu-latest
      steps:
        - name: Checkout
          uses: actions/checkout@v4

        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version: "1.21"
            cache: true

        - name: Run Tests
          run: go test -v ./...
  ```

- [ ] **Step 3: Commit test & lint files**
  Run:
  ```bash
  git add .golangci.yml .github/workflows/test.yml
  git commit -m "ci: add test and lint workflow with golangci-lint configuration"
  ```

---

### Task 3: Release Please Workflow with Automated Sync-Back

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write `.github/workflows/release.yml`**
  Write the Release Please Action config targeting `main` with an automated sync-back script.
  
  Code for `.github/workflows/release.yml`:
  ```yaml
  name: Release Please

  on:
    push:
      branches:
        - main

  permissions:
    contents: write
    pull-requests: write

  jobs:
    release-please:
      runs-on: ubuntu-latest
      outputs:
        releases_created: ${{ steps.release.outputs.releases_created }}
        version: ${{ steps.release.outputs.version }}
        tag_name: ${{ steps.release.outputs.tag_name }}
      steps:
        - name: Run Release Please
          id: release
          uses: google-github-actions/release-please-action@v4
          with:
            config-file: release-please-config.json
            manifest-file: .release-please-manifest.json

        - name: Checkout Code
          if: ${{ steps.release.outputs.releases_created }}
          uses: actions/checkout@v4
          with:
            fetch-depth: 0

        - name: Automated Sync-Back to develop
          if: ${{ steps.release.outputs.releases_created }}
          run: |
            git config --global user.name "github-actions[bot]"
            git config --global user.email "github-actions[bot]@users.noreply.github.com"
            
            # Fetch all branches
            git fetch origin develop
            
            # Switch to develop
            git checkout develop
            
            # Merge main's tag back into develop to keep changelog/version in sync
            git merge --no-ff -m "chore: merge release v${{ steps.release.outputs.version }} back into develop [skip ci]" origin/main
            
            # Push changes back to origin
            git push origin develop
  ```

- [ ] **Step 2: Commit release workflow**
  Run:
  ```bash
  git add .github/workflows/release.yml
  git commit -m "ci: add release-please workflow targeting main with automated sync-back"
  ```

---

### Task 4: Multi-Arch Container Image Builder

**Files:**
- Create: `.github/workflows/build-image.yml`

- [ ] **Step 1: Write `.github/workflows/build-image.yml`**
  Create the multi-architecture builder workflow using the matrix runner and publish digest assembly flow.
  
  Code for `.github/workflows/build-image.yml`:
  ```yaml
  name: Build container image

  on:
    push:
      tags:
        - "v*"

  env:
    REGISTRY: ghcr.io
    IMAGE_NAME: ${{ github.repository }}

  jobs:
    test:
      runs-on: ubuntu-latest
      permissions:
        contents: read
      steps:
        - name: Checkout
          uses: actions/checkout@v4

        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version: "1.21"
            cache: true

        - name: Run Tests
          run: go test -v ./...

    build:
      needs: test
      runs-on: ${{ matrix.runner }}
      permissions:
        contents: read
        packages: write
      strategy:
        fail-fast: false
        matrix:
          include:
            - family: linux
              arch: amd64
              runner: ubuntu-latest
            - family: linux
              arch: arm64
              runner: ubuntu-24.04-arm
      steps:
        - name: Checkout
          uses: actions/checkout@v4

        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version: "1.21"
            cache: true

        - name: Login to container registry
          uses: docker/login-action@v3
          with:
            registry: ${{ env.REGISTRY }}
            username: ${{ github.actor }}
            password: ${{ secrets.GITHUB_TOKEN }}

        - name: Docker meta
          id: meta-arch
          uses: docker/metadata-action@v5
          with:
            images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}

        - name: Set up QEMU
          uses: docker/setup-qemu-action@v3

        - name: Set up Docker Buildx
          id: setup-buildx
          uses: docker/setup-buildx-action@v3

        - name: Build and push by digest
          id: docker-build
          uses: docker/build-push-action@v6
          with:
            cache-from: type=gha
            cache-to: type=gha,mode=max
            context: .
            tags: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
            labels: ${{ steps.meta-arch.outputs.labels }}
            annotations: ${{ steps.meta-arch.outputs.annotations }}
            platforms: ${{ matrix.family }}/${{ matrix.arch }}
            outputs: type=image,push-by-digest=true,name-canonical=true,push=true

        - name: Export digest
          run: |
            mkdir -p ${{ runner.temp }}/digests
            digest="${{ steps.docker-build.outputs.digest }}"
            touch "${{ runner.temp }}/digests/${digest#sha256:}"

        - name: Upload digest
          uses: actions/upload-artifact@v4
          with:
            name: digests-${{ matrix.family }}-${{ matrix.arch }}
            path: ${{ runner.temp }}/digests/*
            if-no-files-found: error
            retention-days: 1

    publish-multiarch:
      runs-on: ubuntu-latest
      needs: build
      permissions:
        contents: read
        packages: write
      steps:
        - name: Download digests
          uses: actions/download-artifact@v4
          with:
            path: ${{ runner.temp }}/digests
            pattern: digests-*
            merge-multiple: true

        - name: Login to container registry
          uses: docker/login-action@v3
          with:
            registry: ${{ env.REGISTRY }}
            username: ${{ github.actor }}
            password: ${{ secrets.GITHUB_TOKEN }}

        - name: Docker meta
          id: meta-final
          uses: docker/metadata-action@v5
          with:
            images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
            tags: |
              type=semver,pattern={{version}}
              type=semver,pattern={{major}}.{{minor}}
              type=semver,pattern={{major}},enable=${{ !startsWith(github.ref, 'refs/tags/v0.') }}
              type=edge

        - name: Create multiarch manifest and push
          working-directory: ${{ runner.temp }}/digests
          run: |
            docker buildx imagetools create $(jq -cr '.tags | map("-t " + .) | join(" ")' <<< "$DOCKER_METADATA_OUTPUT_JSON") \
            $(printf '${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@sha256:%s ' *)

        - name: Inspect image
          run: |
            docker buildx imagetools inspect ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ steps.meta-final.outputs.version }}
  ```

- [ ] **Step 2: Commit build image workflow**
  Run:
  ```bash
  git add .github/workflows/build-image.yml
  git commit -m "ci: add multi-arch build-image workflow utilizing runner matrix"
  ```

---

### Task 5: Dependabot Configuration

**Files:**
- Create: `.github/dependabot.yml`

- [ ] **Step 1: Write `.github/dependabot.yml`**
  Write Dependabot rules mapping updates for actions, gomod, and docker.
  
  Code for `.github/dependabot.yml`:
  ```yaml
  version: 2
  updates:
    - package-ecosystem: "github-actions"
      directory: "/"
      schedule:
        interval: "weekly"
        day: "sunday"

    - package-ecosystem: "gomod"
      directory: "/"
      schedule:
        interval: "weekly"
        day: "sunday"

    - package-ecosystem: "docker"
      directory: "/"
      schedule:
        interval: "weekly"
        day: "sunday"
  ```

- [ ] **Step 2: Commit dependabot config**
  Run:
  ```bash
  git add .github/dependabot.yml
  git commit -m "ci: add dependabot weekly updates configuration"
  ```

---

### Task 6: Verification and Remote Sync

**Files:**
- Verify: `make verify-harness`
- Verify: `go test -v ./...`

- [ ] **Step 1: Validate Harness Readiness**
  Run: `make verify-harness`
  Expected: Level 3 COMPLETE with 0 errors.

- [ ] **Step 2: Verify Go tests pass**
  Run: `go test -v ./...`
  Expected: PASS

- [ ] **Step 3: Push changes to Origin**
  Run: `git push origin develop`
  Expected: Successfully pushed to develop.
