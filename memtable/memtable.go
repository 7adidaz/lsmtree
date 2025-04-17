package memtable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"lsmtree/interfaces"
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
	Clear()
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

func (t *MemTable) Flush(file io.Writer) error {
	/*
		     * Binary Format:
		     * [4 bytes] - Table size (uint32)
		     * For each entry:
		     *   [1 byte]  - Key type (0x00 for IntKey)
		     *   [4 bytes] - Key value (uint32)
		     *   [4 bytes] - Value length (int32)
		     *   [N bytes] - Value data
			 *
			 *  the caller should close the file.
	*/

	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, t.Size()); err != nil {
		return fmt.Errorf("error serializing table size: %w", err)
	}

	for _, entry := range t.tree.ToKVs() {
		// if bytes.Equal(entry.value, []byte{0x7f}) {
		//     continue
		// }

		// Write key
		keyBytes, err := entry.key.ToBytes()
		if err != nil {
			return fmt.Errorf("error serializing key: %w", err)
		}
		if _, err := buf.Write(keyBytes); err != nil {
			return fmt.Errorf("error writing key: %w", err)
		}

		// Write value length
		valueLen := int32(len(entry.value))
		if err := binary.Write(buf, binary.BigEndian, valueLen); err != nil {
			return fmt.Errorf("error serializing value length: %w", err)
		}

		// Write value data
		if _, err := buf.Write(entry.value); err != nil {
			return fmt.Errorf("error writing value: %w", err)
		}
	}

	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("error writing buffer to file")
	}
	t.tree.Clear()

	return nil
}
