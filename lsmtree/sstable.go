package lsmtree

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

type sstableOptions struct {
	threshold         uint32
	sparsityFactor    uint32
	falsePositiveRate float64
}

// openSSTable streams an SSTable file once, rebuilding the bloom filter and
// sparse index that were built when the file was written.
func openSSTable(path string, opts sstableOptions) (*SSTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	sparseIndex := memtable.NewAVLTree()
	bloomFilter := bloomfilter.NewBloomFilter(opts.threshold, opts.falsePositiveRate)

	if err := scanSSTable(bufio.NewReader(f), bloomFilter, sparseIndex, opts.sparsityFactor); err != nil {
		return nil, err
	}

	return &SSTable{
		dataLocation: path,
		dataLength:   int(info.Size()),
		sparseIndex:  sparseIndex,
		bloomfilter:  bloomFilter,
	}, nil
}

func writeSSTable(dataPath string, id uint64, mt *memtable.MemTable, opts sstableOptions) (*SSTable, error) {
	buf := new(bytes.Buffer)
	sparseIndex := memtable.NewAVLTree()
	bloomFilter := bloomfilter.NewBloomFilter(opts.threshold, opts.falsePositiveRate)

	if err := encodeSSTable(buf, mt.ToKVs(), bloomFilter, sparseIndex, opts.sparsityFactor); err != nil {
		return nil, err
	}

	path := filepath.Join(dataPath, fileName(sstablePrefix, id))
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return nil, err
	}
	defer f.Close()

	fmt.Println("Writing SSTable to:", path)
	if _, err = f.Write(buf.Bytes()); err != nil {
		fmt.Println("Error writing file:", err)
		return nil, err
	}

	if err := f.Sync(); err != nil {
		return nil, err
	}

	return &SSTable{
		dataLocation: path,
		dataLength:   buf.Len(),
		sparseIndex:  sparseIndex,
		bloomfilter:  bloomFilter,
	}, nil
}

// encodeSSTable writes entries, which must be sorted by key, to w:
//
//	For each entry:
//	  [1 byte]  - Key type (0x00 IntKey, 0x01 StringKey)
//	  [N bytes] - Key data
//	  [4 bytes] - Payload length
//	  [N bytes] - Payload
//
// Every key goes into bloomFilter, and every sampling-th key goes into index
// with its byte offset. Either may be nil.
func encodeSSTable(w io.Writer, entries []*memtable.Entry,
	bloomFilter bloomfilter.BloomFilterImplementation,
	index memtable.MemTableImplementation,
	sampling uint32) error {

	buf := new(bytes.Buffer)
	for i, entry := range entries {
		if bloomFilter != nil {
			bloomFilter.Insert(entry.Key)
		}
		if index != nil && sampling > 0 && uint32(i)%sampling == 0 {
			index.Put(entry.Key, offsetBytes(buf.Len()))
		}

		if err := record.WriteEntry(buf, entry.Key, entry.Value); err != nil {
			return err
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// scanSSTable reads an SSTable written by encodeSSTable and fills bloomFilter and
// index exactly as encodeSSTable did. The records themselves are not kept.
func scanSSTable(r io.Reader,
	bloomFilter bloomfilter.BloomFilterImplementation,
	index memtable.MemTableImplementation,
	sampling uint32) error {

	cr := &countingReader{r: r}
	for i := uint32(0); ; i++ {
		start := cr.n
		key, _, err := record.ReadEntry(cr)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		if bloomFilter != nil {
			bloomFilter.Insert(key)
		}
		if index != nil && sampling > 0 && i%sampling == 0 {
			index.Put(key, offsetBytes(int(start)))
		}
	}
}

func offsetBytes(offset int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(offset))
	return b
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

// Find returns the stored payload, or nil if this table has nothing to say
// about the key and the caller should look in older tables. A tombstone is an
// answer, so it comes back as a payload; decoding it is LSM.Get's job.
func (t *SSTable) Find(key interfaces.Comparable) ([]byte, error) {
	found, err := t.bloomfilter.Contains(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	// the smallest key in the table is always sampled, so a missing floor means
	// the key sorts before every key in this SSTable.
	floorBytes := t.sparseIndex.Floor(key)
	if len(floorBytes) != 4 {
		return nil, nil
	}
	lowerBound, err := record.ParseInt32(bytes.NewReader(floorBytes))
	if err != nil {
		return nil, err
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
		return nil, err
	}

	for {
		parsedKey, payload, err := record.ReadEntry(dataBuf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if key.Compare(parsedKey) == 0 {
			return payload, nil
		}
	}
	return nil, nil
}
