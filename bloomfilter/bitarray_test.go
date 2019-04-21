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
	if err := ba.Setbit(13); err != nil {
		t.Error(err)
	}
	if !ba.IsSet(13) {
		t.Error("test set failed")
	}
}
