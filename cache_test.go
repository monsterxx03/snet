package main

import (
	_ "fmt"
	"testing"
)

func TestLRU(t *testing.T) {
	_, err := NewLRU(-1)
	if err == nil {
		t.Fatalf("negative capacity should not be allowed")
	}
	l, err := NewLRU(3)
	if l.Add("k1", "v1") == false {
		t.Errorf("k1 should't exist")
	}

	if l.Get("k1") != "v1" {
		t.Error("wrong value for k1")
	}

	l.Add("k2", "v2")
	l.Add("k3", "v3")
	l.Add("k4", "v4")
	if l.Get("k1") != nil {
		t.Error("k1 should be evicted")
	}
	// k2 is moved to front
	if l.Get("k2") != "v2" {
		t.Error("k2 invalid")
	}
	l.Add("k5", "v5")
	// k3 will be evicted
	if l.Get("k3") != nil {
		t.Error("k3 should be evicted")
	}
	if l.Get("k2") != "v2" {
		t.Error("k2 invalid")
	}
}
