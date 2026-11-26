package memtable

import (
	"math"
	"testing"

	"main/keys"
)

func TestSkipListLevelStaysBounded(t *testing.T) {
	list := NewSkipList()
	const n = 1000
	for i := range uint32(n) {
		list.Put(keys.NewIntKey(i), []byte{1})
	}

	// With p=1/4, the expected top level for n entries is around log4(n);
	// +10 leaves generous slack for randomness.
	maxExpectedLevel := int(math.Log2(n)/2) + 10
	if list.level < 1 || list.level > maxLevel || list.level > maxExpectedLevel {
		t.Errorf("List level %d out of bounds for %d elements", list.level, n)
	}
}
