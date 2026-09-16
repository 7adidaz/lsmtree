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
	"main/record"
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

func NewLSMTree(threshold uint32, sparsityFactor uint32, falsePositiveRate float64) *LSM {
	if sparsityFactor == 0 {
		panic("lsmtree: sparsityFactor must be >= 1 (1 = index every key)")
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		panic("lsmtree: falsePositiveRate must be between 0 and 1 (exclusive)")
	}

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
	if found, payload := l.memtable.Get(key); found {
		return resolve(payload)
	}

	for _, table := range slices.Backward(l.SStables) {
		payload, err := table.Find(key)
		if err != nil {
			return false, nil, err
		}
		if payload != nil {
			return resolve(payload)
		}
	}
	return false, nil, nil
}

// resolve is the only place that decodes a payload or reads a record type.
func resolve(payload []byte) (bool, []byte, error) {
	rec, err := record.Parse(payload)
	if err != nil {
		return false, nil, err
	}
	if rec.Type == record.RecordTombstone {
		return false, nil, nil
	}
	return true, rec.Value, nil
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
	rec := &record.Record{Type: record.RecordValue, Value: val}
	return l.put(key, rec.ToBytes())
}

func (l *LSM) Delete(key interfaces.Comparable) error {
	rec := &record.Record{Type: record.RecordTombstone}
	return l.put(key, rec.ToBytes())
}

func (l *LSM) put(key interfaces.Comparable, payload []byte) error {
	if err := l.wal.Append(key, payload); err != nil {
		return err
	}

	l.memtable.Put(key, payload)

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
