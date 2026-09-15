package lsmtree

import (
	"bytes"
	"errors"
	"io"
	"os"

	"main/bloomfilter"
	"main/interfaces"
	"main/memtable"
	"main/record"
)

type SSTable struct {
	dataLocation string
	dataLength   int
	sparseIndex  memtable.MemTableImplementation
	bloomfilter  bloomfilter.BloomFilterImplementation
}

// openSSTable reads an SSTable file back into memory, rebuilding the bloom
// filter and sparse index that Dump produced when the file was written.
func openSSTable(path string, sparsityFactor uint32, threshold uint32, falsePositiveRate float64) (*SSTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(f)
	f.Close()
	if err != nil {
		return nil, err
	}

	dataLength := buf.Len()

	sparseIndex := memtable.NewAVLTree()
	bloomFilter := bloomfilter.NewBloomFilter(threshold, falsePositiveRate)

	// the memtable is a by-product: Load drives the parse, and what we keep is
	// the filter and the index it fills in along the way.
	loaded := memtable.NewMemTable(memtable.NewAVLTree())
	if err := loaded.Load(buf, bloomFilter, sparseIndex, sparsityFactor); err != nil {
		return nil, err
	}

	return &SSTable{
		dataLocation: path,
		dataLength:   dataLength,
		sparseIndex:  sparseIndex,
		bloomfilter:  bloomFilter,
	}, nil
}

func (t *SSTable) readSSTableData(lowerBound uint32, upperBound uint32) (*bytes.Reader, error) {
	buffer := make([]byte, upperBound-lowerBound)
	f, err := os.Open(t.dataLocation)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.Seek(int64(lowerBound), io.SeekStart)
	if err != nil {
		return nil, err
	}

	_, err = f.Read(buffer)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buffer), nil
}

func (t *SSTable) Find(key interfaces.Comparable) (bool, []byte, error) {
	found, err := t.bloomfilter.Contains(key)
	if err != nil {
		return false, nil, err
	}
	if !found {
		return false, nil, nil
	}

	// the smallest key in the table is always sampled, so a missing floor means
	// the key sorts before every key in this SSTable.
	floorBytes := t.sparseIndex.Floor(key)
	if len(floorBytes) != 4 {
		return false, nil, nil
	}
	lowerBound, err := record.ParseInt32(bytes.NewReader(floorBytes))
	if err != nil {
		return false, nil, err
	}

	// no ceiling, or the key is itself a sampled offset: read to the end.
	upperBound := uint32(t.dataLength)
	if ceilBytes := t.sparseIndex.Ceil(key); len(ceilBytes) == 4 {
		if bound, err := record.ParseInt32(bytes.NewReader(ceilBytes)); err == nil && bound > lowerBound {
			upperBound = bound
		}
	}

	dataBuf, err := t.readSSTableData(lowerBound, upperBound)
	if err != nil {
		return false, nil, err
	}

	for {
		parsedKey, valueBytes, err := record.ReadEntry(dataBuf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false, nil, err
		}

		if key.Compare(parsedKey) == 0 {
			if bytes.Equal(valueBytes, TOMBSTONE) {
				return true, nil, nil
			}
			return true, valueBytes, nil
		}
	}
	return false, nil, nil
}
