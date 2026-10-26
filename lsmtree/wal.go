package lsmtree

import (
	"bufio"
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
	file         *os.File
}

func openWAL(path string) (*WriteAheadLog, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &WriteAheadLog{dataLocation: path, file: file}, nil
}

func (w *WriteAheadLog) Append(key interfaces.Comparable, val []byte) error {
	if err := record.WriteEntry(w.file, key, val); err != nil {
		return err
	}

	return w.file.Sync()
}

func (w *WriteAheadLog) Close() error {
	return w.file.Close()
}

func (w *WriteAheadLog) Remove() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	return os.Remove(w.dataLocation)
}

// Replay puts every record in the log back into mt, oldest first, and truncates
// a torn record at the tail.
func (w *WriteAheadLog) Replay(mt *memtable.MemTable) error {
	file, err := os.Open(w.dataLocation)
	if err != nil {
		return err
	}
	defer file.Close()

	r := &countingReader{r: bufio.NewReader(file)}
	for {
		good := r.n
		key, val, err := record.ReadEntry(r)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return os.Truncate(w.dataLocation, good)
		}

		mt.Put(key, val)
	}
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
