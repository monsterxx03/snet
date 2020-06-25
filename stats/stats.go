package stats

import (
	"container/ring"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
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
	if s.rxRing.Move(-3).Value != nil {
		valPrev := s.rxRing.Move(-3).Value.(uint64)
		valCur := s.rxRing.Move(-1).Value.(uint64)
		return float64(valCur-valPrev) / RATE_INTERVAL
	}
	return 0
}

func (s *HostStats) TxRate2() float64 {
	if s.txRing.Move(-2).Value != nil {
		valPrev := s.txRing.Move(-2).Value.(uint64)
		valCur := s.txRing.Move(-1).Value.(uint64)
		return float64(valCur-valPrev) / RATE_INTERVAL
	}
	return 0
}

type Stats struct {
	uptime  time.Time
	rxBytes uint64
	txBytes uint64
	hosts   map[string]*HostStats
}

func NewStats() *Stats {
	return &Stats{uptime: time.Now(), hosts: make(map[string]*HostStats)}
}

// should be called once every second
func (s *Stats) Record(rxMap, txMap map[string]uint64) {
	var rxBytes, txBytes uint64 = 0, 0
	for host, val := range rxMap {
		rxBytes += val
		if _p, ok := s.hosts[host]; ok {
			_p.RecordRx(val)
		} else {
			_p := NewHostStats()
			_p.RecordRx(val)
			s.hosts[host] = _p
		}
	}
	s.rxBytes = rxBytes

	for host, val := range txMap {
		txBytes += val
		if _p, ok := s.hosts[host]; ok {
			_p.RecordTx(val)
		} else {
			_p := NewHostStats()
			_p.RecordTx(val)
			s.hosts[host] = _p
		}
	}
	s.txBytes = txBytes
}

func (s *Stats) ToJson() []byte {
	result := new(StatsApiModel)
	result.Uptime = time.Now().Sub(s.uptime).Truncate(time.Second).String()
	result.Total = total{RxSize: s.rxBytes, TxSize: s.txBytes}
	result.Hosts = make([]*host, 0, len(s.hosts))
	for h, p := range s.hosts {
		pa := strings.Split(h, ":")
		port, _ := strconv.Atoi(pa[1])
		result.Hosts = append(result.Hosts, &host{Host: pa[0],
			Port:   port,
			RxRate: p.RxRate2(), TxRate: p.TxRate2(),
			RxSize: p.RxTotal(), TxSize: p.TxTotal(),
		})
	}
	res, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
	}
	return res
}

func (s *Stats) Print() {
	for h, p := range s.hosts {
		fmt.Printf("%s, rx rate: %.2f/s,tx rate: %.2f/s, rx size: %d, tx size: %d\n", h, p.RxRate2(), p.TxRate2(), p.RxTotal(), p.TxTotal())
	}
	if len(s.hosts) > 0 {
		fmt.Println()
	}
}

type P struct {
	Host string
	Rx   uint64
	Tx   uint64
}
