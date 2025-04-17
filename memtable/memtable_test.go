package memtable

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestMemTable(t *testing.T) {
	// Create AVL tree
	avl := NewAVLTree()

	// Create memtable with AVL tree
	memtable := NewMemTable(avl)

	// Test Put and Get operations
	key1 := NewIntKey(1)
	val1 := []byte("value1")
	memtable.Put(key1, val1)

	// Check if stored correctly
	result := memtable.Get(key1)
	if !bytes.Equal(result, val1) {
		t.Errorf("Expected %s, got %s", val1, result)
	}

	// Test overwriting existing key
	val2 := []byte("updated_value1")
	memtable.Put(key1, val2)
	result = memtable.Get(key1)
	if !bytes.Equal(result, val2) {
		t.Errorf("Expected %s, got %s", val2, result)
	}

	// Test multiple keys
	key2 := NewIntKey(2)
	val3 := []byte("value2")
	memtable.Put(key2, val3)

	result = memtable.Get(key2)
	if !bytes.Equal(result, val3) {
		t.Errorf("Expected %s, got %s", val3, result)
	}

	// Test non-existent key
	key3 := NewIntKey(3)
	result = memtable.Get(key3)
	if result != nil {
		t.Errorf("Expected nil for non-existent key, got %s", result)
	}

	// Test Delete operation
	memtable.Delete(key1)
	result = memtable.Get(key1)
	if result != nil {
		t.Errorf("Expected nil for deleted key, got %s", result)
	}

	// Ensure key2 still exists
	result = memtable.Get(key2)
	if !bytes.Equal(result, val3) {
		t.Errorf("Expected %s, got %s", val3, result)
	}

	// Test size
	if avl.size != 2 { // key1 (tombstoned) and key2
		t.Errorf("Expected size 2, got %d", avl.size)
	}

	// avl.Dump(true)
    buf := new(bytes.Buffer)
	memtable.Flush(buf)

	//  2 values|key->type-value|value len|TOMBSTONE|key->type-value|value len| data
	//  00000002|(00)00000001   |00000001 |7f       |(00)00000002   |00000006 |76616c756532
	bufExpectedVal := "000000020000000001000000017f00000000020000000676616c756532"
	if hex.EncodeToString(buf.Bytes()) != bufExpectedVal {
		t.Errorf("Output of the flush doesn't match expected value")
	}
}
