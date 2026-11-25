package lsmtree

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"fmt"
	"log"
	"main/interfaces"
	"main/record"
)

const (
	maxFrozen       = 4
	maxFlushRetries = 5
)

type LSM struct {
	mu   sync.RWMutex
	opts sstableOptions

	nextID      uint64
	active      *loggedMemtable
	frozen      []*loggedMemtable
	flushCh     chan *loggedMemtable
	flusherDone chan struct{}
	flushErr    error

	dataPath string
	sstables []*SSTable
}

func NewLSMTree(dataPath string, maxMemtableBytes uint32, sparsityFactor uint32, falsePositiveRate float64) (*LSM, error) {
	if sparsityFactor == 0 {
		return nil, errors.New("lsmtree: sparsityFactor must be >= 1 (1 = index every key)")
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		return nil, errors.New("lsmtree: falsePositiveRate must be between 0 and 1 (exclusive)")
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return nil, err
	}

	lsm := &LSM{
		mu: sync.RWMutex{},
		opts: sstableOptions{
			maxMemtableBytes:  maxMemtableBytes,
			sparsityFactor:    sparsityFactor,
			falsePositiveRate: falsePositiveRate,
		},

		frozen:      []*loggedMemtable{},
		flushCh:     make(chan *loggedMemtable, maxFrozen),
		flusherDone: make(chan struct{}),
		flushErr:    nil,

		dataPath: dataPath,
		sstables: []*SSTable{},
	}

	if err := lsm.recover(); err != nil {
		lsm.closeWALs()
		return nil, err
	}

	go lsm.runFlusher()
	return lsm, nil
}

func (l *LSM) recover() error {
	walIDs, err := l.loadLoggedMemtables(l.dataPath)
	if err != nil {
		return err
	}

	if err := l.loadSSTables(l.dataPath, walIDs); err != nil {
		return err
	}
	for _, lm := range slices.Clone(l.frozen) {
		if err := l.flush(lm); err != nil {
			return err
		}
	}

	l.nextID, err = nextFreeID(l.dataPath)
	if err != nil {
		return err
	}

	l.active, err = l.nextLoggedMemtable()
	return err
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
		l.sstables = append(l.sstables, table)
	}

	return nil
}

// tables snapshots the SSTable list; valid after unlock because
// flush never mutates the slice in place.
func (l *LSM) tables() []*SSTable {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.sstables
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

	sstables := l.sstables
	l.mu.RUnlock()
	for _, table := range slices.Backward(sstables) {
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

	if l.flushErr != nil {
		l.mu.Unlock()
		return l.flushErr
	}

	if err := l.active.wal.Append(key, payload); err != nil {
		l.mu.Unlock()
		return err
	}

	l.active.memTable.Put(key, payload)
	if l.active.memTable.Size() < l.opts.maxMemtableBytes {
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
		attempt, delay := 0, 100*time.Millisecond
		for {
			err := l.flush(lm)
			if err == nil {
				break
			}

			log.Printf("lsmtree: flushing memtable %d (attempt %d): %v", lm.id, attempt+1, err)
			if attempt == maxFlushRetries {
				l.mu.Lock()
				l.flushErr = fmt.Errorf("lsmtree: flushing memtable %d: %w", lm.id, err)
				l.mu.Unlock()
				break
			}

			time.Sleep(delay)
			attempt, delay = attempt+1, delay*2
		}
	}
}

func (l *LSM) flush(lm *loggedMemtable) error {
	table, err := writeSSTable(l.dataPath, lm.id, lm.memTable, l.opts)
	if err != nil {
		return err
	}

	l.mu.Lock()
	// Insert by ID (flushes finish out of order); Clip keeps readers' snapshots intact.
	i, _ := slices.BinarySearchFunc(l.sstables, table.id, func(t *SSTable, id uint64) int {
		return cmp.Compare(t.id, id)
	})
	l.sstables = slices.Insert(slices.Clip(l.sstables), i, table)
	l.frozen = slices.DeleteFunc(l.frozen, func(other *loggedMemtable) bool { return other == lm })
	l.mu.Unlock()

	return lm.discard()
}

func (l *LSM) Close() error {
	close(l.flushCh)
	<-l.flusherDone

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.closeWALs()
}

func (l *LSM) closeWALs() error {
	var errs []error
	if l.active != nil {
		errs = append(errs, l.active.close())
	}
	for _, lm := range l.frozen {
		errs = append(errs, lm.close())
	}
	return errors.Join(errs...)
}
