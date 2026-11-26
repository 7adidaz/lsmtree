package lsmtree

import (
	"bytes"
	"errors"
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
	if got := mt.Len(); got != 20 {
		t.Fatalf("replayed %d records, want 20", got)
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
	if got := mt.Len(); got != 3 {
		t.Errorf("replayed %d records, want 3", got)
	}
	if got := fileSize(t, path); got != complete {
		t.Errorf("file is %d bytes after replay, want %d", got, complete)
	}
}

func TestWALReplayRejectsCorruptFrame(t *testing.T) {
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

	// Flip one bit inside the first record's payload (header is 8 bytes).
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 1)
	if _, err := f.ReadAt(b, 10); err != nil {
		t.Fatal(err)
	}
	b[0] ^= 0x01
	if _, err := f.WriteAt(b, 10); err != nil {
		t.Fatal(err)
	}
	f.Close()

	mt := memtable.NewMemTable(memtable.NewAVLTree())
	err = (&WriteAheadLog{dataLocation: path}).Replay(mt)
	if !errors.Is(err, errCorruptWAL) {
		t.Fatalf("Replay returned %v, want errCorruptWAL", err)
	}
	if mt.Len() != 0 {
		t.Errorf("replayed %d records from a corrupt log, want 0", mt.Len())
	}
	if got := fileSize(t, path); got != complete {
		t.Errorf("corrupt log was truncated to %d bytes, want untouched %d", got, complete)
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
