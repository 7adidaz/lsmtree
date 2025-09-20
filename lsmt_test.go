package lsmtree

import (
	"fmt"
	"lsmtree/keys"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestLSMTree(t *testing.T) {
	lsm := newLSMTree(10, 2, 0.01)
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
	lsm.Delete(key)
	found, got, _ = lsm.Get(key)
	if found && got != nil {
		t.Errorf("Expected nil after delete, got '%v'", got)
	}
}

func TestMaintainTheLatestVersionOfKey(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	lsm := newLSMTree(10, 10, 0.01)
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
	lsm := newLSMTree(10, 2, 0.01)
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
	lsm := newLSMTree(50, 3, 0.01)
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
	lsm := newLSMTree(5, 3, 0.01)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 20
	keyToValue := make(map[int]string)
	keysArr := make([]int, n)
	for i := range n {
		s := r.Intn(1000)
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
		lsm.Delete(keys.NewIntKey(uint32(k)))
	}
	for i := range n / 2 {
		k := keysArr[i]
		found, got, err := lsm.Get(keys.NewIntKey(uint32(k)))
		if err != nil {
			t.Fatalf("Get failed for deleted key %d: %v", k, err)
		}
		if found && got != nil {
			t.Errorf("Expected nil for deleted key '%d', got '%s'", k, string(got))
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
	lsm := newLSMTree(1000, 10, 0.01)
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
