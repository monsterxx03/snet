package stats

type total struct {
	RxSize uint64
	TxSize uint64
}

type host struct {
	Host   string
	RxRate float64
	TxRate float64
	RxSize uint64
	TxSize uint64
}

type StatsApiModel struct {
	Total total
	Hosts []*host
}
