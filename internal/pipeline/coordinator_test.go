package types

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

type mockIngester struct {
	startFunc func(ctx context.Context, cursor int64, out chan<- *RawEvent) error
}

func (m *mockIngester) Start(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
	if m.startFunc != nil {
		return m.startFunc(ctx, cursor, out)
	}
	return nil
}

type mockHydrator struct {
	hydrateFunc func(ctx context.Context, ev *RawEvent) (*HydratedPost, error)
}

func (m *mockHydrator) Hydrate(ctx context.Context, ev *RawEvent) (*HydratedPost, error) {
	if m.hydrateFunc != nil {
		return m.hydrateFunc(ctx, ev)
	}
	return &HydratedPost{
		TargetDID:   ev.Did,
		TargetRKey:  ev.Commit.RKey,
		TargetURI:   "at://" + ev.Did + "/" + ev.Commit.Collection + "/" + ev.Commit.RKey,
		TargetCID:   ev.Commit.CID,
		EventTimeUS: ev.TimeUS,
	}, nil
}

type mockClassifier struct {
	classifyFunc func(ctx context.Context, hp *HydratedPost) (*ClassificationResult, error)
}

func (m *mockClassifier) Classify(ctx context.Context, hp *HydratedPost) (*ClassificationResult, error) {
	if m.classifyFunc != nil {
		return m.classifyFunc(ctx, hp)
	}
	return &ClassificationResult{
		Post:        hp,
		Probability: 0.1,
		TargetPost: PostClassification{
			Classification: LabelNotMeta,
			Reasoning:      "Default mock non-meta response",
		},
	}, nil
}

type mockOzone struct {
	emitLabelFunc        func(ctx context.Context, result *ClassificationResult) error
	emitEscalationFunc   func(ctx context.Context, result *ClassificationResult) error
	isAlreadyLabeledFunc func(ctx context.Context, uri string) (bool, error)
}

func (m *mockOzone) EmitLabel(ctx context.Context, result *ClassificationResult) error {
	if m.emitLabelFunc != nil {
		return m.emitLabelFunc(ctx, result)
	}
	return nil
}

func (m *mockOzone) EmitEscalation(ctx context.Context, result *ClassificationResult) error {
	if m.emitEscalationFunc != nil {
		return m.emitEscalationFunc(ctx, result)
	}
	return nil
}

func (m *mockOzone) IsAlreadyLabeled(ctx context.Context, uri string) (bool, error) {
	if m.isAlreadyLabeledFunc != nil {
		return m.isAlreadyLabeledFunc(ctx, uri)
	}
	return false, nil
}

func setupCursorTracker(t *testing.T, initial int64) (*CursorTracker, func()) {
	tmpDir, err := os.MkdirTemp("", "cursor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	path := filepath.Join(tmpDir, "cursor.json")
	ct := NewCursorTracker(path)
	if initial > 0 {
		if err := ct.Write(initial); err != nil {
			t.Fatalf("failed to write initial cursor: %v", err)
		}
	}
	return ct, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestLRUCache(t *testing.T) {
	lru := NewLRUCache(3)
	if lru.Contains("a") {
		t.Error("Expected a not to be in cache")
	}
	if lru.ContainsOrAdd("a") {
		t.Error("Expected a not to be in cache initially")
	}
	if !lru.Contains("a") {
		t.Error("Expected a to be in cache now")
	}
	if lru.ContainsOrAdd("b") {
		t.Error("Expected b not to be in cache")
	}
	if lru.ContainsOrAdd("c") {
		t.Error("Expected c not to be in cache")
	}

	// LRU is now full: keys = ["a", "b", "c"]
	// Access "a" to make it most recent
	if !lru.ContainsOrAdd("a") {
		t.Error("Expected a to be in cache")
	}
	// keys = ["b", "c", "a"]

	// Adding "d" should evict "b" (oldest)
	if lru.ContainsOrAdd("d") {
		t.Error("Expected d not to be in cache")
	}

	// Verify "b" was evicted
	if lru.Contains("b") {
		t.Error("Expected b to be evicted")
	}
	// Verify "c" and "a" are still there
	if !lru.Contains("c") {
		t.Error("Expected c to still be present")
	}
	if !lru.Contains("a") {
		t.Error("Expected a to still be present")
	}
}

func TestCoordinator_IngestionStartAndRewind(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 5_000_000)
	defer cleanup()

	startCalled := make(chan int64, 1)
	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			startCalled <- cursor
			return nil
		},
	}
	hydrator := &mockHydrator{}
	classifier := &mockClassifier{}
	ozone := &mockOzone{}

	// Rewind by 2 seconds = 2,000,000 microseconds
	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 2, 1, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = co.Run(ctx)
	}()

	select {
	case cursor := <-startCalled:
		expected := int64(3_000_000) // 5_000_000 - 2_000_000
		if cursor != expected {
			t.Errorf("Expected start cursor to be %d, got %d", expected, cursor)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for Ingester.Start")
	}
}

func TestCoordinator_FilteringAndDeduplication(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	events := []*RawEvent{
		{
			Did:  "did:plc:1",
			Type: "identity", // Should be filtered out
		},
		{
			Did:  "did:plc:2",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.like", // Should be filtered out
				RKey:       "r1",
			},
		},
		{
			Did:  "did:plc:3",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.post", // OK
				RKey:       "r2",
			},
			TimeUS: 1000,
		},
		{
			Did:  "did:plc:3",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.post", // Duplicate, should be filtered out by LRU
				RKey:       "r2",
			},
			TimeUS: 2000,
		},
	}

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			for _, ev := range events {
				out <- ev
			}
			return nil
		},
	}

	var hydratedMu sync.Mutex
	var hydrated []*RawEvent
	hydrator := &mockHydrator{
		hydrateFunc: func(ctx context.Context, ev *RawEvent) (*HydratedPost, error) {
			hydratedMu.Lock()
			hydrated = append(hydrated, ev)
			hydratedMu.Unlock()
			return &HydratedPost{
				TargetDID:   ev.Did,
				TargetRKey:  ev.Commit.RKey,
				TargetCID:   ev.Commit.CID,
				EventTimeUS: ev.TimeUS,
			}, nil
		},
	}

	classifier := &mockClassifier{}
	ozone := &mockOzone{}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 1, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := co.Run(ctx)
	if err != nil {
		t.Errorf("Coordinator.Run returned error: %v", err)
	}

	hydratedMu.Lock()
	count := len(hydrated)
	hydratedMu.Unlock()

	if count != 1 {
		t.Errorf("Expected exactly 1 hydrated post, got %d", count)
	} else {
		if hydrated[0].Did != "did:plc:3" {
			t.Errorf("Expected hydrated post from did:plc:3, got %s", hydrated[0].Did)
		}
	}
}

func TestCoordinator_HydrationRetry(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	ev := &RawEvent{
		Did:  "did:plc:1",
		Type: "commit",
		Commit: &JetstreamCommit{
			Collection: "app.bsky.feed.post",
			RKey:       "r1",
		},
		TimeUS: 1000,
	}

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			out <- ev
			return nil
		},
	}

	var calls int
	var callsMu sync.Mutex
	hydrator := &mockHydrator{
		hydrateFunc: func(ctx context.Context, ev *RawEvent) (*HydratedPost, error) {
			callsMu.Lock()
			calls++
			currentCalls := calls
			callsMu.Unlock()

			if currentCalls < 2 {
				// Fail the first call
				return nil, errors.New("temporary hydration error")
			}
			// Succeed on the second call
			return &HydratedPost{
				TargetDID:   ev.Did,
				TargetRKey:  ev.Commit.RKey,
				TargetCID:   ev.Commit.CID,
				EventTimeUS: ev.TimeUS,
			}, nil
		},
	}

	classifier := &mockClassifier{}
	ozone := &mockOzone{}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 1, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := co.Run(ctx)
	if err != nil {
		t.Errorf("Coordinator.Run returned error: %v", err)
	}

	callsMu.Lock()
	finalCalls := calls
	callsMu.Unlock()

	if finalCalls != 2 {
		t.Errorf("Expected exactly 2 hydration attempts, got %d", finalCalls)
	}
}

func TestCoordinator_HydrationDropAfterMaxRetries(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	ev := &RawEvent{
		Did:  "did:plc:1",
		Type: "commit",
		Commit: &JetstreamCommit{
			Collection: "app.bsky.feed.post",
			RKey:       "r1",
		},
		TimeUS: 1000,
	}

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			out <- ev
			return nil
		},
	}

	var calls int
	var callsMu sync.Mutex
	hydrator := &mockHydrator{
		hydrateFunc: func(ctx context.Context, ev *RawEvent) (*HydratedPost, error) {
			callsMu.Lock()
			calls++
			callsMu.Unlock()
			return nil, errors.New("permanent hydration error")
		},
	}

	classifier := &mockClassifier{}
	ozone := &mockOzone{}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 1, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	err := co.Run(ctx)
	if err != nil {
		t.Errorf("Coordinator.Run returned error: %v", err)
	}

	callsMu.Lock()
	finalCalls := calls
	callsMu.Unlock()

	if finalCalls != 3 {
		t.Errorf("Expected exactly 3 hydration attempts before drop, got %d", finalCalls)
	}
}

func TestCoordinator_ClassificationAndOzone(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	ev := &RawEvent{
		Did:  "did:plc:1",
		Type: "commit",
		Commit: &JetstreamCommit{
			Collection: "app.bsky.feed.post",
			RKey:       "r1",
		},
		TimeUS: 1000,
	}

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			out <- ev
			return nil
		},
	}

	hydrator := &mockHydrator{}

	classifier := &mockClassifier{
		classifyFunc: func(ctx context.Context, hp *HydratedPost) (*ClassificationResult, error) {
			return &ClassificationResult{
				Post:        hp,
				Probability: 0.95,
				TargetPost: PostClassification{
					Classification: LabelDefiniteMeta,
					Reasoning:      "Definitely meta",
				},
			}, nil
		},
	}

	var ozoneQueryCalled, ozoneEmitCalled int
	var ozoneMu sync.Mutex
	ozone := &mockOzone{
		isAlreadyLabeledFunc: func(ctx context.Context, uri string) (bool, error) {
			ozoneMu.Lock()
			ozoneQueryCalled++
			ozoneMu.Unlock()
			return false, nil
		},
		emitLabelFunc: func(ctx context.Context, result *ClassificationResult) error {
			ozoneMu.Lock()
			ozoneEmitCalled++
			ozoneMu.Unlock()
			return nil
		},
	}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 1, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = co.Run(ctx)

	ozoneMu.Lock()
	if ozoneQueryCalled != 1 {
		t.Errorf("Expected Ozone.IsAlreadyLabeled to be called 1 time, got %d", ozoneQueryCalled)
	}
	if ozoneEmitCalled != 1 {
		t.Errorf("Expected Ozone.EmitLabel to be called 1 time, got %d", ozoneEmitCalled)
	}
	ozoneMu.Unlock()

	// Dry-run = true
	ct2, cleanup2 := setupCursorTracker(t, 0)
	defer cleanup2()

	var ozoneQueryCalled2, ozoneEmitCalled2 int
	ozone2 := &mockOzone{
		isAlreadyLabeledFunc: func(ctx context.Context, uri string) (bool, error) {
			ozoneQueryCalled2++
			return false, nil
		},
		emitLabelFunc: func(ctx context.Context, result *ClassificationResult) error {
			ozoneEmitCalled2++
			return nil
		},
	}

	co2 := NewCoordinator(ingester, hydrator, classifier, ozone2, ozone2, ct2, 0, 1, 1, true)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	_ = co2.Run(ctx2)

	if ozoneQueryCalled2 != 0 {
		t.Errorf("Expected Ozone.IsAlreadyLabeled NOT to be called under dry run, got %d", ozoneQueryCalled2)
	}
	if ozoneEmitCalled2 != 0 {
		t.Errorf("Expected Ozone.EmitLabel NOT to be called under dry run, got %d", ozoneEmitCalled2)
	}
}

func TestCoordinator_AtomicCursorUpdates(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			for i := 1; i <= 105; i++ {
				out <- &RawEvent{
					Did:  "did:plc:1",
					Type: "commit",
					Commit: &JetstreamCommit{
						Collection: "app.bsky.feed.post",
						RKey:       "r" + strconv.Itoa(i),
					},
					TimeUS: int64(i * 1000),
				}
			}
			return nil
		},
	}

	hydrator := &mockHydrator{
		hydrateFunc: func(ctx context.Context, ev *RawEvent) (*HydratedPost, error) {
			return &HydratedPost{
				TargetDID:   ev.Did,
				TargetRKey:  ev.Commit.RKey,
				TargetCID:   ev.Commit.CID,
				EventTimeUS: ev.TimeUS,
			}, nil
		},
	}

	classifier := &mockClassifier{
		classifyFunc: func(ctx context.Context, hp *HydratedPost) (*ClassificationResult, error) {
			return &ClassificationResult{
				Post: hp,
				TargetPost: PostClassification{
					Classification: LabelNotMeta,
					Reasoning:      "Not meta",
				},
			}, nil
		},
	}

	ozone := &mockOzone{}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 2, 2, false)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = co.Run(ctx)

	val, err := ct.Read()
	if err != nil {
		t.Fatalf("failed to read cursor: %v", err)
	}

	if val == 0 {
		t.Error("Expected cursor to be updated and non-zero")
	}
}

func TestCoordinator_GracefulShutdown(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	hydrator := &mockHydrator{}
	classifier := &mockClassifier{}
	ozone := &mockOzone{}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 2, 2, false)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := co.Run(ctx)
	duration := time.Since(start)

	if duration > 1*time.Second {
		t.Errorf("Coordinator.Run did not exit quickly upon context cancellation, took %v", duration)
	}
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("unexpected error on cancellation: %v", err)
	}
}

func TestCoordinator_CategoricalRouting(t *testing.T) {
	ct, cleanup := setupCursorTracker(t, 0)
	defer cleanup()

	// 4 events to test 4 categories
	events := []*RawEvent{
		{
			Did:  "did:plc:1",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.post",
				RKey:       "r1",
			},
			TimeUS: 1000,
		},
		{
			Did:  "did:plc:2",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.post",
				RKey:       "r2",
			},
			TimeUS: 2000,
		},
		{
			Did:  "did:plc:3",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.post",
				RKey:       "r3",
			},
			TimeUS: 3000,
		},
		{
			Did:  "did:plc:4",
			Type: "commit",
			Commit: &JetstreamCommit{
				Collection: "app.bsky.feed.post",
				RKey:       "r4",
			},
			TimeUS: 4000,
		},
	}

	ingester := &mockIngester{
		startFunc: func(ctx context.Context, cursor int64, out chan<- *RawEvent) error {
			for _, ev := range events {
				out <- ev
			}
			return nil
		},
	}

	hydrator := &mockHydrator{
		hydrateFunc: func(ctx context.Context, ev *RawEvent) (*HydratedPost, error) {
			return &HydratedPost{
				TargetDID:   ev.Did,
				TargetRKey:  ev.Commit.RKey,
				TargetURI:   "at://" + ev.Did + "/" + ev.Commit.Collection + "/" + ev.Commit.RKey,
				TargetCID:   ev.Commit.CID,
				TargetText:  "Post " + ev.Commit.RKey,
				EventTimeUS: ev.TimeUS,
			}, nil
		},
	}

	classifier := &mockClassifier{
		classifyFunc: func(ctx context.Context, hp *HydratedPost) (*ClassificationResult, error) {
			var label ClassificationLabel
			var prob float64
			switch hp.TargetRKey {
			case "r1":
				label = LabelDefiniteMeta
				prob = 0.95
			case "r2":
				label = LabelLikelyMeta
				prob = 0.80
			case "r3":
				label = LabelUnsure
				prob = 0.50
			case "r4":
				label = LabelNotMeta
				prob = 0.05
			}
			return &ClassificationResult{
				Post:        hp,
				Probability: prob,
				TargetPost: PostClassification{
					Classification: label,
					Reasoning:      "Reason for " + string(label),
				},
			}, nil
		},
	}

	var emitLabelCalls []*ClassificationResult
	var emitEscalationCalls []*ClassificationResult
	var isAlreadyLabeledCalls []string
	var ozoneMu sync.Mutex

	ozone := &mockOzone{
		isAlreadyLabeledFunc: func(ctx context.Context, uri string) (bool, error) {
			ozoneMu.Lock()
			isAlreadyLabeledCalls = append(isAlreadyLabeledCalls, uri)
			ozoneMu.Unlock()
			return false, nil
		},
		emitLabelFunc: func(ctx context.Context, result *ClassificationResult) error {
			ozoneMu.Lock()
			emitLabelCalls = append(emitLabelCalls, result)
			ozoneMu.Unlock()
			return nil
		},
		emitEscalationFunc: func(ctx context.Context, result *ClassificationResult) error {
			ozoneMu.Lock()
			emitEscalationCalls = append(emitEscalationCalls, result)
			ozoneMu.Unlock()
			return nil
		},
	}

	co := NewCoordinator(ingester, hydrator, classifier, ozone, ozone, ct, 0, 1, 1, false)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = co.Run(ctx)

	ozoneMu.Lock()
	defer ozoneMu.Unlock()

	// 1. Definite and Likely Meta should trigger IsAlreadyLabeled query and EmitLabel
	// We expect 2 IsAlreadyLabeled calls (for r1 and r2)
	if len(isAlreadyLabeledCalls) != 2 {
		t.Errorf("Expected 2 IsAlreadyLabeled calls, got %d", len(isAlreadyLabeledCalls))
	}

	// We expect 2 EmitLabel calls (for r1 and r2)
	if len(emitLabelCalls) != 2 {
		t.Errorf("Expected 2 EmitLabel calls, got %d", len(emitLabelCalls))
	} else {
		hasDefinite := false
		hasLikely := false
		for _, call := range emitLabelCalls {
			if call.TargetPost.Classification == LabelDefiniteMeta {
				hasDefinite = true
			}
			if call.TargetPost.Classification == LabelLikelyMeta {
				hasLikely = true
			}
		}
		if !hasDefinite {
			t.Error("Expected an EmitLabel call with definite_meta")
		}
		if !hasLikely {
			t.Error("Expected an EmitLabel call with likely_meta")
		}
	}

	// 2. Unsure should trigger EmitEscalation
	if len(emitEscalationCalls) != 1 {
		t.Errorf("Expected 1 EmitEscalation call, got %d", len(emitEscalationCalls))
	} else {
		if emitEscalationCalls[0].TargetPost.Classification != LabelUnsure {
			t.Errorf("Expected EmitEscalation call to be unsure, got %s", emitEscalationCalls[0].TargetPost.Classification)
		}
	}
}
