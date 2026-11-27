package interfaces

type Comparable interface {
	// Compare returns 0 if equal, 1 if the receiver is greater, -1 if smaller.
	Compare(t Comparable) int8
	// ToBytes encodes the key prefixed with its type byte
	// (0x00 IntKey, 0x01 StringKey); keys.ParseKey decodes it.
	ToBytes() []byte
	// Size returns the length of the ToBytes encoding without allocating it.
	Size() int
	Hash(numHashes uint32) []uint32
}
