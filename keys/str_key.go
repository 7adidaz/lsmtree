package keys

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/fnv"
	"io"
	"lsmtree/interfaces"
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

func (s *StringKey) ToBytes() ([]byte, error) {
	// 0x01 for string key type, then 4 bytes for length, then string bytes
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
	// 4 bytes for length
	lenBytes := make([]byte, 4)
	if n, err := buf.Read(lenBytes); err != nil || n != 4 {
		return nil, errors.New("failed to read length bytes")
	}
	length := binary.BigEndian.Uint32(lenBytes)
	// string bytes
	strBytes := make([]byte, length)
	if n, err := buf.Read(strBytes); err != nil || uint32(n) != length {
		return nil, errors.New("failed to read string bytes")
	}
	return NewStringKey(string(strBytes)), nil
}

func (s *StringKey) Hash(numHashes uint32) ([]uint32, error) {
	h1 := fnv.New32a()
	if _, err := h1.Write([]byte(s.value)); err != nil {
		return nil, err
	}
	hash1 := h1.Sum32()

	h2 := fnv.New32()
	if _, err := h2.Write([]byte(s.value)); err != nil {
		return nil, err
	}
	hash2 := h2.Sum32()

	hashes := make([]uint32, numHashes)
	for j := uint32(0); j < numHashes; j++ {
		hashes[j] = hash1 + j*hash2
	}
	return hashes, nil
}
