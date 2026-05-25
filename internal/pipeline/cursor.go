package types

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// cursorState is the on-disk JSON representation of the persisted cursor.
type cursorState struct {
	Cursor int64 `json:"cursor"`
}

// CursorTracker provides atomic read/write persistence for a Jetstream
// time_us cursor value. The cursor is stored as a JSON file, and writes
// use a write-to-temp-then-rename pattern to guarantee atomicity even
// if the process crashes mid-write.
type CursorTracker struct {
	filePath string
}

// NewCursorTracker creates a CursorTracker that persists cursor state to
// the given filePath.
func NewCursorTracker(filePath string) *CursorTracker {
	return &CursorTracker{filePath: filePath}
}

// Read returns the persisted cursor value from the JSON file.
// If the file does not exist (first run), it returns 0 with no error.
func (ct *CursorTracker) Read() (int64, error) {
	data, err := os.ReadFile(ct.filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	var state cursorState
	if err := json.Unmarshal(data, &state); err != nil {
		return 0, err
	}

	return state.Cursor, nil
}

// Write atomically persists the cursor value to the JSON file.
// It writes to a temporary file in the same directory and then renames
// it to the target path. Because rename is atomic on POSIX filesystems,
// the cursor file is never left in a partially-written state.
func (ct *CursorTracker) Write(cursor int64) error {
	state := cursorState{Cursor: cursor}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	dir := filepath.Dir(ct.filePath)

	// Create a temp file in the same directory so rename is guaranteed
	// to be atomic (same filesystem).
	tmp, err := os.CreateTemp(dir, ".cursor-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	// Clean up the temp file on any failure path.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}

	// Sync to ensure data hits disk before rename.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, ct.filePath); err != nil {
		return err
	}

	success = true
	return nil
}
