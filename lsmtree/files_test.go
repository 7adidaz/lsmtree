package lsmtree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseIDRoundTrip(t *testing.T) {
	for _, id := range []uint64{0, 42, 1792851066804385000} {
		got, ok := parseID(fileName(sstablePrefix, id), sstablePrefix)
		if !ok || got != id {
			t.Errorf("parseID(fileName(%d)) = %d, %v", id, got, ok)
		}
	}

	for _, name := range []string{".DS_Store", "sstable_", "sstable_abc", "sstable_-5", "xsstable_5", "WAL_00000000000000000001"} {
		if _, ok := parseID(name, sstablePrefix); ok {
			t.Errorf("parseID(%q) accepted a name that isn't an SSTable", name)
		}
	}
}

func TestNextFreeIDSkipsUsedIDs(t *testing.T) {
	dir := t.TempDir()
	if got, err := nextFreeID(dir); err != nil || got != 0 {
		t.Fatalf("empty dir: got %d, %v; want 0", got, err)
	}

	for _, name := range []string{fileName(sstablePrefix, 3), fileName(sstablePrefix, 9), ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := nextFreeID(dir); err != nil || got != 10 {
		t.Errorf("got %d, %v; want 10", got, err)
	}
}
