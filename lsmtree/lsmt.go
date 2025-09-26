package lsmtree

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"

	"fmt"
	"main/bloomfilter"
	"main/interfaces"
	"main/keys"
	"main/memtable"
	"main/util"
)

type LSM struct {
	memtable          memtable.MemTable
	SStables          []*SSTable
	sparsityFactor    uint32
	threshold         uint32
	falsePositiveRate float64
	dataPath          string
}

type SSTable struct {
	// data         bytes.Buffer
	dataLocation string
	dataLength   int
	sparseIndex  memtable.MemTableImplementation
	bloomfilter  bloomfilter.BloomFilterImplementation
}

var TOMBSTONE = []byte{0x7f}

func NewLSMTree(threshold uint32, sparsityFactor uint32, falsePositiveRate float64) *LSM {

	cwd, _ := os.Getwd()
	dataPath := filepath.Join(cwd, "data")
	loadSSTables(dataPath)

	return &LSM{
		threshold:         threshold,
		sparsityFactor:    sparsityFactor,
		falsePositiveRate: falsePositiveRate,
		memtable:          *memtable.NewMemTable(memtable.NewAVLTree()),
		dataPath:          dataPath,
	}
}

func loadSSTables(dataPath string) error {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return err
	}

	for _, e := range entries {
		// construct sparsity and bloom filter
		fmt.Println(e.Name())
	}

	return nil
}

func (l *LSM) Get(key interfaces.Comparable) (bool, []byte, error) {
	found, val := l.memtable.Get(key)
	if found {
		if bytes.Equal(val, TOMBSTONE) {
			return false, nil, nil
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
	return false, nil, nil
}

func (l *LSM) writeSSTableData(buf bytes.Buffer) (string, error) {
	fileName := filepath.Join(l.dataPath, fmt.Sprintf("sstable_%d", time.Now().UnixNano()))
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return "", err
	}
	defer f.Close()

	fmt.Println("Writing SSTable to:", fileName)
	_, err = f.Write(buf.Bytes())
	if err != nil {
		fmt.Println("Error creating file:", err)
		return "", err
	}

	return fileName, nil
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

		fileName, err := l.writeSSTableData(*buf)
		if err != nil {
			return err
		}

		l.SStables = append(l.SStables, &SSTable{
			dataLocation: fileName,
			dataLength:   buf.Len(),
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

	lowerBound, _ := util.ParseInt32(bytes.NewReader(t.sparseIndex.Floor(key)))
	upperBound, _ := util.ParseInt32(bytes.NewReader(t.sparseIndex.Ceil(key)))
	if lowerBound == 0 {
		return false, nil, nil
	}
	// this case happens when the element itself if found.
	if lowerBound >= upperBound {
		upperBound = uint32(t.dataLength)
	}

	dataBuf, err := t.readSSTableData(lowerBound, upperBound)
	if err != nil {
		return false, nil, err
	}

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
