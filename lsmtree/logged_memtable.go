package lsmtree

import (
	"path/filepath"

	"main/memtable"
)

type loggedMemtable struct {
	id       uint64
	wal      *WriteAheadLog
	memTable *memtable.MemTable
}

func newLoggedMemtable(dataPath string, id uint64) (*loggedMemtable, error) {
	wal, err := openWAL(filepath.Join(dataPath, fileName(walPrefix, id)))
	if err != nil {
		return nil, err
	}

	return &loggedMemtable{
		id:       id,
		wal:      wal,
		memTable: memtable.NewMemTable(memtable.NewAVLTree()),
	}, nil
}

func openLoggedMemtable(path string) (*loggedMemtable, error) {
	id, _ := parseID(filepath.Base(path), walPrefix)
	wal, err := openWAL(path)
	if err != nil {
		return nil, err
	}

	lm := &loggedMemtable{
		id:       id,
		wal:      wal,
		memTable: memtable.NewMemTable(memtable.NewAVLTree()),
	}

	if err := lm.wal.Replay(lm.memTable); err != nil {
		wal.Close()
		return nil, err
	}

	return lm, nil
}

func (lm *loggedMemtable) close() error {
	return lm.wal.Close()
}

func (lm *loggedMemtable) discard() error {
	return lm.wal.Remove()
}
