package memtable

import (
	"encoding/hex"
	"lsmtree/interfaces"
	"lsmtree/util"
)

type MemTable struct {
	tree MemTableImplementation
}

type Entry struct {
	key   interfaces.Comparable
	value []byte
}

func NewMemTable(impl MemTableImplementation) *MemTable {
	return &MemTable{tree: impl}
}

type MemTableImplementation interface {
	Get(key interfaces.Comparable) []byte
	Put(key interfaces.Comparable, val []byte)
	Delete(key interfaces.Comparable)
	Size() uint32
	ToKVs() []*Entry
}

func (t *MemTable) Get(key interfaces.Comparable) []byte {
	return t.tree.Get(key)
}

func (t *MemTable) Put(key interfaces.Comparable, val []byte) {
	t.tree.Put(key, val)
}

func (t *MemTable) Delete(key interfaces.Comparable) {
	t.tree.Delete(key)
}

func (t *MemTable) Size() uint32 {
	if t.tree == nil {
		return 0
	}
	return t.tree.Size()
}

/*
 * TODO: continue from here
 *
 * make sure the serialized data have the correct length 
 * main bug issues (uint32  length is 2)
 * 
	keylength 2
	datalength 1
	entrydata 1
	keylength 2
	datalength 1
	entrydata 6
 *
 */

func (t *MemTable) Flush() error {
	var seralizedData []byte
	seralizedData = append(seralizedData, byte(t.Size()))
	for _, entry := range t.tree.ToKVs() {
		key, ok := entry.key.ToBytes()
		if ok != nil {
			panic("Error on serializing Key")
		}
		println("keylength", len(key))
		seralizedData = append(seralizedData, key...)
		dataLength, ok := util.ToByteArray(int32(len(entry.value)))
		if ok != nil {
			panic("Error on serializing DataLength")
		}
		seralizedData = append(seralizedData, dataLength...)
		println("datalength", len(dataLength))
		seralizedData = append(seralizedData, entry.value...)
		println("entrydata", len(entry.value))
	}

	println("seralized: ", hex.EncodeToString(seralizedData))

	return nil
}
