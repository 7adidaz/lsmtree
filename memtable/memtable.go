package memtable

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"main/bloomfilter"
	"main/interfaces"
	"main/record"
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

func (t *MemTable) Dump(file io.Writer,
	bloomfilter bloomfilter.BloomFilterImplementation,
	index MemTableImplementation,
	sampling int32) error {
	/*
		     * Binary Format:
		     * For each entry:
		     *   [1 byte]  - Key type (0x00 for IntKey)
		     *   [4 bytes] - Key value (uint32)
		     *   [4 bytes] - Value length (int32)
		     *   [N bytes] - Value data
			 *
			 *  the caller should close the file.
	*/

	buf := new(bytes.Buffer)

	i := int32(0)
	for _, entry := range t.tree.ToKVs() {
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

		if err := record.WriteEntry(buf, entry.Key, entry.Value); err != nil {
			return err
		}
	}

	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("error writing buffer to file")
	}

	return nil
}

type offsetReader struct {
	r      io.Reader
	offset int
}

func (o *offsetReader) Read(p []byte) (int, error) {
	n, err := o.r.Read(p)
	o.offset += n
	return n, err
}

func (t *MemTable) Load(buf io.Reader,
	bloomfilter bloomfilter.BloomFilterImplementation,
	index MemTableImplementation,
	sampling uint32) error {

	rd := &offsetReader{r: buf}

	j := uint32(0)
	startOfCurrKey := 0
	for {
		startOfCurrKey = rd.offset
		key, valueBytes, err := record.ReadEntry(rd)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		if bloomfilter != nil {
			bloomfilter.Insert(key)
		}
		// sparse index
		if index != nil && sampling > 0 && j%sampling == 0 {
			offsetBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(offsetBytes, uint32(startOfCurrKey))
			index.Put(key, offsetBytes)
		}
		j++

		t.Put(key, valueBytes)
	}

	return nil
}

func (t *MemTable) Clear() {
	t.tree.Clear()
}
