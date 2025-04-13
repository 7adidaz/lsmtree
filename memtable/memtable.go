package memtable

import "lsmtree/interfaces"

type MemTable struct {
}

type MemTableImplementation interface {
	Get(key interfaces.Comparable) interfaces.Comparable
	Put(key interfaces.Comparable, val []byte)
	Delete(key interfaces.Comparable)
}

func (t MemTable) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (t MemTable) Put(key []byte, val []byte) ([]byte, error) {
	return nil, nil
}

func (t MemTable) Delete(key []byte) ([]byte, error) {
	return nil, nil
}

func (t MemTable) Flush() error {
	return nil
}
