package keys

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"lsmtree/interfaces"
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

func (i *IntKey) ToBytes() ([]byte, error) {
	/*
	 * this functions outputs 5 bytes, first bye is for key type
	 * and other 4 for the key.
	 */
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
	if n, err := buf.Read(intBytes); err != nil || n != 4 {
		return nil, fmt.Errorf("error parsing int32: %w", err)
	}
	return NewIntKey(binary.BigEndian.Uint32(intBytes)), nil
}

func ParseKey(buf io.Reader) (interfaces.Comparable, error) {
	typeByte := make([]byte, 1)
	if n, err := buf.Read(typeByte); err != nil || n != 1 {
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
	h1 := fnv.New32a()
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, i.value)
	if _, err := h1.Write(buf); err != nil {
		return nil, err
	}
	hash1 := h1.Sum32()

	h2 := fnv.New32()
	if _, err := h2.Write(buf); err != nil {
		return nil, err
	}
	hash2 := h2.Sum32()

	hashes := make([]uint32, numHashes)
	for j := uint32(0); j < numHashes; j++ {
		hashes[j] = hash1 + j*hash2
	}

	return hashes, nil
}
