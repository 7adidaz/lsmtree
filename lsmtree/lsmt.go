package lsmtree

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"fmt"
	"main/bloomfilter"
	"main/interfaces"
	"main/memtable"
)

type LSM struct {
	memtable          memtable.MemTable
	SStables          []*SSTable
	sparsityFactor    uint32
	threshold         uint32
	falsePositiveRate float64
	dataPath          string
	wal               *WriteAheadLog
}

var TOMBSTONE = []byte{0x7f}

func NewLSMTree(threshold uint32, sparsityFactor uint32, falsePositiveRate float64) *LSM {

	cwd, _ := os.Getwd()
	dataPath := filepath.Join(cwd, "data")

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		panic(err)
	}

	lsm := &LSM{
		threshold:         threshold,
		sparsityFactor:    sparsityFactor,
		falsePositiveRate: falsePositiveRate,
		memtable:          *memtable.NewMemTable(memtable.NewAVLTree()),
		dataPath:          dataPath,
		wal:               &WriteAheadLog{filepath.Join(dataPath, walFileName)},
	}

	if err := lsm.loadSSTables(dataPath); err != nil {
		panic(err)
	}

	if err := lsm.wal.Replay(&lsm.memtable); err != nil {
		panic(err)
	}

	return lsm
}

// sstableFiles lists the SSTables in dataPath oldest first, which is the order
// Get depends on: later tables shadow earlier ones.
func sstableFiles(dataPath string) ([]string, error) {
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return nil, err
	}

	type fileEntry struct {
		name    string
		modTime time.Time
	}

	var files []fileEntry
	for _, e := range entries {
		if e.Name() == walFileName {
			continue
		}

		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		files = append(files, fileEntry{name: e.Name(), modTime: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	paths := make([]string, 0, len(files))
	for _, e := range files {
		paths = append(paths, filepath.Join(dataPath, e.name))
	}

	return paths, nil
}

func (l *LSM) loadSSTables(dataPath string) error {
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return err
	}

	paths, err := sstableFiles(dataPath)
	if err != nil {
		return err
	}

	for _, path := range paths {
		table, err := openSSTable(path, l.sparsityFactor, l.threshold, l.falsePositiveRate)
		if err != nil {
			return err
		}

		fmt.Printf("Loaded SSTable from %s\n", path)
		l.SStables = append(l.SStables, table)
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

	tables := l.SStables
	for _, v := range slices.Backward(tables) {
		SSTable := *v
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
	if _, err = f.Write(buf.Bytes()); err != nil {
		fmt.Println("Error writing file:", err)
		return "", err
	}

	if err := f.Sync(); err != nil {
		return "", err
	}

	return fileName, nil
}

func (l *LSM) Put(key interfaces.Comparable, val []byte) error {
	if err := l.wal.Append(key, val); err != nil {
		return err
	}

	l.memtable.Put(key, val)

	if l.memtable.Size() < l.threshold {
		return nil
	}

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
	l.memtable.Clear()

	l.SStables = append(l.SStables, &SSTable{
		dataLocation: fileName,
		dataLength:   buf.Len(),
		sparseIndex:  sparseIndex,
		bloomfilter:  bloomFilter,
	})

	err = os.Truncate(l.wal.dataLocation, 0)
	if err != nil {
		return err
	}

	return nil
}

func (l *LSM) Delete(key interfaces.Comparable) {
	l.Put(key, TOMBSTONE)
}
