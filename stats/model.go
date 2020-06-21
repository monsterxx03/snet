package stats

type total struct {
	RxSize string
	TxSize string
}

type host struct {
	Host   string
	RxRate string
	TxRate string
	RxSize uint64
	TxSize uint64
}

type StatsApiModel struct {
	Total total
	Hosts []*host
}
