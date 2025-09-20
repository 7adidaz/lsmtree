package bloomfilter_test

import (
	"lsmtree/bloomfilter"
	"lsmtree/keys"
	"math/rand"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	bf := bloomfilter.NewBloomFilter(500, 0.01)

	arr := []uint32{27, 18, 11, 8, 44, 29, 5, 90, 58, 53}
	for _, i := range arr {
		err := bf.Insert(keys.NewIntKey(i))
		if err != nil {
			t.Errorf("Error inserting element %d", i)
		}
	}

	for _, i := range arr {
		val, er := bf.Contains(keys.NewIntKey(i))
		if !val || er != nil {
			t.Errorf("error getting inserted element %d", i)
		}
	}
	val, er := bf.Contains(keys.NewIntKey(15))
	if val || er != nil {
		t.Errorf("Error getting non inserted element %d", 15)
	}
}

func TestProbability(t *testing.T) {
	entriesNumb := 7000
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bf := bloomfilter.NewBloomFilter(10000, 0.01)

	inserted := make(map[uint32]struct{})
	nonInserted := make(map[uint32]struct{})

	for len(inserted) < entriesNumb/2 {
		n := uint32(r.Intn(10000000))
		if _, exists := inserted[n]; !exists {
			inserted[n] = struct{}{}
			if err := bf.Insert(keys.NewIntKey(n)); err != nil {
				t.Fatalf("insert failed: %v", err)
			}
		}
	}

	for len(nonInserted) < entriesNumb/2 {
		n := uint32(r.Intn(10000000))
		if _, exists := inserted[n]; !exists {
			nonInserted[n] = struct{}{}
		}
	}

	falsePositives := 0
	for n := range nonInserted {
		found, _ := bf.Contains(keys.NewIntKey(n))
		if found {
			falsePositives++
		}
	}

	falsePositiveRate := float64(falsePositives) / float64(len(nonInserted)) * 100
	t.Logf("False positive rate: %.2f%% (%d out of %d items)",
		falsePositiveRate, falsePositives, len(nonInserted))
}
