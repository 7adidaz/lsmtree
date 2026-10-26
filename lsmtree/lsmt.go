package lsmtree

import (
	"os"
	"path/filepath"
	"slices"

	"fmt"
	"main/interfaces"
	"main/memtable"
	"main/record"
)

type LSM struct {
	memtable memtable.MemTable
	SStables []*SSTable
	opts     sstableOptions
	nextID   uint64
	dataPath string
	wal      *WriteAheadLog
}

func NewLSMTree(threshold uint32, sparsityFactor uint32, falsePositiveRate float64) *LSM {
	if sparsityFactor == 0 {
		panic("lsmtree: sparsityFactor must be >= 1 (1 = index every key)")
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		panic("lsmtree: falsePositiveRate must be between 0 and 1 (exclusive)")
	}

	dataPath, err := DataPath()
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		panic(err)
	}

	lsm := &LSM{
		opts: sstableOptions{
			threshold:         threshold,
			sparsityFactor:    sparsityFactor,
			falsePositiveRate: falsePositiveRate,
		},
		memtable: *memtable.NewMemTable(memtable.NewAVLTree()),
		dataPath: dataPath,
	}

	if err := lsm.loadSSTables(dataPath); err != nil {
		panic(err)
	}

	lsm.nextID, err = nextFreeID(dataPath)
	if err != nil {
		panic(err)
	}

	lsm.wal, err = openWAL(filepath.Join(dataPath, walFileName))
	if err != nil {
		panic(err)
	}

	if err := lsm.wal.Replay(&lsm.memtable); err != nil {
		panic(err)
	}

	return lsm
}

func (l *LSM) loadSSTables(dataPath string) error {
	paths, err := sstableFiles(dataPath)
	if err != nil {
		return err
	}

	for _, path := range paths {
		table, err := openSSTable(path, l.opts)
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

	if l.memtable.Size() < l.opts.threshold {
		return nil
	}

	table, err := writeSSTable(l.dataPath, l.nextID, &l.memtable, l.opts)
	if err != nil {
		return err
	}
	l.nextID++
	l.memtable.Clear()

	l.SStables = append(l.SStables, table)

	err = os.Truncate(l.wal.dataLocation, 0)
	if err != nil {
		return err
	}

	return nil
}
