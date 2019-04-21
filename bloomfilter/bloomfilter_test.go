package bloomfilter

import (
	"testing"
)

func TestBloomfilter(t *testing.T) {
	b, err := NewBloomfilter(40000, 0.00001)
	if err != nil {
		t.Error(err)
	}
	if b.hashN != 17 {
		t.Error("wrong hashN", b.hashN)
	}
	if b.size != 958512 {
		t.Error("wrong bit size", b.size)
	}
	// TODO, should test error rate
	item := []byte("baidu.com")
	if err := b.Add(item); err != nil {
		t.Error(err)
	}
	if !b.Has(item) {
		t.Error("bloomfilter test failed")
	}
	if b.Has([]byte("baidu")) {
		t.Error("should not exist")
	}
}
