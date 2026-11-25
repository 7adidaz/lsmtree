package keys

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/fnv"
	"io"
	"main/interfaces"
)

type StringKey struct {
	value string
}

func NewStringKey(s string) *StringKey {
	return &StringKey{value: s}
}

func (s *StringKey) Compare(other interfaces.Comparable) int8 {
	otherKey, ok := other.(*StringKey)
	if !ok {
		panic("Cannot compare StringKey with a different type")
	}
	if s.value < otherKey.value {
		return -1
	} else if s.value > otherKey.value {
		return 1
	}
	return 0
}

func (s *StringKey) GetValue() any {
	return s.value
}

func (s *StringKey) Size() int {
	return 5 + len(s.value)
}

// ToBytes encodes 1 type byte (0x01), 4 big-endian length bytes, then the
// string bytes.
func (s *StringKey) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint8(0x01)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(s.value))); err != nil {
		return nil, err
	}
	if _, err := buf.Write([]byte(s.value)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func StringKeyFromBytes(buf io.Reader) (interfaces.Comparable, error) {
	lenBytes := make([]byte, 4)
	if _, err := io.ReadFull(buf, lenBytes); err != nil {
		return nil, errors.New("failed to read length bytes")
	}
	length := binary.BigEndian.Uint32(lenBytes)
	strBytes := make([]byte, length)
	if _, err := io.ReadFull(buf, strBytes); err != nil {
		return nil, errors.New("failed to read string bytes")
	}
	return NewStringKey(string(strBytes)), nil
}

func (s *StringKey) Hash(numHashes uint32) ([]uint32, error) {
	h := fnv.New64a()
	if _, err := h.Write([]byte(s.value)); err != nil {
		return nil, err
	}
	sum := h.Sum64()
	hash1 := uint32(sum >> 32)
	hash2 := uint32(sum) | 1 // stride must be odd: 0 collapses all probes, even strides clash with even sizes

	hashes := make([]uint32, numHashes)
	for j := range numHashes {
		hashes[j] = hash1 + j*hash2
	}
	return hashes, nil
}
