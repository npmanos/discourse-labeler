# Bluesky Meta-Discourse Labeler: Implementation Specification

## Project Overview
This document outlines the architecture and implementation details for a Bluesky labeler designed to identify and tag "Bluesky Meta-Discourse" (complaints or commentary about the platform's culture, vibes, and user behavior). 

The system leverages a hybrid cloud-and-edge architecture to minimize local compute constraints on the host machine. It offloads firehose ingestion to Graze Contrails and post hydration to the Microcosm Slingshot edge cache, while performing local LLM classification using `gemma-4-e2b` via `llama.cpp`.

## System Architecture

1.  **Tier 1 Ingestion (Graze Contrails):** A Go application listens to a Graze Contrails WebSocket. Contrails handles the heavy lifting of filtering the ATProto firehose, pushing only JSON payloads that contain specific keywords (e.g., `bluesky`, `algorithm`, `atproto`, `timeline`).
2.  **Context Hydration (Microcosm Slingshot):** If an incoming Contrails event is a reply or a quote post, the Go app pauses classification and makes a synchronous HTTP GET request to `https://slingshot.microcosm.blue` to fetch the parent/quoted text.
3.  **Local Inference (`llama.cpp`):** The fully hydrated context is passed via local HTTP to a Dockerized `llama.cpp` server running `gemma-4-e2b`. The request uses the OpenAI Go SDK and enforces a strict GBNF JSON Schema.
4.  **Label Signing & Broadcasting:** If the classification is positive, the Go app uses Bluesky's `indigo` library to cryptographically sign a `com.atproto.label.defs#label` object and expose it.

## Infrastructure Deployment

The classification engine runs as a containerized service. 

### `docker-compose.yml` for llama.cpp

```yaml
services:
  llama-server:
    image: ghcr.io/ggml-org/llama.cpp:server
    container_name: discourse_llm
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./models:/models
    command: >
      -m /models/gemma-4-e2b.gguf
      --host 0.0.0.0
      --port 8080
      -c 8192
      --hf-repo "google/gemma-4-e2b-gguf"
      --hf-file "gemma-4-e2b.Q4_K_M.gguf"

```

*(Note: Adjust the quantization level and context window `-c` based on the host system's available RAM and CPU threads.)*

## Go Application Specification

### 1. Ingestion Worker

Use `gorilla/websocket` to connect to the Graze Contrails stream.
The worker should be completely stateless. It parses the incoming JSON, extracting the text, `reply.parent.uri`, and `embed.record.uri`.

### 2. Slingshot Hydration Logic

When hydrating, parse the AT-URI (`at://did:plc:.../app.bsky.feed.post/...`) into its `repo`, `collection`, and `rkey` components.
Make a standard GET request:
`https://slingshot.microcosm.blue/xrpc/com.atproto.repo.getRecord?repo={repo}&collection={collection}&rkey={rkey}`
Extract the text from the resulting JSON and construct the final prompt string:
`Context (Parent Post): [text]\n\nTarget Post: [text]`

### 3. Inference Logic

Use `github.com/openai/openai-go`.
Point the base URL to `http://localhost:8080/v1/`.

**System Prompt:**

```text
You are a classification engine powering a network labeler. Your task is to analyze a social media post and determine if it qualifies as "Bluesky Meta-Discourse."

# DEFINITION: META-DISCOURSE (TRUE)
Meta-discourse consists of posts evaluating, criticizing, or theorizing about the cultural and social experience of the platform itself. This includes:
- Debating the "vibes," echo chambers, or user base behaviors of Bluesky.
- Comparing the social experience, engagement dynamics, or toxicity of Bluesky versus X (Twitter) or other platforms.
- Complaining about the types of conversations people have on the platform (e.g., "dead-end conversations", "too much drama", "people talking to themselves").
- Meta-commentary, subtweets, or reactions to other users' posts regarding Bluesky's culture.

# DEFINITION: NOT META-DISCOURSE (FALSE)
The following are strictly NOT meta-discourse:
- Technical discussions about building on the AT Protocol (atproto), creating custom feeds, using APIs, connecting to Jetstream, or hosting infrastructure.
- Announcements or discussions of new Bluesky application features (e.g., "DMs are now live", "how to use the new video player").
- General political, social, or pop culture arguments, even if they are toxic or reference platform moderation, as long as they are not explicitly analyzing the platform's culture as a whole.
- Ordinary posts using platform-specific terminology (like "skeet" or "repost") in passing.

# INSTRUCTIONS
Analyze the provided user post. Output a valid JSON object containing exactly one boolean key: `is_meta_discourse`.

```

**Few-Shot Examples (Insert into `Messages` array before the target post):**

* **User:** `i think, end of the day, the real problem with Bluesky is that most of its users are here *because* they want to be in a bubble. it's why despite the activity on here, the site still gives people bad vibes. X, despite it all, is still a more fun place.` -> **Assistant:** `{"is_meta_discourse": true}`
* **User:** `i'm not on bluesky because i want to live in a bubble. i'm on bluesky because i love reading long manifestos about what's wrong with bluesky by people who don't spend enough time here to know someone does this every 10 days.` -> **Assistant:** `{"is_meta_discourse": true}`
* **User:** `Finally got my labeler up and running! I'm streaming Jetstream into a Go backend and using a local Ollama container to classify text. The atproto documentation for cryptographically signing the labels was a bit dense but I figured it out.` -> **Assistant:** `{"is_meta_discourse": false}`
* **User:** `The Bluesky team just pushed an update for the new video player. You can now scrub through clips without the audio dropping out. Huge improvement over the beta version from last week.` -> **Assistant:** `{"is_meta_discourse": false}`
* **User:** `Every time I post about this election, my replies fill up with the worst takes imaginable. I can't believe people are actually defending this policy.` -> **Assistant:** `{"is_meta_discourse": false}`

**Schema & Logprobs Configuration:**
Configure the request with `Temperature: 0.0` and enable `Logprobs: true` with `TopLogprobs: 2`.
Enforce the JSON schema using `openai.ResponseFormatJSONSchemaParam`:

```json
{
  "type": "object",
  "properties": {
    "is_meta_discourse": {
      "type": "boolean",
      "description": "True if the post is a meta-complaint."
    }
  },
  "required": ["is_meta_discourse"],
  "additionalProperties": false
}

```

### 4. Evaluation and Labeling

When the completion returns, extract the boolean value.
If the boolean is `true`, evaluate the confidence using the returned `logprobs`:

1. Iterate through `response.choices[0].logprobs.content[0]`.
2. Find the logprob for the token corresponding to `true`.
3. Calculate statistical probability: `probability = math.Exp(logprob)`.
4. Apply logic:
* If `probability >= 0.85`, assign the `meta_discourse` label.
* If `0.60 <= probability < 0.85`, assign the `possible_meta_discourse` label.
* If `< 0.60`, discard.


5. Use `github.com/bluesky-social/indigo/api/atproto` to cryptographically sign the `com.atproto.label.defs#label` object and broadcast to subscribers.

