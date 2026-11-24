package keys

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"main/interfaces"
)

type IntKey struct {
	value uint32
}

func NewIntKey(k uint32) *IntKey {
	return &IntKey{value: k}
}

func (i *IntKey) Compare(other interfaces.Comparable) int8 {
	otherKey, ok := other.(*IntKey)
	if !ok {
		panic("Cannot compare IntKey with a different type")
	}

	if i.value < otherKey.value {
		return -1
	} else if i.value > otherKey.value {
		return 1
	}
	return 0
}

func (i *IntKey) GetValue() any {
	return i.value
}

// ToBytes encodes 1 type byte (0x00) followed by 4 big-endian value bytes.
func (i *IntKey) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, uint8(0x00)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, i.value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func IntKeyFromBytes(buf io.Reader) (interfaces.Comparable, error) {
	intBytes := make([]byte, 4)
	if _, err := io.ReadFull(buf, intBytes); err != nil {
		return nil, fmt.Errorf("error parsing int32: %w", err)
	}
	return NewIntKey(binary.BigEndian.Uint32(intBytes)), nil
}

func ParseKey(buf io.Reader) (interfaces.Comparable, error) {
	typeByte := make([]byte, 1)
	if _, err := io.ReadFull(buf, typeByte); err != nil {
		return nil, fmt.Errorf("error parsing key: %w", err)
	}

	switch typeByte[0] {
	case uint8(0x00): // IntKey
		key, err := IntKeyFromBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("error parsing key: %w", err)
		}
		return key, nil
	case uint8(0x01): // StringKey
		key, err := StringKeyFromBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("error parsing key: %w", err)
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unknown type: %d", typeByte[0])
	}
}

func (i *IntKey) Hash(numHashes uint32) ([]uint32, error) {
	h := fnv.New64a()
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, i.value)
	if _, err := h.Write(buf); err != nil {
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
