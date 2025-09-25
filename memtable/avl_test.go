package memtable

import (
	"main/keys"
	"math"
	"testing"
)

func TestComplexTree(t *testing.T) {
	avlTree := new(AVLTree)

	// Initial size should be 0
	if avlTree.Size() != 0 {
		t.Errorf("Expected initial size to be 0, got %d", avlTree.Size())
	}

	// Insert a series of testKeys to force multiple rotations
	testKeys := []uint32{50, 30, 70, 20, 40, 60, 80, 15, 25, 35, 45, 55, 65, 75, 85}
	for _, key := range testKeys {
		avlTree.Put(keys.NewIntKey(key), []byte{byte(key)})
	}

	// Check size after initial insertions
	if avlTree.Size() != uint32(len(testKeys)) {
		t.Errorf("Expected size %d after insertions, got %d", len(testKeys), avlTree.Size())
	}

	// Verify tree structure is valid
	if avlTree.head.key.Compare(keys.NewIntKey(50)) != 0 {
		t.Errorf("Expected root key to be 50, got %d", avlTree.head.key.GetValue())
	}

	// Verify all keys are present
	for _, key := range testKeys {
		found, value := avlTree.Get(keys.NewIntKey(key))
		if !found && value == nil {
			t.Errorf("Key %d should be in the tree", key)
		} else if len(value) != 1 || value[0] != byte(key) {
			t.Errorf("Value for key %d is incorrect", key)
		}
	}

	// Test some non-existent keys
	nonExistentKeys := []uint32{10, 90, 42, 58}
	for _, key := range nonExistentKeys {
		found, value := avlTree.Get(keys.NewIntKey(key))
		if found && value != nil {
			t.Errorf("Key %d should not be in the tree", key)
		}
	}

	// Delete some keys and verify tree remains balanced
	deleteKeys := []uint32{30, 70, 20, 65}
	for _, key := range deleteKeys {
		avlTree.Delete(keys.NewIntKey(key))
		found, value := avlTree.Get(keys.NewIntKey(key))
		if found && value != nil {
			t.Errorf("Key %d should have been deleted", key)
		}
	}

	// Size should be unchanged after logical deletion (our implementation uses tombstones)
	if avlTree.Size() != uint32(len(testKeys)) {
		t.Errorf("Expected size %d after deletions (using tombstones), got %d",
			len(testKeys), avlTree.Size())
	}

	// Insert some new keys
	newKeys := []uint32{42, 58, 90, 5}
	for _, key := range newKeys {
		avlTree.Put(keys.NewIntKey(key), []byte{byte(key)})
	}

	// Check size after new insertions
	expectedSize := uint32(len(testKeys) + len(newKeys))
	if avlTree.Size() != expectedSize {
		t.Errorf("Expected size %d after new insertions, got %d",
			expectedSize, avlTree.Size())
	}

	// Update some existing keys
	updateKeys := []uint32{50, 40, 80}
	for _, key := range updateKeys {
		avlTree.Put(keys.NewIntKey(key), []byte{byte(key + 100)})
		found, value := avlTree.Get(keys.NewIntKey(key))
		if !found && value == nil || len(value) != 1 || value[0] != byte(key+100) {
			t.Errorf("Updated value for key %d is incorrect", key)
		}
	}

	// Size should not change when updating existing keys
	if avlTree.Size() != expectedSize {
		t.Errorf("Expected size to remain %d after updates, got %d",
			expectedSize, avlTree.Size())
	}

	// avlTree.Dump(true)

	remainingKeys := len(testKeys) - len(deleteKeys) + len(newKeys)
	maxExpectedHeight := int(1.44*math.Log2(float64(remainingKeys+2)) - 0.328)

	actualHeight := avlTree.head.height
	if actualHeight > maxExpectedHeight {
		t.Errorf("Tree height %d exceeds expected maximum %d for %d elements",
			actualHeight, maxExpectedHeight, remainingKeys)
	}

	avlTree.Clear()

	if avlTree.Size() != 0 {
		t.Errorf("Tree should have size 0 after clearning it")
	}

	if len(avlTree.ToKVs()) != 0 {
		t.Errorf("Tree should have 0 elements after clearning it")
	}
}
