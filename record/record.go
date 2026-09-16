package record

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"main/interfaces"
	"main/keys"
)

type RecordType uint8

const (
	RecordValue     RecordType = 0x00
	RecordTombstone RecordType = 0x01
)

type Record struct {
	Type  RecordType
	Value []byte
}

func (r *Record) ToBytes() []byte {
	payload := make([]byte, 0, len(r.Value)+1)
	payload = append(payload, byte(r.Type))
	return append(payload, r.Value...)
}

func Parse(payload []byte) (*Record, error) {
	if len(payload) == 0 {
		return nil, errors.New("record: empty payload")
	}

	recType := RecordType(payload[0])
	switch recType {
	case RecordValue, RecordTombstone:
	default:
		return nil, fmt.Errorf("record: unknown record type 0x%02x", payload[0])
	}

	return &Record{Type: recType, Value: payload[1:]}, nil
}

func ParseInt32(buf io.Reader) (uint32, error) {
	intBytes := make([]byte, 4)
	if n, err := buf.Read(intBytes); err != nil || n != 4 {
		return 0, fmt.Errorf("error parsing int32: %w", err)
	}
	return binary.BigEndian.Uint32(intBytes), nil
}

// WriteEntry appends one key/payload record in the SSTable layout:
//
//	[1 byte]  - key type tag
//	[4+ bytes]- key data (uint32, or length + bytes for strings)
//	[4 bytes] - payload length (int32)
//	[N bytes] - payload (a serialized Record; see Record.ToBytes)
func WriteEntry(w io.Writer, key interfaces.Comparable, payload []byte) error {
	keyBytes, err := key.ToBytes()
	if err != nil {
		return fmt.Errorf("error serializing key: %w", err)
	}
	if _, err := w.Write(keyBytes); err != nil {
		return fmt.Errorf("error writing key: %w", err)
	}

	if err := binary.Write(w, binary.BigEndian, int32(len(payload))); err != nil {
		return fmt.Errorf("error serializing payload length: %w", err)
	}

	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("error writing payload: %w", err)
	}

	return nil
}

// ReadEntry reads back one record written by WriteEntry. It returns a bare
// io.EOF at a clean record boundary, and a wrapped error for an EOF part-way
// through a record. The payload is returned undecoded.
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

	payloadLength, err := ParseInt32(r)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing payload length: %w", err)
	}

	payload := make([]byte, payloadLength)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, nil, fmt.Errorf("error parsing payload data: %w", err)
	}

	return key, payload, nil
}
