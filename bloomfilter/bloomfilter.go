package bloomfilter

import (
	"hash/fnv"
	"math"
)

func hash(item []byte, seed int8) uint32 {
	h := fnv.New32a()
	h.Write(item)
	h.Write([]byte{byte(seed)})
	return h.Sum32()
}

type Bloomfilter struct {
	capacity  int
	errorRate float64
	hashN     int8
	size      uint32
	bits      *Bitarray
}

func NewBloomfilter(capacity int, errorRate float64) (*Bloomfilter, error) {
	hashN := int8(math.Ceil(math.Log2(1 / errorRate)))
	size := uint32(math.Ceil(float64(capacity) * math.Abs(math.Log2(errorRate)/math.Log(2))))
	size = size + 8 - size%8
	bits, err := NewBitarray(uint32(size))
	if err != nil {
		return nil, err
	}
	return &Bloomfilter{capacity: capacity, errorRate: errorRate, hashN: hashN, size: size, bits: bits}, nil
}

func (b *Bloomfilter) Add(item []byte) error {
	for i := 0; int8(i) < b.hashN; i++ {
		loc := hash(item, int8(i)) % b.size
		if err := b.bits.Setbit(loc); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bloomfilter) Has(item []byte) bool {
	for i := 0; int8(i) < b.hashN; i++ {
		loc := hash(item, int8(i)) % b.size
		if !b.bits.IsSet(loc) {
			return false
		}
	}
	// it's false positive
	return true
}

func (b *Bloomfilter) FillRatio() float64 {
	return float64(b.bits.Count()) / float64(b.size)
}
