package lsmtree

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"main/bloomfilter"
	"main/keys"
	"main/memtable"
)

func TestEncodeSSTable(t *testing.T) {
	mt := memtable.NewMemTable(memtable.NewAVLTree())
	mt.Put(keys.NewIntKey(1), []byte("value1"))
	mt.Put(keys.NewIntKey(2), []byte("value2"))
	mt.Put(keys.NewIntKey(1), []byte{0x7f})

	buf := new(bytes.Buffer)
	if err := encodeSSTable(buf, mt.ToKVs(), nil, nil, 0); err != nil {
		t.Fatal(err)
	}

	//  key->type-value|value len|value|key->type-value|value len|data
	//  (00)00000001   |00000001 |7f   |(00)00000002   |00000006 |76616c756532
	bufExpectedVal := "0000000001000000017f00000000020000000676616c756532"
	if hex.EncodeToString(buf.Bytes()) != bufExpectedVal {
		t.Errorf("Output of the flush doesn't match expected value")
	}
}

func TestScanSSTable(t *testing.T) {
	data, err := hex.DecodeString("0000000001000000017f00000000020000000676616c756532")
	if err != nil {
		t.Fatal(err)
	}

	index := memtable.NewAVLTree()
	bloomFilter := bloomfilter.NewBloomFilter(10, 0.01)
	if err := scanSSTable(bytes.NewReader(data), bloomFilter, index, 1); err != nil {
		t.Fatal(err)
	}

	// the first record is 10 bytes, so key 2 starts at offset 10
	for key, want := range map[uint32]uint32{1: 0, 2: 10} {
		k := keys.NewIntKey(key)
		if found, _ := bloomFilter.Contains(k); !found {
			t.Errorf("bloom filter is missing key %d", key)
		}
		found, got := index.Get(k)
		if !found || binary.BigEndian.Uint32(got) != want {
			t.Errorf("index offset for key %d = %v, want %d", key, got, want)
		}
	}
}

func TestEncodeThenScanBuildSameIndex(t *testing.T) {
	mt := memtable.NewMemTable(memtable.NewAVLTree())
	for i := range 100 {
		mt.Put(keys.NewStringKey(fmt.Sprintf("key-%03d", i)), bytes.Repeat([]byte("v"), i*50))
	}

	buf := new(bytes.Buffer)
	written := memtable.NewAVLTree()
	if err := encodeSSTable(buf, mt.ToKVs(), nil, written, 3); err != nil {
		t.Fatal(err)
	}

	scanned := memtable.NewAVLTree()
	if err := scanSSTable(bytes.NewReader(buf.Bytes()), nil, scanned, 3); err != nil {
		t.Fatal(err)
	}

	want, got := written.ToKVs(), scanned.ToKVs()
	if len(want) != len(got) {
		t.Fatalf("index has %d entries after scan, want %d", len(got), len(want))
	}
	for i := range want {
		if want[i].Key.Compare(got[i].Key) != 0 || !bytes.Equal(want[i].Value, got[i].Value) {
			t.Errorf("index entry %d differs after scan", i)
		}
	}
}
