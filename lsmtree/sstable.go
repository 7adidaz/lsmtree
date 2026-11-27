package lsmtree

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"

	"main/bloomfilter"
	"main/interfaces"
	"main/memtable"
	"main/record"
)

type SSTable struct {
	id           uint64
	dataLocation string
	dataLength   int
	sparseIndex  memtable.MemTableImplementation
	bloomfilter  bloomfilter.BloomFilterImplementation
}

type sstableOptions struct {
	maxMemtableBytes  uint32
	sparsityFactor    uint32
	falsePositiveRate float64
}

type blockHandle struct {
	// key is build-time only; not serialized.
	key   interfaces.Comparable
	start int
	size  int
}

func (h *blockHandle) toBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b[0:4], uint32(h.start))
	binary.BigEndian.PutUint32(b[4:8], uint32(h.size))
	return b
}

func blockHandleFromBytes(b []byte) blockHandle {
	return blockHandle{
		start: int(binary.BigEndian.Uint32(b[0:4])),
		size:  int(binary.BigEndian.Uint32(b[4:8])),
	}
}

// openSSTable streams an SSTable file once, rebuilding the bloom filter and
// sparse index that were built when the file was written.
func openSSTable(path string, id uint64, opts sstableOptions) (*SSTable, error) {
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
	bloomFilter, err := scanSSTable(bufio.NewReader(f), sparseIndex, opts.sparsityFactor, opts.falsePositiveRate)
	if err != nil {
		return nil, err
	}

	return &SSTable{
		id:           id,
		dataLocation: path,
		dataLength:   int(info.Size()),
		sparseIndex:  sparseIndex,
		bloomfilter:  bloomFilter,
	}, nil
}

func writeSSTable(dataPath string, id uint64, mt *memtable.MemTable, opts sstableOptions) (*SSTable, error) {
	buf := new(bytes.Buffer)
	sparseIndex := memtable.NewAVLTree()
	entries := mt.ToKVs()
	bloomFilter := bloomfilter.NewBloomFilter(uint32(len(entries)), opts.falsePositiveRate)

	if err := encodeSSTable(buf, entries, bloomFilter, sparseIndex, opts.sparsityFactor); err != nil {
		return nil, err
	}

	path := filepath.Join(dataPath, fileName(sstablePrefix, id))
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err = f.Write(buf.Bytes()); err != nil {
		return nil, err
	}

	if err := f.Sync(); err != nil {
		return nil, err
	}
	if err := syncDir(dataPath); err != nil {
		return nil, err
	}

	return &SSTable{
		id:           id,
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
// Every key goes into bloomFilter, and every sampling-th key starts a block,
// indexed by its handle. Either may be nil.
func encodeSSTable(w io.Writer, entries []*memtable.Entry,
	bloomFilter bloomfilter.BloomFilterImplementation,
	index memtable.MemTableImplementation,
	sampling uint32) error {

	buf := new(bytes.Buffer)
	var blocks []blockHandle
	for i, entry := range entries {
		if bloomFilter != nil {
			bloomFilter.Insert(entry.Key)
		}
		if index != nil && sampling > 0 && uint32(i)%sampling == 0 {
			blocks = append(blocks, blockHandle{key: entry.Key, start: buf.Len()})
		}

		if err := record.WriteEntry(buf, entry.Key, entry.Value); err != nil {
			return err
		}
	}
	indexBlocks(index, blocks, buf.Len())

	_, err := w.Write(buf.Bytes())
	return err
}

// indexBlocks fills in each block's length (known once the next block starts)
// and puts the finished handles into the index.
func indexBlocks(index memtable.MemTableImplementation, blocks []blockHandle, total int) {
	for i := range blocks {
		end := total
		if i+1 < len(blocks) {
			end = blocks[i+1].start
		}
		blocks[i].size = end - blocks[i].start
		index.Put(blocks[i].key, blocks[i].toBytes())
	}
}

// scanSSTable fills index exactly as encodeSSTable did and returns a bloom
// filter over every key, sized from the real entry count after the scan.
// TODO: building the filter here is ugly; goes away once it lives in the file.
func scanSSTable(r io.Reader,
	index memtable.MemTableImplementation,
	sampling uint32,
	falsePositiveRate float64) (bloomfilter.BloomFilterImplementation, error) {

	var tableKeys []interfaces.Comparable
	var blocks []blockHandle
	cr := &countingReader{r: r}
	for i := uint32(0); ; i++ {
		start := cr.n
		key, _, err := record.ReadEntry(cr)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		tableKeys = append(tableKeys, key)
		if index != nil && sampling > 0 && i%sampling == 0 {
			blocks = append(blocks, blockHandle{key: key, start: int(start)})
		}
	}
	indexBlocks(index, blocks, int(cr.n))

	bloomFilter := bloomfilter.NewBloomFilter(uint32(len(tableKeys)), falsePositiveRate)
	for _, key := range tableKeys {
		bloomFilter.Insert(key)
	}
	return bloomFilter, nil
}

func (t *SSTable) readBlock(h blockHandle) (*bytes.Reader, error) {
	buffer := make([]byte, h.size)
	f, err := os.Open(t.dataLocation)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.ReadAt(buffer, int64(h.start)); err != nil {
		return nil, err
	}

	return bytes.NewReader(buffer), nil
}

// Find returns the stored payload, or nil if this table has nothing to say
// about the key and the caller should look in older tables. A tombstone is an
// answer, so it comes back as a payload; decoding it is LSM.Get's job.
func (t *SSTable) Find(key interfaces.Comparable) ([]byte, error) {
	if !t.bloomfilter.Contains(key) {
		return nil, nil
	}

	// the smallest key in the table is always sampled, so a missing floor means
	// the key sorts before every key in this SSTable.
	floorBytes := t.sparseIndex.Floor(key)
	if floorBytes == nil {
		return nil, nil
	}

	dataBuf, err := t.readBlock(blockHandleFromBytes(floorBytes))
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

		switch key.Compare(parsedKey) {
		case 0:
			return payload, nil
		case -1:
			return nil, nil
		}
	}
	return nil, nil
}

