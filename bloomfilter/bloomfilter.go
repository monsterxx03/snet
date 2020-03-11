package bloomfilter

import (
	"hash/fnv"
	"math"
)

func hash(item []byte, seed uint8) uint32 {
	h := fnv.New32a()
	// Must place seed before item.
	// FNV hash with common prefix, will produce
	// integer in small range,  cause high error rate
	h.Write(append([]byte{byte(seed)}, item...))
	return h.Sum32()
}

type Bloomfilter struct {
	capacity  int
	errorRate float64
	hashN     uint8
	size      uint32
	bits      *Bitarray
}

func NewBloomfilter(capacity int, errorRate float64) (*Bloomfilter, error) {
	hashN := uint8(math.Ceil(math.Log2(1 / errorRate)))
	size := uint32(math.Ceil(float64(capacity) * math.Abs(math.Log2(errorRate)/math.Log(2))))
	size = size + 8 - size%8
	bits, err := NewBitarray(uint32(size))
	if err != nil {
		return nil, err
	}
	return &Bloomfilter{capacity: capacity, errorRate: errorRate, hashN: hashN, size: size, bits: bits}, nil
}

func (b *Bloomfilter) Add(item []byte) error {
	for i := uint8(0); i < b.hashN; i++ {
		loc := hash(item, i) % b.size
		if err := b.bits.Setbit(loc); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bloomfilter) Size() uint32 {
	return b.size / 8
}

func (b *Bloomfilter) Has(item []byte) bool {
	for i := uint8(0); i < b.hashN; i++ {
		loc := hash(item, i) % b.size
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
