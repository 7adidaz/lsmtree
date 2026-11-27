package keys

import (
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

func (s *StringKey) Size() int {
	return 5 + len(s.value)
}

// ToBytes encodes 1 type byte (0x01), 4 big-endian length bytes, then the
// string bytes.
func (s *StringKey) ToBytes() []byte {
	b := make([]byte, 5+len(s.value))
	b[0] = 0x01
	binary.BigEndian.PutUint32(b[1:5], uint32(len(s.value)))
	copy(b[5:], s.value)
	return b
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

func (s *StringKey) Hash(numHashes uint32) []uint32 {
	h := fnv.New64a()
	h.Write([]byte(s.value))
	sum := h.Sum64()
	hash1 := uint32(sum >> 32)
	hash2 := uint32(sum) | 1

	hashes := make([]uint32, numHashes)
	for j := range numHashes {
		hashes[j] = hash1 + j*hash2
	}
	return hashes
}
