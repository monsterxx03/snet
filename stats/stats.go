package stats

import (
	"container/ring"
	"fmt"
)

const (
	RATE_INTERVAL = 2
	RING_SiZE     = RATE_INTERVAL + 1
)

type HostStats struct {
	rxRing *ring.Ring
	txRing *ring.Ring
}

func NewHostStats() *HostStats {
	return &HostStats{rxRing: ring.New(RING_SiZE), txRing: ring.New(RING_SiZE)}
}

func (s *HostStats) RxTotal() uint64 {
	if s.rxRing.Value == nil {
		if s.rxRing.Prev().Value == nil {
			return 0
		}
		return s.rxRing.Prev().Value.(uint64)
	}
	return s.rxRing.Value.(uint64)
}

func (s *HostStats) TxTotal() uint64 {
	if s.txRing.Value == nil {
		if s.txRing.Prev().Value == nil {
			return 0
		}
		return s.txRing.Prev().Value.(uint64)
	}
	return s.txRing.Value.(uint64)
}

// Record current total rx bytes length
func (s *HostStats) RecordRx(rx uint64) {
	s.rxRing.Value = rx
	s.rxRing = s.rxRing.Next()

}

// Record current total tx bytes length
func (s *HostStats) RecordTx(tx uint64) {
	s.txRing.Value = tx
	s.txRing = s.txRing.Next()
}

func (s *HostStats) RxRate2() float64 {
	if s.rxRing.Move(-2).Value != nil {
		valPrev := s.rxRing.Move(-2).Value.(uint64)
		valCur := s.rxRing.Move(-1).Value.(uint64)
		return float64(valCur-valPrev) / 2
	}
	return 0
}

func (s *HostStats) TxRate2() float64 {
	if s.txRing.Move(-2).Value != nil {
		valPrev := s.txRing.Move(-2).Value.(uint64)
		valCur := s.txRing.Move(-1).Value.(uint64)
		return float64(valCur-valPrev) / 2
	}
	return 0
}

type Stats struct {
	hosts map[string]*HostStats
}

func NewStats() *Stats {
	return &Stats{hosts: make(map[string]*HostStats)}
}

// should be called once every second
func (s *Stats) Record(rxMap, txMap map[string]uint64) {
	for host, val := range rxMap {
		if _p, ok := s.hosts[host]; ok {
			_p.RecordRx(val)
		} else {
			_p := NewHostStats()
			_p.RecordRx(val)
			s.hosts[host] = _p
		}
	}

	for host, val := range txMap {
		if _p, ok := s.hosts[host]; ok {
			_p.RecordTx(val)
		} else {
			_p := NewHostStats()
			_p.RecordTx(val)
			s.hosts[host] = _p
		}
	}
}

func (s *Stats) Print() {
	for h, p := range s.hosts {
		fmt.Printf("%s,rx/s: %f,tx/s: %f, rx: %d, tx: %d\n", h, p.RxRate2(), p.TxRate2(), p.RxTotal(), p.TxTotal())
	}
	fmt.Println()
}

type P struct {
	Host string
	Rx   uint64
	Tx   uint64
}
