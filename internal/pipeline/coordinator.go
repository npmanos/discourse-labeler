package types

import (
	"context"
	"log"
	"sync"
	"time"
)

// Ingester defines the interface for consuming events from a source.
type Ingester interface {
	Start(ctx context.Context, cursor int64, out chan<- *RawEvent) error
}

// Hydrator defines the interface for resolving context for raw events.
type Hydrator interface {
	Hydrate(ctx context.Context, ev *RawEvent) (*HydratedPost, error)
}

// Classifier defines the interface for determining if a post is meta discourse.
type Classifier interface {
	Classify(ctx context.Context, hp *HydratedPost) (*ClassificationResult, error)
}

// LabelEmitter defines the interface for query and emission of labels.
type LabelEmitter interface {
	EmitLabel(ctx context.Context, result *ClassificationResult) error
	EmitEscalation(ctx context.Context, result *ClassificationResult) error
	IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
}

// LRUCache implements a thread-safe Least Recently Used cache for string keys.
type LRUCache struct {
	mu    sync.Mutex
	items map[string]bool
	keys  []string
	size  int
}

// NewLRUCache initializes a new LRUCache with the specified maximum capacity.
func NewLRUCache(size int) *LRUCache {
	if size <= 0 {
		size = 10000
	}
	return &LRUCache{
		items: make(map[string]bool),
		keys:  make([]string, 0, size),
		size:  size,
	}
}

// Contains checks if key is in the cache without updating its recency or adding it.
func (c *LRUCache) Contains(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.items[key]
}

// ContainsOrAdd checks if key is already in the cache, moves it to the newest position,
// and if not present, adds it and potentially evicts the oldest item.
// Returns true if the key was already present, false otherwise.
func (c *LRUCache) ContainsOrAdd(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.items[key] {
		// Move to end (newest)
		for i, k := range c.keys {
			if k == key {
				c.keys = append(c.keys[:i], c.keys[i+1:]...)
				break
			}
		}
		c.keys = append(c.keys, key)
		return true
	}

	// Add new item, evicting oldest if capacity reached
	if len(c.keys) >= c.size {
		oldest := c.keys[0]
		c.keys = c.keys[1:]
		delete(c.items, oldest)
	}

	c.keys = append(c.keys, key)
	c.items[key] = true
	return false
}

// Coordinator orchestrates the ingestion, filtering, deduplication, hydration,
// classification, and label emission pipeline.
type Coordinator struct {
	Ingester    Ingester
	Hydrator    Hydrator
	Classifier  Classifier
	OzoneClient LabelEmitter
	OzoneQuery  interface {
		IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
	}
	Cursor                *CursorTracker
	CursorRewindSeconds   int
	HydrationWorkers      int
	ClassificationWorkers int
	DryRun                bool
	LRU                   *LRUCache

	// Concurrency and tracking fields
	wg             sync.WaitGroup
	processedCount int
	processedMu    sync.Mutex
	cursorMu       sync.Mutex
	maxEventTimeUS int64
}

// NewCoordinator instantiates a new Coordinator with the required dependencies and parameters.
func NewCoordinator(
	ingester Ingester,
	hydrator Hydrator,
	classifier Classifier,
	emitter LabelEmitter,
	oq interface {
		IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
	},
	cursor *CursorTracker,
	rewindSecs, hydWorkers, classWorkers int,
	dryRun bool,
) *Coordinator {
	return &Coordinator{
		Ingester:              ingester,
		Hydrator:              hydrator,
		Classifier:            classifier,
		OzoneClient:           emitter,
		OzoneQuery:            oq,
		Cursor:                cursor,
		CursorRewindSeconds:   rewindSecs,
		HydrationWorkers:      hydWorkers,
		ClassificationWorkers: classWorkers,
		DryRun:                dryRun,
		LRU:                   NewLRUCache(10000),
	}
}

// Run executes the pipeline under the given context. It blocks until the context is cancelled
// and all workers exit gracefully.
func (co *Coordinator) Run(ctx context.Context) error {
	// Read current cursor
	startCursor, err := co.Cursor.Read()
	if err != nil {
		log.Printf("Warning: Failed to read cursor state: %v. Defaulting to 0.", err)
		startCursor = 0
	}

	// Apply cursor rewind if rewind seconds is positive
	if co.CursorRewindSeconds > 0 {
		rewind := int64(co.CursorRewindSeconds) * 1_000_000
		if startCursor > rewind {
			startCursor -= rewind
		} else {
			startCursor = 0
		}
	}

	// Buffered channels for stage communication
	rawChan := make(chan *RawEvent, 1000)
	hydratedChan := make(chan *HydratedPost, 100)

	// Launch ingestion goroutine
	go func() {
		if err := co.Ingester.Start(ctx, startCursor, rawChan); err != nil {
			log.Printf("Ingester stopped with error: %v", err)
		}
		close(rawChan)
	}()

	// Launch HydrationWorkers
	var hydWG sync.WaitGroup
	for i := 0; i < co.HydrationWorkers; i++ {
		hydWG.Add(1)
		go func() {
			defer hydWG.Done()
			co.runHydrationWorker(ctx, rawChan, hydratedChan)
		}()
	}

	// Launch ClassificationWorkers
	var classWG sync.WaitGroup
	for i := 0; i < co.ClassificationWorkers; i++ {
		classWG.Add(1)
		go func() {
			defer classWG.Done()
			co.runClassificationWorker(ctx, hydratedChan)
		}()
	}

	// Orchestrator goroutine to close hydratedChan when hydration is complete
	go func() {
		hydWG.Wait()
		co.wg.Wait()
		close(hydratedChan)
	}()

	// Block until all classification workers have finished (which happens after hydratedChan is closed)
	classWG.Wait()

	return nil
}

// runHydrationWorker handles incoming RawEvent records, filtering, deduplication, and triggering hydration.
func (co *Coordinator) runHydrationWorker(ctx context.Context, in <-chan *RawEvent, out chan<- *HydratedPost) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-in:
			if !ok {
				return
			}

			// Validate and filter
			if ev.Type != "commit" || ev.Commit == nil || ev.Commit.Collection != "app.bsky.feed.post" {
				continue
			}

			// LRU Cache check for deduplication
			postURI := "at://" + ev.Did + "/" + ev.Commit.Collection + "/" + ev.Commit.RKey
			if co.LRU.ContainsOrAdd(postURI) {
				continue
			}

			// Launch hydration asynchronously with retry and WaitGroup tracking
			co.wg.Add(1)
			go co.hydrateWithRetry(ctx, ev, out, 0)
		}
	}
}

// hydrateWithRetry implements a resilient hydration wrapper with exponential backoff.
func (co *Coordinator) hydrateWithRetry(ctx context.Context, ev *RawEvent, out chan<- *HydratedPost, attempt int) {
	if ctx.Err() != nil {
		co.wg.Done()
		return
	}

	if attempt >= 3 {
		log.Printf("Error: Hydration failed after %d attempts for post from DID %s. Dropping event.", attempt, ev.Did)
		co.wg.Done()
		return
	}

	hp, err := co.Hydrator.Hydrate(ctx, ev)
	if err != nil {
		backoff := time.Duration(1<<(attempt+1)) * 250 * time.Millisecond
		select {
		case <-ctx.Done():
			co.wg.Done()
			return
		case <-time.After(backoff):
			co.hydrateWithRetry(ctx, ev, out, attempt+1)
		}
		return
	}

	// Successfully hydrated, push to the next stage channel
	select {
	case out <- hp:
	case <-ctx.Done():
	}
	co.wg.Done()
}

// runClassificationWorker drains hydrated posts, evaluates them via Classifier, and emits labels via Ozone.
func (co *Coordinator) runClassificationWorker(ctx context.Context, in <-chan *HydratedPost) {
	for {
		select {
		case <-ctx.Done():
			return
		case hp, ok := <-in:
			if !ok {
				return
			}
			co.processClassification(ctx, hp)
		}
	}
}

// processClassification runs evaluation on a single hydrated post and takes actions based on results.
func (co *Coordinator) processClassification(ctx context.Context, hp *HydratedPost) {
	res, err := co.Classifier.Classify(ctx, hp)
	if err != nil {
		log.Printf("Classification failed: %v", err)
		co.incrementProcessedAndWriteCursor(hp, false)
		return
	}

	if res != nil {
		log.Printf("Classification result: URI=%s, IsMetaDiscourse=%t, Probability=%.4f", hp.TargetURI, res.IsMetaDiscourse, res.Probability)
	}

	if res != nil && res.IsMetaDiscourse {
		if !co.DryRun {
			already, err := co.OzoneQuery.IsAlreadyLabeled(ctx, hp.TargetURI)
			if err == nil && !already {
				if err := co.OzoneClient.EmitLabel(ctx, res); err != nil {
					log.Printf("Failed to emit label to Ozone: %v", err)
				}
			} else if err != nil {
				log.Printf("Ozone query IsAlreadyLabeled failed: %v", err)
			}
		} else {
			labelVal := "possible-meta-discourse"
			if res.Probability >= 0.85 {
				labelVal = "meta-discourse"
			}
			log.Printf("[DRY RUN] Would have emitted label %q to Ozone for URI=%s (probability: %.4f)", labelVal, hp.TargetURI, res.Probability)
		}
		co.incrementProcessedAndWriteCursor(hp, true)
	} else {
		co.incrementProcessedAndWriteCursor(hp, false)
	}
}

// incrementProcessedAndWriteCursor manages concurrent processed count tracking and atomic cursor updates.
func (co *Coordinator) incrementProcessedAndWriteCursor(hp *HydratedPost, isMeta bool) {
	co.processedMu.Lock()
	co.processedCount++
	count := co.processedCount
	co.processedMu.Unlock()

	shouldWrite := isMeta || (count%100 == 0)
	if shouldWrite {
		co.cursorMu.Lock()
		if hp.EventTimeUS > co.maxEventTimeUS {
			if err := co.Cursor.Write(hp.EventTimeUS); err == nil {
				co.maxEventTimeUS = hp.EventTimeUS
			} else {
				log.Printf("Failed to write cursor state: %v", err)
			}
		}
		co.cursorMu.Unlock()
	}
}
