package bloomfilter

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestBloomfilter(t *testing.T) {
	capacity := 400
	errorRate := 0.001
	b, err := NewBloomfilter(capacity, errorRate)
	if err != nil {
		t.Error(err)
	}
	if b.hashN != 10 {
		t.Error("wrong hashN", b.hashN)
	}
	if b.size != 5760 {
		t.Error("wrong bit size", b.size)
	}
	// test error rate
	test_items := [][]byte{}
	for i := 0; i < capacity; i++ {
		item := make([]byte, 10)
		rand.Read(item)
		b.Add(item)
		test_items = append(test_items, item)
	}
	in_test_items := func(item []byte) bool {
		for _, _item := range test_items {
			if bytes.Equal(_item, item) {
				return true
			}
		}
		return false
	}
	errCount := 0
	falsePositiveCount := 0
	for i := 0; uint32(i) < b.size; i++ {
		item := []byte{}
		if i < len(test_items) {
			// reuse test_items
			item = test_items[i]
		} else {
			item = make([]byte, 10)
			rand.Read(item)
		}
		if !b.Has(item) && in_test_items(item) {
			errCount++
		}
		if b.Has(item) && !in_test_items(item) {
			falsePositiveCount++
		}
	}
	if errCount != 0 {
		t.Error("err count not 0 ", errCount)
	}
	er := float64(falsePositiveCount) / float64(b.size)
	if er > errorRate {
		t.Error("unexpected error rate", er)
	}
}
