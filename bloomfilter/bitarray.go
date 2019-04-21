package bloomfilter

import (
	"fmt"
	"math/bits"
)

type Bitarray struct {
	bytes []byte
}

func NewBitarray(size uint32) (*Bitarray, error) {
	if size%8 != 0 {
		return nil, fmt.Errorf("%d is not 8 disionable", size)
	}
	return &Bitarray{bytes: make([]byte, size/8)}, nil
}

func (b *Bitarray) Len() uint32 {
	return uint32(len(b.bytes) * 8)
}

func (b *Bitarray) Setbit(loc uint32) error {
	if loc > b.Len() {
		return fmt.Errorf("%d is larger than array size %d", loc, b.Len())
	}
	b.bytes[loc/8] |= 1 << (loc % 8)
	return nil
}

func (b *Bitarray) IsSet(loc uint32) bool {
	return (b.bytes[loc/8] & (1 << (loc % 8))) > 0
}

func (b *Bitarray) Count() uint32 {
	count := 0
	for _, _b := range b.bytes {
		count += bits.OnesCount8(_b)
	}
	return uint32(count)
}
