package util

import (
	"bytes"
	"encoding/binary"
)

func ToByteArray(data any) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, uint8(0x00))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
