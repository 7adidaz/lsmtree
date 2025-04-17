package memtable

import (
	"bytes"
	"encoding/binary"
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
