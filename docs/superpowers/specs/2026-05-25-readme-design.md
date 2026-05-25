# Spec: High-Quality README.md for Bluesky Meta-Discourse Labeler

## Goal
The goal of this task is to create a premium, high-quality, and developer-friendly `README.md` at the root of the project. The document will follow "Approach 1 (Quick-Start First)" with the following structural layout:
1. **Header & Conceptual Intro**: Define "Bluesky Meta-Discourse" and summarize the hybrid cloud-edge tech stack in 1-3 sentences of elegant prose, linking to [graze.social](https://graze.social) and [constellation.microcosm.blue](https://constellation.microcosm.blue).
2. **Quick Start**: Clear 3-step walkthrough to download the model, run `llama.cpp` using Docker Compose, copy the configuration, and run the daemon.
3. **Configuration Reference**: Comprehensive Markdown table of all environment variables from `.env.example`.
4. **System Architecture & Data Flow**: A Mermaid diagram showcasing the firehose-to-Ozone pipeline, along with the component map.
5. **Development & Contribution**: Build/test commands and our Git-Flow branching guidelines (referencing `AGENTS.md`).

---

## Proposed Changes

### 1. Root Directory [NEW] [README.md](file:///home/nmanos/Documents/Code/discourse-labeler/README.md)
We will create a new `README.md` at the repository root.

#### Content Design

*   **Title & Introduction**: A clear, professional title and description. The introduction will contain:
    > "The system leverages a hybrid cloud-and-edge architecture, using Go to orchestrate firehose ingestion from [Graze Contrails](https://graze.social) and post context hydration via [Microcosm Slingshot](https://constellation.microcosm.blue). Hydrated posts are then classified locally using Gemma 4 via `llama.cpp`, with positive matches cryptographically signed using the ATProto Indigo library and emitted to an Ozone moderation server."

*   **Quick Start**:
    1.  **Run `llama.cpp` with Gemma 4**: Run the Docker Compose server defined in the repo:
        ```bash
        docker compose up -d
        ```
    2.  **Set up Environment Variables**:
        ```bash
        cp .env.example .env
        ```
    3.  **Run the Daemon**:
        ```bash
        make run
        ```

*   **Configuration Table**: Fully document all variables (e.g. `PORT`, `LOG_LEVEL`, `CURSOR_FILE_PATH`, `HYDRATION_WORKERS`, `CLASSIFICATION_WORKERS`, `GRAZE_FEED_URI`, `CONTRAILS_WS_URL`, `SLINGSHOT_URL`, `LLM_ENDPOINT`, `OZONE_ENDPOINT`, `DRY_RUN`).

*   **Mermaid Architecture Diagram**: Include a clear Mermaid graph depicting the flow:
    `Graze Contrails (WS) -> Ingester -> Pipeline Coordinator (LRU Cache Check) -> Slingshot Hydrator (REST) -> llama.cpp (HTTP) -> Ozone Emitter (REST/Indigo) -> Ozone Instance`.

*   **Development Rules**:
    *   List key `make` targets: `make build`, `make test`, `make run`, `make verify-harness`.
    *   Detail the git-flow rules: Branch off `develop`, push to remote, and open PRs back to `develop`.

---

## Verification Plan

### Manual Verification
1.  **Markdown Integrity**: Review the generated `README.md` file locally to ensure all links, markdown tables, code blocks, and formatting are renderable and correct.
2.  **Mermaid Rendering**: Ensure the Mermaid syntax is valid.
3.  **Agent Harness Check**: Run `make verify-harness` to verify the project's agent readiness remains fully intact and compliant.
