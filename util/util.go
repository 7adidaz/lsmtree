package util

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"main/interfaces"
	"main/keys"
)

func ToByteArray(data any) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseInt32(buf io.Reader) (uint32, error) {
	intBytes := make([]byte, 4)
	if n, err := buf.Read(intBytes); err != nil || n != 4 {
		return 0, fmt.Errorf("error parsing int32: %w", err)
	}
	return binary.BigEndian.Uint32(intBytes), nil
}

// WriteEntry appends one key/value record in the SSTable layout:
//
//	[1 byte]  - key type tag
//	[4+ bytes]- key data (uint32, or length + bytes for strings)
//	[4 bytes] - value length (int32)
//	[N bytes] - value data
func WriteEntry(w io.Writer, key interfaces.Comparable, value []byte) error {
	keyBytes, err := key.ToBytes()
	if err != nil {
		return fmt.Errorf("error serializing key: %w", err)
	}
	if _, err := w.Write(keyBytes); err != nil {
		return fmt.Errorf("error writing key: %w", err)
	}

	if err := binary.Write(w, binary.BigEndian, int32(len(value))); err != nil {
		return fmt.Errorf("error serializing value length: %w", err)
	}

	if _, err := w.Write(value); err != nil {
		return fmt.Errorf("error writing value: %w", err)
	}

	return nil
}

// ReadEntry reads back one record written by WriteEntry. It returns a bare
// io.EOF at a clean record boundary, and a wrapped error for an EOF part-way
// through a record.
func ReadEntry(r io.Reader) (interfaces.Comparable, []byte, error) {
	tag := make([]byte, 1)
	if _, err := io.ReadFull(r, tag); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil, io.EOF
		}
		return nil, nil, fmt.Errorf("error reading key tag: %w", err)
	}

	// ParseKey expects to read the tag itself, so hand it back.
	key, err := keys.ParseKey(io.MultiReader(bytes.NewReader(tag), r))
	if err != nil {
		return nil, nil, err
	}

	valueLength, err := ParseInt32(r)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing value length: %w", err)
	}

	value := make([]byte, valueLength)
	if _, err := io.ReadFull(r, value); err != nil {
		return nil, nil, fmt.Errorf("error parsing value data: %w", err)
	}

	return key, value, nil
}
