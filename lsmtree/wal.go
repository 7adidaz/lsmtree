package lsmtree

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"

	"main/interfaces"
	"main/memtable"
	"main/record"
)

const walPrefix = "WAL_"

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
	buf := new(bytes.Buffer)
	if err := record.WriteEntry(buf, key, val); err != nil {
		return err
	}
	if err := writeRecord(w.file, buf.Bytes()); err != nil {
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
	if err := os.Remove(w.dataLocation); err != nil {
		return err
	}
	return syncDir(filepath.Dir(w.dataLocation))
}

// Replay puts every record back into mt, oldest first. A torn tail is
// truncated; a checksum mismatch stops the replay.
func (w *WriteAheadLog) Replay(mt *memtable.MemTable) error {
	file, err := os.Open(w.dataLocation)
	if err != nil {
		return err
	}
	defer file.Close()

	r := &countingReader{r: bufio.NewReader(file)}
	for {
		good := r.n
		payload, err := readRecord(r)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return os.Truncate(w.dataLocation, good)
		}
		if err != nil {
			return err
		}

		key, val, err := record.ReadEntry(bytes.NewReader(payload))
		if err != nil {
			return err
		}
		mt.Put(key, val)
	}
}

var crcTable = crc32.MakeTable(crc32.Castagnoli)

var errCorruptWAL = errors.New("lsmtree: wal record crc mismatch")

func writeRecord(w io.Writer, payload []byte) error {
	var header [8]byte
	binary.BigEndian.PutUint32(header[0:4], crc32.Checksum(payload, crcTable))
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readRecord(r io.Reader) ([]byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	want := binary.BigEndian.Uint32(header[0:4])
	length := binary.BigEndian.Uint32(header[4:8])

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	if crc32.Checksum(payload, crcTable) != want {
		return nil, errCorruptWAL
	}
	return payload, nil
}
