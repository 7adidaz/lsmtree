package memtable

import (
	"bytes"
	"encoding/hex"
	"lsmtree/keys"
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
	if found || result != nil {
		t.Errorf("Expected nil for non-existent key, got %s", result)
	}

	// Test Delete operation
	memtable.Delete(key1)
	found, result = memtable.Get(key1)
	if found || result != nil {
		t.Errorf("Expected nil for deleted key, got %s", result)
	}

	// Ensure key2 still exists
	found, result = memtable.Get(key2)
	if !found || !bytes.Equal(result, val3) {
		t.Errorf("Expected %s, got %s", val3, result)
	}

	// Test size
	if avl.size != 2 { // key1 (tombstoned) and key2
		t.Errorf("Expected size 2, got %d", avl.size)
	}

}

func TestMemTableDump(t *testing.T) {
	// Create AVL tree
	avl := NewAVLTree()

	// Create memtable with AVL tree
	memtable := NewMemTable(avl)

	key1 := keys.NewIntKey(1)
	val1 := []byte("value1")
	memtable.Put(key1, val1)

	key2 := keys.NewIntKey(2)
	val2 := []byte("value2")
	memtable.Put(key2, val2)

	memtable.Delete(key1)

	// avl.Dump(true)
	buf := new(bytes.Buffer)
	memtable.Dump(buf, nil, nil, 0)

	//  2 values|key->type-value|value len|TOMBSTONE|key->type-value|value len| data
	//  00000002|(00)00000001   |00000001 |7f       |(00)00000002   |00000006 |76616c756532
	bufExpectedVal := "000000020000000001000000017f00000000020000000676616c756532"
	if hex.EncodeToString(buf.Bytes()) != bufExpectedVal {
		t.Errorf("Output of the flush doesn't match expected value")
	}
}

func TestMemTableLoad(t *testing.T) {
	// Create AVL tree
	avl := NewAVLTree()

	// Create memtable with AVL tree
	memtable := NewMemTable(avl)

	readBuf := new(bytes.Buffer)
	dumpBytes, err := hex.DecodeString("000000020000000001000000017f00000000020000000676616c756532")
	if err != nil {
		t.Error("Unable to decode string")
	}
	readBuf.Write(dumpBytes)

	memtable.Load(readBuf)

	if memtable.Size() != 2 {
		t.Error("Unable to load dumped data")
	}

	found, result := memtable.Get(keys.NewIntKey(1))
	if found || result != nil {
		t.Errorf("Expected nil for deleted key, got %s", result)
	}

	found, result = memtable.Get(keys.NewIntKey(1))
	if found || result != nil {
		t.Errorf("Expected nil for deleted key, got %s", result)
	}

	found, result = memtable.Get(keys.NewIntKey(2))
	val2 := []byte("value2")
	if !found || !bytes.Equal(result, val2) {
		t.Errorf("Expected %s, got %s", val2, result)
	}
}
