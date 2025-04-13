package interfaces

type Comparable interface {
	// returns 0 if equal
	// 1 if greater
	// -1 if smaller
	Compare(t Comparable) int8
	GetValue() any
}
