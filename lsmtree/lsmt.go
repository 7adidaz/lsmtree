package lsmtree

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"fmt"
	"log"
	"main/interfaces"
	"main/record"
)

const maxFrozen = 4

type LSM struct {
	mu   sync.RWMutex
	opts sstableOptions

	nextID      uint64
	active      *loggedMemtable
	frozen      []*loggedMemtable
	flushCh     chan *loggedMemtable
	flusherDone chan struct{}

	dataPath string
	SStables []*SSTable
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
		mu: sync.RWMutex{},
		opts: sstableOptions{
			threshold:         threshold,
			sparsityFactor:    sparsityFactor,
			falsePositiveRate: falsePositiveRate,
		},

		frozen:      []*loggedMemtable{},
		flushCh:     make(chan *loggedMemtable, maxFrozen),
		flusherDone: make(chan struct{}),

		dataPath: dataPath,
		SStables: []*SSTable{},
	}

	walIDs, err := lsm.loadLoggedMemtables(dataPath)
	if err != nil {
		panic(err)
	}

	if err := lsm.loadSSTables(dataPath, walIDs); err != nil {
		panic(err)
	}
	for _, lm := range slices.Clone(lsm.frozen) {
		if err := lsm.flush(lm); err != nil {
			panic(err)
		}
	}

	lsm.nextID, err = nextFreeID(dataPath)
	if err != nil {
		panic(err)
	}

	lsm.active, err = lsm.nextLoggedMemtable()
	if err != nil {
		panic(err)
	}

	go lsm.runFlusher()
	return lsm
}

func (l *LSM) loadLoggedMemtables(dataPath string) ([]uint64, error) {
	paths, err := walFiles(dataPath)
	if err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(paths))
	for _, path := range paths {
		lm, err := openLoggedMemtable(path)
		if err != nil {
			return nil, err
		}

		if lm.memTable.Size() == 0 {
			if err := lm.wal.Remove(); err != nil {
				return nil, err
			}
			continue
		}

		fmt.Printf("Replayed WAL from %s\n", path)
		l.frozen = append(l.frozen, lm)
		ids = append(ids, lm.id)
	}

	return ids, nil
}

func (l *LSM) loadSSTables(dataPath string, skip []uint64) error {
	paths, err := sstableFiles(dataPath)
	if err != nil {
		return err
	}

	for _, path := range paths {
		id, _ := parseID(filepath.Base(path), sstablePrefix)
		if slices.Contains(skip, id) {
			continue
		}

		table, err := openSSTable(path, id, l.opts)
		if err != nil {
			return err
		}

		fmt.Printf("Loaded SSTable from %s\n", path)
		l.SStables = append(l.SStables, table)
	}

	return nil
}

func (l *LSM) Get(key interfaces.Comparable) (bool, []byte, error) {
	l.mu.RLock()
	if found, payload := l.active.memTable.Get(key); found {
		l.mu.RUnlock()
		return resolve(payload)
	}

	for _, lm := range slices.Backward(l.frozen) {
		if found, payload := lm.memTable.Get(key); found {
			l.mu.RUnlock()
			return resolve(payload)
		}
	}

	SSTables := l.SStables
	l.mu.RUnlock()
	for _, table := range slices.Backward(SSTables) {
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
	l.mu.Lock()
	if err := l.active.wal.Append(key, payload); err != nil {
		l.mu.Unlock()
		return err
	}

	l.active.memTable.Put(key, payload)
	if l.active.memTable.Size() < l.opts.threshold {
		l.mu.Unlock()
		return nil
	}

	frozen, err := l.freeze()
	l.mu.Unlock()
	if err != nil {
		return err
	}

	l.flushCh <- frozen
	return nil
}

func (l *LSM) freeze() (*loggedMemtable, error) {
	next, err := l.nextLoggedMemtable()
	if err != nil {
		return nil, err
	}

	frozen := l.active
	l.frozen = append(l.frozen, frozen)
	l.active = next
	return frozen, nil
}

func (l *LSM) nextLoggedMemtable() (*loggedMemtable, error) {
	id := l.nextID
	l.nextID++
	return newLoggedMemtable(l.dataPath, id)
}

func (l *LSM) runFlusher() {
	defer close(l.flusherDone)
	for lm := range l.flushCh {
		if err := l.flush(lm); err != nil {
			log.Printf("lsmtree: flushing memtable %d: %v", lm.id, err)
		}
	}
}

func (l *LSM) flush(lm *loggedMemtable) error {
	table, err := writeSSTable(l.dataPath, lm.id, lm.memTable, l.opts)
	if err != nil {
		return err
	}

	l.mu.Lock()
	// insert it into the right position to maintain order
	i, _ := slices.BinarySearchFunc(l.SStables, table.id, func(t *SSTable, id uint64) int {
		return cmp.Compare(t.id, id)
	})
	l.SStables = slices.Insert(slices.Clip(l.SStables), i, table)
	l.frozen = slices.DeleteFunc(l.frozen, func(other *loggedMemtable) bool { return other == lm })
	l.mu.Unlock()

	return lm.discard()
}

func (l *LSM) Close() error {
	close(l.flushCh)
	<-l.flusherDone

	l.mu.Lock()
	defer l.mu.Unlock()

	errs := []error{l.active.close()}
	for _, lm := range l.frozen {
		errs = append(errs, lm.close())
	}
	return errors.Join(errs...)
}
