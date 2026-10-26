package memtable

import (
	"bytes"
	"main/keys"
	"testing"
)

func TestMemTable(t *testing.T) {
	// Create AVL tree
	avl := NewAVLTree()

	// Create memtable with AVL tree
	memtable := NewMemTable(avl)

	// Test Put and Get operations
	key1 := keys.NewIntKey(1)
	val1 := []byte("value1")
	memtable.Put(key1, val1)

	// Check if stored correctly
	found, result := memtable.Get(key1)
	if !found || !bytes.Equal(result, val1) {
		t.Errorf("Expected %s, got %s", val1, result)
	}

	// Test overwriting existing key
	val2 := []byte("updated_value1")
	memtable.Put(key1, val2)
	found, result = memtable.Get(key1)
	if !found || !bytes.Equal(result, val2) {
		t.Errorf("Expected %s, got %s", val2, result)
	}

	// Test multiple keys
	key2 := keys.NewIntKey(2)
	val3 := []byte("value2")
	memtable.Put(key2, val3)

	found, result = memtable.Get(key2)
	if !found || !bytes.Equal(result, val3) {
		t.Errorf("Expected %s, got %s", val3, result)
	}

	// Test non-existent key
	key3 := keys.NewIntKey(3)
	found, result = memtable.Get(key3)
	if found && result != nil {
		t.Errorf("Expected nil for non-existent key, got %s", result)
	}

	// Ensure key2 still exists
	found, result = memtable.Get(key2)
	if !found || !bytes.Equal(result, val3) {
		t.Errorf("Expected %s, got %s", val3, result)
	}

	// Test size
	if avl.size != 2 { // key1 and key2
		t.Errorf("Expected size 2, got %d", avl.size)
	}

}
