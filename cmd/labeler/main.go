package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/npmanos/discourse-labeler/internal/config"
	types "github.com/npmanos/discourse-labeler/internal/pipeline" // Resolves to 'types' package
	"github.com/npmanos/discourse-labeler/internal/services"
)

func main() {
	log.Println("Starting Bluesky Meta-Discourse Labeler Daemon...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Fatal: failed to load configuration: %v", err)
	}

	if cfg.GrazeFeedURI == "" {
		log.Fatal("Fatal: GRAZE_FEED_URI environment variable is required")
	}

	// Signal handling context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Initialize Attached Resources/Adapters
	ingester := services.NewContrailsIngester(cfg.ContrailsWSURL, cfg.GrazeFeedURI)
	hydrator := services.NewSlingshotHydrator(cfg.SlingshotURL)
	classifier := services.NewLLMClassifier(cfg.LLMEndpoint, cfg.LLMModel, services.WithSystemPrompt(cfg.LLMSystemPrompt), services.WithAPIKey(cfg.LLMAPIKey))
	ozoneClient := services.NewOzoneClient(cfg.OzoneEndpoint, cfg.OzoneAdminToken, cfg.LabelerDID)
	cursor := types.NewCursorTracker(cfg.CursorFilePath)

	// 2. Build Pipeline Coordinator
	coordinator := types.NewCoordinator(
		ingester,
		hydrator,
		classifier,
		ozoneClient,
		ozoneClient,
		cursor,
		cfg.CursorRewindSeconds,
		cfg.HydrationWorkers,
		cfg.ClassificationWorkers,
		cfg.DryRun,
	)

	log.Println("Pipeline initialized. Spawning worker pools...")

	// 3. Start Ingestion and Worker Loops
	if err := coordinator.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Fatal: Coordinator execution error: %v", err)
	}

	log.Println("Gracefully shut down all worker pools. Storing final states.")
}
