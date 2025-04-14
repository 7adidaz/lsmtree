package memtable

import (
	"math"
	"testing"
)

func TestComplexTree(t *testing.T) {
	avlTree := new(AVLTree)

	// Insert a series of keys to force multiple rotations
	keys := []uint{50, 30, 70, 20, 40, 60, 80, 15, 25, 35, 45, 55, 65, 75, 85}
	for _, key := range keys {
		avlTree.Put(NewIntKey(key), []byte{byte(key)})
	}

	// Verify tree structure is valid
	if avlTree.head.key.Compare(NewIntKey(50)) != 0 {
		t.Errorf("Expected root key to be 50, got %d", avlTree.head.key.GetValue())
	}

	// Verify all keys are present
	for _, key := range keys {
		value := avlTree.Get(NewIntKey(key))
		if value == nil {
			t.Errorf("Key %d should be in the tree", key)
		} else if len(value) != 1 || value[0] != byte(key) {
			t.Errorf("Value for key %d is incorrect", key)
		}
	}

	// Test some non-existent keys
	nonExistentKeys := []uint{10, 90, 42, 58}
	for _, key := range nonExistentKeys {
		if avlTree.Get(NewIntKey(key)) != nil {
			t.Errorf("Key %d should not be in the tree", key)
		}
	}

	// Delete some keys and verify tree remains balanced
	deleteKeys := []uint{30, 70, 20, 65}
	for _, key := range deleteKeys {
		avlTree.Delete(NewIntKey(key))
		if avlTree.Get(NewIntKey(key)) != nil {
			t.Errorf("Key %d should have been deleted", key)
		}
	}

	// Insert some new keys
	newKeys := []uint{42, 58, 90, 5}
	for _, key := range newKeys {
		avlTree.Put(NewIntKey(key), []byte{byte(key)})
	}

	// Update some existing keys
	updateKeys := []uint{50, 40, 80}
	for _, key := range updateKeys {
		avlTree.Put(NewIntKey(key), []byte{byte(key + 100)})
		value := avlTree.Get(NewIntKey(key))
		if value == nil || len(value) != 1 || value[0] != byte(key+100) {
			t.Errorf("Updated value for key %d is incorrect", key)
		}
	}

	avlTree.Dump(true)

	remainingKeys := len(keys) - len(deleteKeys) + len(newKeys)
	maxExpectedHeight := int(1.44*math.Log2(float64(remainingKeys+2)) - 0.328)

	actualHeight := avlTree.head.height
	if actualHeight > maxExpectedHeight {
		t.Errorf("Tree height %d exceeds expected maximum %d for %d elements",
			actualHeight, maxExpectedHeight, remainingKeys)
	}
}
