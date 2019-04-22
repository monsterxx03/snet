package bloomfilter

import (
	"testing"
)

func TestBitArray(t *testing.T) {
	if _, err := NewBitarray(3); err == nil {
		t.Error("error didn't returned for wrong bit array size")
	}
	ba, err := NewBitarray(16)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 16; i++ {
		if err := ba.Setbit(uint32(i)); err != nil {
			t.Error(err)
		}
		if !ba.IsSet(uint32(i)) {
			t.Error("test set failed at", i)
		}
	}
}
