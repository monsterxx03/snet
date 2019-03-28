package main

import (
	"testing"
	"time"
)

func TestLRU(t *testing.T) {
	fakeTTL := time.Now().Add(time.Hour * 10000) // skip ttl check
	_, err := NewLRU(-1)
	if err == nil {
		t.Fatalf("negative capacity should not be allowed")
	}
	l, err := NewLRU(3)
	if l.Add("k1", "v1", fakeTTL) == false {
		t.Errorf("k1 should't exist")
	}

	if l.Get("k1") != "v1" {
		t.Error("wrong value for k1")
	}

	l.Add("k2", "v2", fakeTTL)
	l.Add("k3", "v3", fakeTTL)
	l.Add("k4", "v4", fakeTTL)
	if l.Get("k1") != nil {
		t.Error("k1 should be evicted")
	}
	// k2 is moved to front
	if l.Get("k2") != "v2" {
		t.Error("k2 invalid")
	}
	l.Add("k5", "v5", fakeTTL)
	// k3 will be evicted
	if l.Get("k3") != nil {
		t.Error("k3 should be evicted")
	}
	if l.Get("k2") != "v2" {
		t.Error("k2 invalid")
	}
}

func TestLRUWithTTL(t *testing.T) {
	cache, _ := NewLRU(3)
	ttl := time.Now().Add(time.Second * 1)
	cache.Add("k1", "v1", ttl)
	if cache.Get("k1") != "v1" {
		t.Error("cache inconsistent")
	}
	time.Sleep(time.Second * 2)
	if cache.Get("k1") != nil {
		t.Error("should alredy expired")
	}
}
