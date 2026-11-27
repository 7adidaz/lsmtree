package bloomfilter_test

import (
	"crypto/rand"
	"encoding/binary"
	"main/bloomfilter"
	"main/keys"
	"testing"
)

func randomUint32() uint32 {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("crypto/rand failed")
	}
	return binary.LittleEndian.Uint32(b[:])
}

func TestBasic(t *testing.T) {
	bf := bloomfilter.NewBloomFilter(500, 0.01)

	arr := []uint32{27, 18, 11, 8, 44, 29, 5, 90, 58, 53}
	for _, i := range arr {
		bf.Insert(keys.NewIntKey(i))
	}

	for _, i := range arr {
		if !bf.Contains(keys.NewIntKey(i)) {
			t.Errorf("error getting inserted element %d", i)
		}
	}
	if bf.Contains(keys.NewIntKey(15)) {
		t.Errorf("Error getting non inserted element %d", 15)
	}
}

func TestProbability(t *testing.T) {
	entriesNumb := 7000
	bf := bloomfilter.NewBloomFilter(10000, 0.5)

	inserted := make(map[uint32]struct{})
	nonInserted := make(map[uint32]struct{})

	for len(inserted) < entriesNumb/2 {
		n := randomUint32()
		if _, exists := inserted[n]; !exists {
			inserted[n] = struct{}{}
			bf.Insert(keys.NewIntKey(n))
		}
	}

	for len(nonInserted) < entriesNumb/2 {
		n := randomUint32()
		if _, exists := inserted[n]; !exists {
			nonInserted[n] = struct{}{}
		}
	}

	falsePositives := 0
	for n := range nonInserted {
		if bf.Contains(keys.NewIntKey(n)) {
			falsePositives++
		}
	}

	falsePositiveRate := float64(falsePositives) / float64(len(nonInserted)) * 100
	t.Logf("False positive rate: %.2f%% (%d out of %d items)",
		falsePositiveRate, falsePositives, len(nonInserted))
}
