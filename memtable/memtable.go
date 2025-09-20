package memtable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"lsmtree/bloomfilter"
	"lsmtree/interfaces"
	"lsmtree/keys"
	"lsmtree/util"
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
	Delete(key interfaces.Comparable)
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

func (t *MemTable) Delete(key interfaces.Comparable) {
	t.tree.Delete(key)
}

func (t *MemTable) Size() uint32 {
	if t.tree == nil {
		return 0
	}
	return t.tree.Size()
}


func (t *MemTable) Dump(file io.Writer, bloomfilter bloomfilter.BloomFilterImplementation, index MemTableImplementation, sampling int32) error  {
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
	
	i := int32(0)
	for _, entry := range t.tree.ToKVs() {
		// if bytes.Equal(entry.value, []byte{0x7f}) {
		//     continue
		// }

		if bloomfilter != nil {
			bloomfilter.Insert(entry.Key) 
		}

		// sparse index
		if index != nil && sampling > 0 && i%sampling == 0 {
			offsetBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(offsetBytes, uint32(buf.Len()))
			index.Put(entry.Key, offsetBytes)
		}
		i++

		// Write key
		keyBytes, err := entry.Key.ToBytes()
		if err != nil {
			return fmt.Errorf("error serializing key: %w", err)
		}
		if _, err := buf.Write(keyBytes); err != nil {
			return fmt.Errorf("error writing key: %w", err)
		}

		// Write value length
		valueLen := int32(len(entry.Value))
		if err := binary.Write(buf, binary.BigEndian, valueLen); err != nil {
			return fmt.Errorf("error serializing value length: %w", err)
		}

		// Write value data
		if _, err := buf.Write(entry.Value); err != nil {
			return fmt.Errorf("error writing value: %w", err)
		}
	}

	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("error writing buffer to file")
	}
	t.tree.Clear()

	return nil
}


func (t *MemTable) Load(buf io.Reader) error {
	tableLength, err := util.ParseInt32(buf)
	if err != nil {
		return fmt.Errorf("error parsing table size: %w", err)
	}

	for i := uint32(0); i < tableLength; i++ {
		key, err := keys.ParseKey(buf)
		if err != nil {
			return err
		}

		valueLength, err := util.ParseInt32(buf)
		if err != nil {
			return fmt.Errorf("error parsing value length: %w", err)
		}

		valueBytes := make([]byte, valueLength)
		if n, err := buf.Read(valueBytes); err != nil || n != int(valueLength) {
			return fmt.Errorf("error parsing value data: %w", err)
		}

		t.Put(key, valueBytes)
	}

	return nil
}
