# --- General Targets ---
.PHONY: build test run clean

build:
	go build -o bin/labeler ./cmd/labeler

test:
	go test -v ./...

run:
	go run ./cmd/labeler

clean:
	rm -rf bin/

# --- Agent Harness Targets ---
.PHONY: verify-harness harness-status

verify-harness:
	@bash scripts/verify-harness.sh --format=text

harness-status:
	@bash scripts/verify-harness.sh --format=text --status
