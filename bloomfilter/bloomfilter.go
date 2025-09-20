package bloomfilter

import (
	"lsmtree/interfaces"
	"math"
)

type BloomFilter struct {
	size      uint32
	buckets   []uint64
	numHashes uint32
}

type BloomFilterImplementation interface {
	Insert(key interfaces.Comparable) error
	Contains(key interfaces.Comparable) (bool, error)
}

func NewBloomFilter(expectedItems uint32, falsePositiveRate float64) *BloomFilter {
	if falsePositiveRate <= 0.0 || falsePositiveRate >= 1.0 {
		panic("falsePositiveRate must be between 0 and 1 (exclusive)")
	}

	// m = -(n * ln(p)) / (ln(2)^2)
	m := uint32(math.Ceil(-(float64(expectedItems) * math.Log(falsePositiveRate)) / (math.Ln2 * math.Ln2)))

	// k = (m / n) * ln(2)
	k := uint32(math.Ceil((float64(m) / float64(expectedItems)) * math.Ln2))
	return &BloomFilter{
		size:      m,
		buckets:   make([]uint64, (m+63)/64),
		numHashes: k,
	}
}

func (b *BloomFilter) Insert(key interfaces.Comparable) error {
	hashs, err := key.Hash(b.numHashes)
	if err != nil {
		return err
	}

	for _, hash := range hashs {
		position := hash % b.size
		bucketIdx := position / 64
		bitPos := position % 64

		b.buckets[bucketIdx] |= 1 << bitPos
	}

	return nil
}

func (b *BloomFilter) Contains(key interfaces.Comparable) (bool, error) {
	hashs, err := key.Hash(b.numHashes)
	if err != nil {
		return false, err
	}

	for _, hash := range hashs {
		position := hash % b.size
		bucketIdx := position / 64
		bitPos := position % 64

		if (b.buckets[bucketIdx] & (1 << bitPos)) == 0 {
			return false, nil
		}
	}

	return true, nil
}
