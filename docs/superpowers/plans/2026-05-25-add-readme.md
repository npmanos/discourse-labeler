# Add high quality README.md Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a comprehensive, premium, developer-friendly README.md at the root of the project explaining the concept, quick start, configuration, architecture, and development workflow.

**Architecture:** We follow a quick-start-first approach. We introduce the core meta-discourse concept alongside a 2-sentence tech stack summary. This is followed by a copy-paste setup path, a detailed configuration reference table, a Mermaid pipeline sequence diagram, and our Git-Flow development rules.

**Tech Stack:** Markdown, Mermaid, Go, Docker Compose, llama.cpp.

---

### Task 1: Create README.md Header, Conceptual Intro, and Quick Start

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write the initial README.md content**
  Write the header, conceptual definition of Meta-Discourse, tech stack overview, and the Quick Start guide.
  
  Code to write to `README.md`:
  ```markdown
  # Bluesky Meta-Discourse Labeler 🏷️🤖

  The **Bluesky Meta-Discourse Labeler** is a high-performance Go-based background daemon designed to classify real-time posts from the Bluesky feed for "meta-discourse" and cryptographically sign and emit labels to an Ozone moderation instance.

  The system leverages a hybrid cloud-and-edge architecture, using Go to orchestrate firehose ingestion from [Graze Contrails](https://graze.social) and post context hydration via [Microcosm Slingshot](https://constellation.microcosm.blue). Hydrated posts are then classified locally using Gemma 4 via `llama.cpp`, with positive matches cryptographically signed using the ATProto Indigo library and emitted to an Ozone moderation server.

  ## Conceptual Definition: Meta-Discourse

  ### What is Meta-Discourse? (TRUE)
  Posts evaluating, criticizing, or theorizing about the cultural and social experience of Bluesky itself. This includes:
  - Debating the platform's "vibes," echo chambers, or user base behaviors.
  - Comparing the social experience, engagement dynamics, or toxicity of Bluesky versus X (Twitter) or other platforms.
  - Complaining about the types of conversations people have (e.g., "too much drama", "people subtweeting").
  - Subtweets or reactions regarding Bluesky's community culture.

  ### What is NOT Meta-Discourse? (FALSE)
  - Technical discussions about building on ATProto, creating feeds, using APIs, or hosting infrastructure.
  - Announcements or discussions of new Bluesky application features (e.g., "DMs are live").
  - General political, social, or pop culture arguments (even if heated or referencing platform moderation), as long as they do not explicitly analyze platform culture.
  - Passing usage of platform terms like "skeet" or "repost".

  ## Quick Start

  Get the stack up and running locally in three steps:

  ### 1. Launch the Inference Server
  Ensure you have Docker and Docker Compose installed. Run the local `llama.cpp` container serving `gemma-4-e2b`:
  ```bash
  docker compose up -d
  ```

  ### 2. Configure Environment Variables
  Copy the example environment file and fill in required keys:
  ```bash
  cp .env.example .env
  ```
  At a minimum, configure the following variables in `.env`:
  - `GRAZE_FEED_URI` (AT-URI of the feed to listen to)
  - `LABELER_DID` (Your cryptographic labeler DID)
  - `OZONE_ADMIN_TOKEN` (Your Ozone auth token)

  ### 3. Run the Daemon
  Start the background classification pipeline:
  ```bash
  make run
  ```
  ```

- [ ] **Step 2: Commit the initial file**
  Run:
  ```bash
  git add README.md
  git commit -m "feat: add README header, conceptual definition, and quick start"
  ```

---

### Task 2: Add Configuration Reference

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Append Configuration Reference section**
  Append the environment variables table detailing all runtime configuration options to `README.md`.

  Markdown to append:
  ```markdown
  ## Configuration Reference

  The daemon is configured entirely through environment variables or a `.env` file at the root.

  | Variable | Default | Description |
  |---|---|---|
  | `PORT` | `8081` | Port to run the status/health server on. |
  | `LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`). |
  | `CURSOR_FILE_PATH` | `./data/cursor.json` | Path to store the firehose replication cursor state. |
  | `CURSOR_REWIND_SECONDS`| `10` | Number of seconds to rewind firehose state upon reconnection. |
  | `HYDRATION_WORKERS` | `10` | Concurrent worker count for fetching parent/quoted post context. |
  | `CLASSIFICATION_WORKERS`| `4` | Concurrent worker count for running local LLM inference. |
  | `GRAZE_FEED_URI` | *Required* | The AT-URI of the Bluesky feed to ingest events from. |
  | `CONTRAILS_WS_URL` | `wss://api.graze.social/app/contrail` | Graze Contrails event WebSocket endpoint. |
  | `SLINGSHOT_URL` | `https://slingshot.microcosm.blue` | Microcosm Slingshot edge cache RPC endpoint. |
  | `LLM_ENDPOINT` | `http://localhost:8080/v1/` | Base URL of OpenAI-compatible inference server. |
  | `LLM_MODEL` | `google/gemma-4-e2b-gguf` | LLM model identifier. |
  | `LLM_TEMPERATURE` | `0.0` | Sampling temperature for classification (keep at `0.0` for determinism). |
  | `OZONE_ENDPOINT` | `http://localhost:3000` | Target Ozone moderation server API endpoint. |
  | `LABELER_DID` | *Required* | Cryptographic DID of the network labeler service. |
  | `OZONE_ADMIN_TOKEN` | *Required* | Authentication token for Ozone server write-access. |
  | `DRY_RUN` | `false` | If `true`, classifications are computed but labels are not sent to Ozone. |
  | `LLM_SYSTEM_PROMPT` | *(Empty)* | Raw system prompt override string. |
  | `LLM_SYSTEM_PROMPT_PATH`| *(Empty)* | File path to load custom system prompt from. |
  ```

- [ ] **Step 2: Commit the configuration reference updates**
  Run:
  ```bash
  git add README.md
  git commit -m "docs: add configuration reference to README"
  ```

---

### Task 3: Add System Architecture & Data Flow

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Append System Architecture and Component Map**
  Append the system architecture section containing a Mermaid sequence flow and component mapping table to `README.md`.

  Markdown to append:
  ```markdown
  ## System Architecture & Data Flow

  The labeler is built for low-latency firehose filtering and processing using a multi-worker async pipeline:

  ```mermaid
  sequenceDiagram
      autonumber
      participant C as Contrails (WebSocket)
      participant I as Ingestion Worker
      participant L as LRU Cache
      participant H as Slingshot Hydrator (REST)
      participant LLM as llama.cpp (Gemma 4)
      participant O as Ozone (Indigo SDK)

      C->>I: Event Stream (Raw Post URI & Text)
      critical Check Duplication
          I->>L: Query post URI
      end
      alt Is Cached / Duplicate
          I->>I: Drop Event
      else Is New Event
          I->>H: Request parent & quoted context
          H-->>I: Return hydrated text blocks
          I->>LLM: JSON Schema Prompt (is_meta_discourse?)
          LLM-->>I: Return JSON Output (boolean + logprobs)
          alt is_meta_discourse == true && probability >= 0.85
              I->>O: Sign & emit com.atproto.label.defs#label
          else probability < 0.85
              I->>I: Discard / Log
          end
      end
  ```

  ### Component Directory Map

  - **Config Loader (`internal/config/`)**: Decodes and validates environment variables and custom runtime system prompt overrides.
  - **Pipeline Coordinator (`internal/pipeline/`)**: Directs multi-worker async flow, caching, cursor state persistence, and event processing.
  - **Services Package (`internal/services/`)**:
    - `contrails.go`: Subscribes to filtered firehose WebSocket stream.
    - `slingshot.go`: Hydrates parent & quote context from Edge RPC cache.
    - `classifier.go`: Encodes posts into custom XML schemas and executes inference against `llama.cpp`.
    - `ozone.go`: Signs and broadcasts cryptographic labels.
  ```

- [ ] **Step 2: Commit architectural updates**
  Run:
  ```bash
  git add README.md
  git commit -m "docs: add architecture section and Mermaid flow to README"
  ```

---

### Task 4: Add Development & Contribution Guide

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Append Contribution instructions and Make commands**
  Append the development rules, `make` commands, and Git-Flow branching instructions.

  Markdown to append:
  ```markdown
  ## Development & Contribution

  ### Prerequisites
  - Go 1.21+
  - Docker & Docker Compose

  ### Tooling & Make Commands

  We package development utilities inside the `Makefile`:

  ```bash
  # Build the labeler binary
  make build

  # Run all unit tests
  make test

  # Run the daemon locally
  make run

  # Verify agent-readiness and harness rules
  make verify-harness
  ```

  ### Collaboration Guidelines (Git-Flow)
  We enforce the **Git-Flow** branching model. Do not commit directly to primary branches.
  1. Always synchronize your branch from the upstream `develop` branch.
  2. Create a dedicated feature branch:
     ```bash
     git checkout develop
     git pull origin develop
     git checkout -b feature/your-feature-name
     ```
  3. Commit with structured Conventional Commit format (e.g. `feat: ...`, `fix: ...`).
  4. Ensure your branch passes the automated harness validation before opening a PR:
     ```bash
     make verify-harness
     ```
  5. Target all Pull Requests to the `develop` integration branch.
  
  For comprehensive agent compliance and repository structural rules, see [AGENTS.md](AGENTS.md).
  ```

- [ ] **Step 2: Commit contribution updates**
  Run:
  ```bash
  git add README.md
  git commit -m "docs: add development and contribution guidelines to README"
  ```

---

### Task 5: Final Verification & Commit Push

**Files:**
- Modify: `README.md`
- Verify: `make verify-harness`

- [ ] **Step 1: Review README.md markdown and links**
  Ensure all links (`https://graze.social`, `https://constellation.microcosm.blue`, `AGENTS.md`) are accurate and the Mermaid diagram parses correctly.

- [ ] **Step 2: Run verification suite**
  Run: `make verify-harness`
  Expected: Levels 1-3 COMPLETE with 0 errors.

- [ ] **Step 3: Run project tests**
  Run: `make test`
  Expected: PASS

- [ ] **Step 4: Push to Remote Origin**
  Push the branch to upstream as required by `AGENTS.md` Rule 2.
  Run: `git push -u origin feature/add-readme`
  Expected: Successfully pushed to origin.
