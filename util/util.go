package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
