# Bluesky Meta-Discourse Labeler 🏷️🤖

The **Bluesky Meta-Discourse Labeler** is a high-performance Go-based background daemon designed to classify real-time posts from the Bluesky feed for "meta-discourse" and cryptographically sign and emit labels to an Ozone moderation instance.

The system leverages a hybrid cloud-and-edge architecture, using Go to orchestrate firehose ingestion from [Graze Contrails](https://graze.social) and post context hydration via [Microcosm Slingshot](https://constellation.microcosm.blue). Hydrated posts are then classified locally using Gemma 4 via `llama.cpp`, with positive matches cryptographically signed using the ATProto Indigo library and emitted to an Ozone moderation server.

## Conceptual Definition: Meta-Discourse

### What is Meta-Discourse? (TRUE)
Posts evaluating, criticizing, or theorizing about the cultural and social experience of Bluesky itself. This includes:
- Debating the platform's "vibes," echo chambers, or user base behaviors.
- Comparing the social experience, engagement dynamics, or toxicity of Bluesky versus X (Twitter) or other platforms.
- Complaining about the types of conversations people have (e.g., "dead-end conversations", "too much drama", "people subtweeting").
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
