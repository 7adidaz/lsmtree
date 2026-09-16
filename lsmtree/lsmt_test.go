package lsmtree

import (
	"bytes"
	"fmt"
	"main/keys"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestLSMTree(t *testing.T) {
	chdirTemp(t)

	lsm := NewLSMTree(10, 2, 0.01)
	key := keys.NewStringKey("foo")
	value := []byte("bar")
	err := lsm.Put(key, value)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	found, got, err := lsm.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || got == nil || string(got) != "bar" {
		t.Errorf("Expected value 'bar', got '%v'", got)
	}
	if err := lsm.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	found, got, _ = lsm.Get(key)
	if found {
		t.Errorf("Expected key to be absent after delete, got '%s'", got)
	}
}

func TestMaintainTheLatestVersionOfKey(t *testing.T) {
	chdirTemp(t)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	lsm := NewLSMTree(10, 10, 0.01)
	n := 50
	latest_val := ""
	for i := range n {
		s := "val_" + strconv.Itoa(r.Intn(1000))
		latest_val = s
		err := lsm.Put(keys.NewStringKey("KEY"), []byte(s))
		if err != nil {
			t.Fatalf("Put failed at %d: %v", i, err)
		}
	}
	found, got, err := lsm.Get(keys.NewStringKey("KEY"))
	if err != nil {
		t.Fatalf("Get failed for key %s: %v", "KEY", err)
	}
	if !found || got == nil || string(got) != latest_val {
		t.Errorf("Expected value '%s' for key '%s', got '%s'", latest_val, "KEY", string(got))
	}
}

func TestMaintainLatestVersionWithCompaction(t *testing.T) {
	chdirTemp(t)

	lsm := NewLSMTree(10, 2, 0.01)
	testKey := keys.NewStringKey("test-key")
	latestValue := ""
	for i := range 100 {
		randomKey := keys.NewStringKey(fmt.Sprintf("key-%d", i))
		err := lsm.Put(randomKey, fmt.Appendf(nil, "value-%d", i))
		if err != nil {
			t.Fatalf("Put failed for random key at %d: %v", i, err)
		}
		if i%10 == 4 {
			latestValue = fmt.Sprintf("updated-value-%d", i)
			err := lsm.Put(testKey, []byte(latestValue))
			if err != nil {
				t.Fatalf("Put failed for test key at %d: %v", i, err)
			}
		}
	}
	found, got, err := lsm.Get(testKey)
	if err != nil {
		t.Fatalf("Get failed for test key: %v", err)
	}
	if !found || got == nil || string(got) != latestValue {
		t.Errorf("Expected value '%s' for test key, got '%s'", latestValue, string(got))
	}
}

func TestLSMTreeBulkRandom(t *testing.T) {
	chdirTemp(t)

	lsm := NewLSMTree(50, 3, 0.01)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 20000
	keyToValue := make(map[int]string)
	keysArr := make([]int, n)
	for i := range n {
		s := r.Intn(1000000)
		keysArr[i] = s
		val := "val_" + strconv.Itoa(i)
		keyToValue[s] = val
		err := lsm.Put(keys.NewIntKey(uint32(s)), []byte(val))
		if err != nil {
			t.Fatalf("Put failed at %d: %v", i, err)
		}
	}
	for range n / 3 {
		idx := r.Intn(n)
		k := keysArr[idx]
		want := keyToValue[k]
		found, got, err := lsm.Get(keys.NewIntKey(uint32(k)))
		if err != nil {
			t.Fatalf("Get failed for key %d: %v", k, err)
		}
		if !found || got == nil || string(got) != want {
			t.Errorf("Expected value '%s' for key '%d', got '%s'", want, k, string(got))
		}
	}
}

func TestDeleteMultipleKeysAndCheckNil(t *testing.T) {
	chdirTemp(t)

	lsm := NewLSMTree(5, 3, 0.01)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 20
	keyToValue := make(map[int]string)
	keysArr := make([]int, n)
	seen := make(map[int]bool, n)
	for i := range n {
		s := r.Intn(1000)
		for seen[s] {
			s = r.Intn(1000)
		}
		seen[s] = true
		keysArr[i] = s
		val := "val_" + strconv.Itoa(i)
		keyToValue[s] = val
		err := lsm.Put(keys.NewIntKey(uint32(s)), []byte(val))
		if err != nil {
			t.Fatalf("Put failed at %d: %v", i, err)
		}
	}
	for i := range n / 2 {
		k := keysArr[i]
		if err := lsm.Delete(keys.NewIntKey(uint32(k))); err != nil {
			t.Fatalf("Delete failed for key %d: %v", k, err)
		}
	}
	for i := range n / 2 {
		k := keysArr[i]
		found, got, err := lsm.Get(keys.NewIntKey(uint32(k)))
		if err != nil {
			t.Fatalf("Get failed for deleted key %d: %v", k, err)
		}
		if found {
			t.Errorf("Expected deleted key '%d' to be absent, got '%s'", k, string(got))
		}
	}
	for i := n / 2; i < n; i++ {
		k := keysArr[i]
		want := keyToValue[k]
		found, got, err := lsm.Get(keys.NewIntKey(uint32(k)))
		if err != nil {
			t.Fatalf("Get failed for key %d: %v", k, err)
		}
		if !found || got == nil || string(got) != want {
			t.Errorf("Expected value '%s' for key '%d', got '%s'", want, k, string(got))
		}
	}
}

func BenchmarkLSMTreePutGet(b *testing.B) {
	chdirTemp(b)

	lsm := NewLSMTree(1000, 10, 0.01)
	n := 100000
	for i := 0; i < n; i++ {
		key := keys.NewIntKey(uint32(i))
		value := []byte("val_" + strconv.Itoa(i))
		if err := lsm.Put(key, value); err != nil {
			b.Fatalf("Put failed at %d: %v", i, err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % n
		key := keys.NewIntKey(uint32(idx))
		_, got, err := lsm.Get(key)
		if err != nil {
			b.Fatalf("Get failed for key %d: %v", idx, err)
		}
		if got == nil || string(got) != "val_"+strconv.Itoa(idx) {
			b.Errorf("Expected value 'val_%d', got '%s'", idx, string(got))
		}
	}
}

func TestSSTablesSurviveRestart(t *testing.T) {
	chdirTemp(t)

	const threshold = 10
	const n = 40

	first := NewLSMTree(threshold, 2, 0.01)
	for i := range n {
		k := keys.NewStringKey(fmt.Sprintf("key-%02d", i))
		if err := first.Put(k, fmt.Appendf(nil, "value-%02d", i)); err != nil {
			t.Fatalf("Put failed for key-%02d: %v", i, err)
		}
	}
	if len(first.SStables) == 0 {
		t.Fatal("expected at least one SSTable to be flushed")
	}

	persisted := len(first.SStables) * threshold

	reopened := NewLSMTree(threshold, 2, 0.01)
	if got, want := len(reopened.SStables), len(first.SStables); got != want {
		t.Fatalf("reloaded %d SSTables, want %d", got, want)
	}
	for i, table := range reopened.SStables {
		if table.dataLength == 0 {
			t.Errorf("SSTable %d reloaded with dataLength 0", i)
		}
	}

	for i := range persisted {
		k := keys.NewStringKey(fmt.Sprintf("key-%02d", i))
		found, got, err := reopened.Get(k)
		if err != nil {
			t.Errorf("Get failed for key-%02d: %v", i, err)
			continue
		}
		if !found {
			t.Errorf("key-%02d is missing after restart", i)
			continue
		}
		if want := fmt.Sprintf("value-%02d", i); string(got) != want {
			t.Errorf("key-%02d has value %q, want %q", i, got, want)
		}
	}
}

func chdirTemp(tb testing.TB) {
	tb.Helper()

	orig, err := os.Getwd()
	if err != nil {
		tb.Fatal(err)
	}
	if err := os.Chdir(tb.TempDir()); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			tb.Fatal(err)
		}
	})
}

// The tombstone lands in a newer SSTable than the value, so Get has to stop at
// the tombstone instead of falling through to the older table.
func TestDeleteSurvivesFlushIntoOlderSSTable(t *testing.T) {
	chdirTemp(t)

	const threshold = 10
	lsm := NewLSMTree(threshold, 2, 0.01)
	victim := keys.NewStringKey("victim")

	if err := lsm.Put(victim, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	for i := range threshold - 1 {
		if err := lsm.Put(keys.NewStringKey(fmt.Sprintf("a-%02d", i)), []byte("x")); err != nil {
			t.Fatal(err)
		}
	}

	if err := lsm.Delete(victim); err != nil {
		t.Fatal(err)
	}
	for i := range threshold - 1 {
		if err := lsm.Put(keys.NewStringKey(fmt.Sprintf("b-%02d", i)), []byte("x")); err != nil {
			t.Fatal(err)
		}
	}

	if len(lsm.SStables) < 2 {
		t.Fatalf("expected the value and the tombstone in separate SSTables, got %d", len(lsm.SStables))
	}

	found, got, err := lsm.Get(victim)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Errorf("deleted key was resurrected from an older SSTable with %q", got)
	}
}

// A value of 0x7f used to be swallowed as a tombstone sentinel.
func TestArbitraryByteValuesRoundTrip(t *testing.T) {
	chdirTemp(t)

	lsm := NewLSMTree(10, 2, 0.01)
	cases := map[string][]byte{
		"tombstone-byte": {0x7f},
		"record-value":   {0x00},
		"record-tomb":    {0x01},
		"empty":          {},
	}
	for name, want := range cases {
		if err := lsm.Put(keys.NewStringKey(name), want); err != nil {
			t.Fatal(err)
		}
	}
	for name, want := range cases {
		found, got, err := lsm.Get(keys.NewStringKey(name))
		if err != nil {
			t.Fatalf("%s: Get failed: %v", name, err)
		}
		if !found {
			t.Errorf("%s: expected found", name)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}
