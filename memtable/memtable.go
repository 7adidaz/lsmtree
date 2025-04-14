package memtable

import "lsmtree/interfaces"

type MemTable struct {
	tree MemTableImplementation
}

func NewMemTable(tree *MemTable) *MemTable {
	return &MemTable{tree: tree}
}

type MemTableImplementation interface {
	Get(key interfaces.Comparable) []byte
	Put(key interfaces.Comparable, val []byte)
	Delete(key interfaces.Comparable)
}

func (t *MemTable) Get(key interfaces.Comparable) []byte {
	return t.tree.Get(key)
}

func (t *MemTable) Put(key interfaces.Comparable, val []byte){
	t.tree.Put(key, val)
}

func (t *MemTable) Delete(key interfaces.Comparable) {
	t.tree.Delete(key)
}

func (t MemTable) Flush() error {
	return nil
}
