package memtable

import (
	"math"
	"testing"

	"main/keys"
)

func TestAVLTreeStaysBalanced(t *testing.T) {
	avlTree := new(AVLTree)

	testKeys := []uint32{50, 30, 70, 20, 40, 60, 80, 15, 25, 35, 45, 55, 65, 75, 85}
	for _, key := range testKeys {
		avlTree.Put(keys.NewIntKey(key), []byte{byte(key)})
	}

	if avlTree.head.key.Compare(keys.NewIntKey(50)) != 0 {
		t.Errorf("Expected root key to be 50")
	}

	maxExpectedHeight := int(1.44*math.Log2(float64(len(testKeys)+2)) - 0.328)
	if avlTree.head.height > maxExpectedHeight {
		t.Errorf("Tree height %d exceeds expected maximum %d for %d elements",
			avlTree.head.height, maxExpectedHeight, len(testKeys))
	}
}
