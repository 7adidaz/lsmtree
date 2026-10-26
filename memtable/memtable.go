package memtable

import (
	"main/interfaces"
)

type MemTable struct {
	tree MemTableImplementation
}

type Entry struct {
	Key   interfaces.Comparable
	Value []byte
}

func NewMemTable(impl MemTableImplementation) *MemTable {
	return &MemTable{tree: impl}
}

type MemTableImplementation interface {
	Get(key interfaces.Comparable) (bool, []byte)
	Put(key interfaces.Comparable, val []byte)
	Floor(key interfaces.Comparable) []byte
	Ceil(key interfaces.Comparable) []byte
	Clear()
	Size() uint32
	ToKVs() []*Entry
}

func (t *MemTable) Get(key interfaces.Comparable) (bool, []byte) {
	return t.tree.Get(key)
}

func (t *MemTable) Put(key interfaces.Comparable, val []byte) {
	t.tree.Put(key, val)
}

func (t *MemTable) Size() uint32 {
	if t.tree == nil {
		return 0
	}
	return t.tree.Size()
}

func (t *MemTable) ToKVs() []*Entry {
	return t.tree.ToKVs()
}

func (t *MemTable) Clear() {
	t.tree.Clear()
}
