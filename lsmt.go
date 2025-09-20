package lsmtree

import (
	"bytes"
	// ...existing code...
	"fmt"
	"lsmtree/bloomfilter"
	"lsmtree/interfaces"
	"lsmtree/keys"
	"lsmtree/memtable"
	"lsmtree/util"
)

type LSM struct {
	memtable          memtable.MemTable
	SStables          []*SSTable
	sparsityFactor    uint32
	threshold         uint32
	falsePositiveRate float64
}

type SSTable struct {
	data         bytes.Buffer
	dataLocation string
	sparseIndex  memtable.MemTableImplementation
	bloomfilter  bloomfilter.BloomFilterImplementation
}

var TOMBSTONE = []byte{0x7f}

func newLSMTree(threshold uint32, sparsityFactor uint32, falsePositiveRate float64) *LSM {
	return &LSM{
		threshold:         threshold,
		sparsityFactor:    sparsityFactor,
		falsePositiveRate: falsePositiveRate,
		memtable:          *memtable.NewMemTable(memtable.NewAVLTree()),
	}
}

func (l LSM) Get(key interfaces.Comparable) (bool, []byte, error) {
	found, val := l.memtable.Get(key)
	if found {
		if bytes.Equal(val, TOMBSTONE) {
			return true, nil, nil
		}
		return true, val, nil
	}
	for i := len(l.SStables) - 1; i >= 0; i-- {
		SSTable := *l.SStables[i]
		found, data, err := SSTable.Find(key)
		if err != nil {
			return false, nil, err
		}
		if found {
			return true, data, nil
		}
	}
	return true, nil, nil
}

func (l *LSM) Put(key interfaces.Comparable, val []byte) error {
	if l.memtable.Size() >= l.threshold {
		buf := new(bytes.Buffer)
		sparseIndex := memtable.NewAVLTree()
		bloomFilter := bloomfilter.NewBloomFilter(l.threshold, l.falsePositiveRate)

		err := l.memtable.Dump(buf, bloomFilter, sparseIndex, int32(l.sparsityFactor))
		if err != nil {
			return err
		}

		// TODO: do compaction and dumping data to the disk.
		l.SStables = append(l.SStables, &SSTable{
			data:         *buf,
			dataLocation: "",
			sparseIndex:  sparseIndex,
			bloomfilter:  bloomFilter,
		})
	}
	l.memtable.Put(key, val)
	return nil
}

func (l *LSM) Delete(key interfaces.Comparable) {
	l.Put(key, TOMBSTONE)
}

func (t *SSTable) Find(key interfaces.Comparable) (bool, []byte, error) {
	found, err := t.bloomfilter.Contains(key)
	if err != nil {
		return false, nil, err
	}
	if !found {
		return false, nil, nil
	}

	lowerBound, _ := util.ParseInt32(bytes.NewReader(t.sparseIndex.Floor(key)))
	upperBound, _ := util.ParseInt32(bytes.NewReader(t.sparseIndex.Ceil(key)))
	if lowerBound == 0 {
		return false, nil, nil
	}
	if lowerBound >= upperBound {
		// this case happens when the element itself if found.
		upperBound = uint32(t.data.Len())
	}
	dataBuf := bytes.NewReader(t.data.Bytes()[lowerBound:upperBound])

	for dataBuf.Len() > 0 {
		parsed_key, err := keys.ParseKey(dataBuf)
		if err != nil {
			return false, nil, err
		}

		valueLength, err := util.ParseInt32(dataBuf)
		if err != nil {
			return false, nil, fmt.Errorf("error parsing value length: %w", err)
		}

		valueBytes := make([]byte, valueLength)
		n, err := dataBuf.Read(valueBytes)
		if err != nil || n != int(valueLength) {
			return false, nil, fmt.Errorf("error parsing value data: %w", err)
		}

		if key.Compare(parsed_key) == 0 {
			if bytes.Equal(valueBytes, TOMBSTONE) {
				return true, nil, nil
			}
			return true, valueBytes, nil
		}
	}
	return false, nil, nil
}

func (t *SSTable) DebugData() error {
	dataCopy := bytes.NewBuffer(t.data.Bytes())

	tableLength, err := util.ParseInt32(dataCopy)
	if err != nil {
		return fmt.Errorf("error parsing table size: %w", err)
	}

	for i := uint32(0); i < tableLength; i++ {
		_, err := keys.ParseKey(dataCopy)
		if err != nil {
			return err
		}

		valueLength, err := util.ParseInt32(dataCopy)
		if err != nil {
			return fmt.Errorf("error parsing value length: %w", err)
		}

		valueBytes := make([]byte, valueLength)
		if n, err := dataCopy.Read(valueBytes); err != nil || n != int(valueLength) {
			return fmt.Errorf("error parsing value data: %w", err)
		}
	}

	return nil
}
