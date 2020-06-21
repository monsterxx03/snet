package stats

import (
	"container/ring"
)

type Stats struct {
	rxRing *ring.Ring
	txRing *ring.Ring
}

func NewStats() *Stats {
	return &Stats{rxRing: ring.New(60), txRing: ring.New(60)}
}

// Record every second's transmited bytes length
func (s *Stats) RecordTx(length uint64) {
	s.txRing.Value = length
	s.txRing = s.txRing.Next()
}

// Record every second's received bytes length
func (s *Stats) RecordRx(length uint64) {
	s.rxRing.Value = length
	s.rxRing = s.rxRing.Next()
}

func (s *Stats) RxRate2() float64 {
	if s.rxRing.Move(-2).Value != nil {
		valPrev := s.rxRing.Move(-2).Value.(uint64)
		valCur := s.rxRing.Move(-1).Value.(uint64)
		return float64(valCur-valPrev) / 2
	}
	return 0
}

func (s *Stats) TxRate2() float64 {
	if s.txRing.Move(-2).Value != nil {
		valPrev := s.txRing.Move(-2).Value.(uint64)
		valCur := s.txRing.Move(-1).Value.(uint64)
		return float64(valCur-valPrev) / 2
	}
	return 0
}
