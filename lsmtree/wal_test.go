package lsmtree

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"main/keys"
	"main/memtable"
)

func TestWALAppendThenReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal")
	w, err := openWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	big := bytes.Repeat([]byte("v"), 5000)
	for i := range 20 {
		if err := w.Append(keys.NewStringKey(fmt.Sprintf("key-%02d", i)), big); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	mt := memtable.NewMemTable(memtable.NewAVLTree())
	if err := (&WriteAheadLog{dataLocation: path}).Replay(mt); err != nil {
		t.Fatal(err)
	}
	if mt.Size() != 20 {
		t.Fatalf("replayed %d records, want 20", mt.Size())
	}
	found, got := mt.Get(keys.NewStringKey("key-07"))
	if !found || !bytes.Equal(got, big) {
		t.Errorf("key-07 did not survive the round trip")
	}
}

func TestWALReplayTruncatesTornTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal")
	w, err := openWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		if err := w.Append(keys.NewStringKey(fmt.Sprintf("key-%d", i)), []byte("value")); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	complete := fileSize(t, path)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte{0x01, 0, 0, 0, 9, 'p', 'a'})
	f.Close()

	mt := memtable.NewMemTable(memtable.NewAVLTree())
	if err := (&WriteAheadLog{dataLocation: path}).Replay(mt); err != nil {
		t.Fatal(err)
	}
	if mt.Size() != 3 {
		t.Errorf("replayed %d records, want 3", mt.Size())
	}
	if got := fileSize(t, path); got != complete {
		t.Errorf("file is %d bytes after replay, want %d", got, complete)
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}
