package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type testData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestStorage_PutAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	data := testData{ID: "123", Name: "test", Value: 42}

	// Put data
	err := s.Put(ctx, []string{"items", "item1"}, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, "items", "item1.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Get data
	var retrieved testData
	err = s.Get(ctx, []string{"items", "item1"}, &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != data.ID || retrieved.Name != data.Name || retrieved.Value != data.Value {
		t.Errorf("Data mismatch: got %+v, want %+v", retrieved, data)
	}
}

func TestStorage_GetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	var data testData
	err := s.Get(ctx, []string{"nonexistent", "item"}, &data)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	data := testData{ID: "123", Name: "test", Value: 42}

	// Put then delete
	err := s.Put(ctx, []string{"items", "toDelete"}, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	err = s.Delete(ctx, []string{"items", "toDelete"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	var retrieved testData
	err = s.Get(ctx, []string{"items", "toDelete"}, &retrieved)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestStorage_DeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// Deleting nonexistent should not error
	err := s.Delete(ctx, []string{"nonexistent", "item"})
	if err != nil {
		t.Errorf("Delete of nonexistent item should not error: %v", err)
	}
}

func TestStorage_List(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// Create multiple items
	for i := 0; i < 3; i++ {
		data := testData{ID: string(rune('a' + i)), Name: "test", Value: i}
		err := s.Put(ctx, []string{"items", data.ID}, data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// List items
	items, err := s.List(ctx, []string{"items"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d: %v", len(items), items)
	}
}

func TestStorage_ListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// List nonexistent directory
	items, err := s.List(ctx, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected empty list, got: %v", items)
	}
}

func TestStorage_Scan(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// Create items
	expected := map[string]testData{
		"a": {ID: "a", Name: "first", Value: 1},
		"b": {ID: "b", Name: "second", Value: 2},
		"c": {ID: "c", Name: "third", Value: 3},
	}

	for id, data := range expected {
		err := s.Put(ctx, []string{"items", id}, data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Scan items
	scanned := make(map[string]testData)
	err := s.Scan(ctx, []string{"items"}, func(key string, data json.RawMessage) error {
		var item testData
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		scanned[key] = item
		return nil
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(scanned) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(scanned))
	}

	for id, exp := range expected {
		got, ok := scanned[id]
		if !ok {
			t.Errorf("Missing key %s", id)
			continue
		}
		if got.ID != exp.ID || got.Name != exp.Name || got.Value != exp.Value {
			t.Errorf("Mismatch for %s: got %+v, want %+v", id, got, exp)
		}
	}
}

func TestStorage_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// Should not exist initially
	if s.Exists(ctx, []string{"items", "test"}) {
		t.Error("Item should not exist")
	}

	// Create item
	data := testData{ID: "test", Name: "test", Value: 1}
	err := s.Put(ctx, []string{"items", "test"}, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Should exist now
	if !s.Exists(ctx, []string{"items", "test"}) {
		t.Error("Item should exist")
	}
}

func TestStorage_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// Concurrent writes to the same key
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			data := testData{ID: "concurrent", Name: "test", Value: val}
			err := s.Put(ctx, []string{"items", "concurrent"}, data)
			if err != nil {
				t.Errorf("Concurrent Put failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Should be able to read final value
	var retrieved testData
	err := s.Get(ctx, []string{"items", "concurrent"}, &retrieved)
	if err != nil {
		t.Fatalf("Get after concurrent writes failed: %v", err)
	}
}

func TestStorage_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	s := New(tmpDir)
	ctx := context.Background()

	// Write initial value
	data := testData{ID: "atomic", Name: "initial", Value: 1}
	err := s.Put(ctx, []string{"items", "atomic"}, data)
	if err != nil {
		t.Fatalf("Initial Put failed: %v", err)
	}

	// Verify no .tmp file exists after write
	tmpPath := filepath.Join(tmpDir, "items", "atomic.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after successful write")
	}
}
