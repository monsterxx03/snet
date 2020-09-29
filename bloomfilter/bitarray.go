package bloomfilter

import (
	"fmt"
)

type Bitarray struct {
	bits []bool
}

func NewBitarray(size uint32) (*Bitarray, error) {
	return &Bitarray{bits: make([]bool, size)}, nil
}

func (b *Bitarray) Len() uint32 {
	return uint32(len(b.bits))
}

func (b *Bitarray) Setbit(loc uint32) error {
	if loc > b.Len() {
		return fmt.Errorf("%d is larger than array size %d", loc, b.Len())
	}
	b.bits[loc] = true
	return nil
}

func (b *Bitarray) IsSet(loc uint32) bool {
	return b.bits[loc]
}

func (b *Bitarray) Count() uint32 {
	count := 0
	for _, _b := range b.bits {
		if _b {
			count++
		}
	}
	return uint32(count)
}
