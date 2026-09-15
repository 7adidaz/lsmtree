package lsmtree

import (
	"bytes"
	"errors"
	"io"
	"os"

	"main/interfaces"
	"main/memtable"
	"main/record"
)

const walFileName = "WAL"

type WriteAheadLog struct {
	dataLocation string
}

func (w *WriteAheadLog) Append(key interfaces.Comparable, val []byte) error {
	file, err := os.OpenFile(w.dataLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := record.WriteEntry(file, key, val); err != nil {
		return err
	}

	return file.Sync()
}

// Replay puts every record in the log back into mt, oldest first. A log that
// does not exist yet is an empty one.
func (w *WriteAheadLog) Replay(mt *memtable.MemTable) error {
	file, err := os.OpenFile(w.dataLocation, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	file.Close()
	if err != nil {
		return err
	}

	total := buf.Len()
	for {
		remaining := buf.Len()
		key, val, err := record.ReadEntry(buf)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return os.Truncate(w.dataLocation, int64(total-remaining))
		}

		mt.Put(key, val)
	}
}
