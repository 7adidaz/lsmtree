package interfaces

type Comparable interface {
	// returns 0 if equal
	// 1 if greater
	// -1 if smaller
	Compare(t Comparable) int8
	GetValue() any
	// first byte is numbering for different supported keys which i will document later
	// for now i will only support uint32 which will be 0x00 in hex.
	ToBytes() ([]byte, error)
}
