package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCursorTrackerReadNonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.json")

	ct := NewCursorTracker(path)
	val, err := ct.Read()
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got: %v", err)
	}
	if val != 0 {
		t.Fatalf("expected 0 for non-existent file, got: %d", val)
	}
}

func TestCursorTrackerWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.json")

	ct := NewCursorTracker(path)

	// Write a cursor value.
	want := int64(1716595200000000) // a realistic time_us value
	if err := ct.Write(want); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read it back.
	got, err := ct.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != want {
		t.Fatalf("roundtrip mismatch: got %d, want %d", got, want)
	}

	// Verify file content is valid JSON with expected format.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cursor file: %v", err)
	}
	expected := `{"cursor":1716595200000000}`
	if strings.TrimSpace(string(data)) != expected {
		t.Fatalf("unexpected file content: %q, want %q", string(data), expected)
	}
}

func TestCursorTrackerOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.json")

	ct := NewCursorTracker(path)

	// Write initial value.
	if err := ct.Write(100); err != nil {
		t.Fatalf("first Write failed: %v", err)
	}

	// Overwrite with new value.
	if err := ct.Write(200); err != nil {
		t.Fatalf("second Write failed: %v", err)
	}

	got, err := ct.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != 200 {
		t.Fatalf("expected 200 after overwrite, got %d", got)
	}
}

func TestCursorTrackerAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.json")

	ct := NewCursorTracker(path)

	// Write a value so the file exists.
	if err := ct.Write(42); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// After a successful write the target file must exist and no temp
	// files should remain in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	foundTarget := false
	for _, e := range entries {
		if e.Name() == filepath.Base(path) {
			foundTarget = true
			continue
		}
		// Any leftover .tmp file indicates a non-atomic write path.
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file %q was not cleaned up — write is not atomic", e.Name())
		}
	}
	if !foundTarget {
		t.Fatal("cursor file does not exist after Write")
	}

	// Verify the value survived (i.e., we didn't end up with a zero-length
	// file from a truncation-based write).
	got, err := ct.Read()
	if err != nil {
		t.Fatalf("Read after atomic write failed: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestCursorTrackerReadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.json")

	// Write garbage to the file.
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	ct := NewCursorTracker(path)
	_, err := ct.Read()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
