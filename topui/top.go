package topui

import (
	"github.com/rivo/tview"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"snet/stats"
)

const (
	_ = 1 << (iota * 10)
	b
	kb
	mb
)

func hb(s uint64) string {
	if s < b {
		return fmt.Sprintf("%dB", s)
	}
	if s < kb {
		return fmt.Sprintf("%.2fKB", float64(s)/b)
	}
	if s < mb {
		return fmt.Sprintf("%.2fMB", float64(s)/kb)
	}
	return fmt.Sprintf("%.2fGB", float64(s)/mb)
}

func highlight(text, word string) string {
	idxStart := strings.Index(text, word)
	if idxStart == -1 {
		return text
	}
	idxEnd := idxStart + len(word)
	return fmt.Sprintf("%s[red]%s[white]%s", text[:idxStart], word, text[idxEnd:])
}

type Top struct {
	addr          string
	app           *tview.Application
	network       *tview.TextView
	stats         *stats.StatsApiModel
	suspend       bool
	suspendAction *SelectAction
	sortBy        rune
	hostFilter    string
	refreshLock   sync.Mutex
}

func (t *Top) Suspend() {
	if !t.suspend {
		t.suspendAction.Select()
	}
}

func (t *Top) UnSuspend() {
	if t.suspend {
		t.suspend = false
		t.suspendAction.UnSelect()
	}
}

func (t *Top) FilterHost(search string) {
	t.hostFilter = search
}

func NewTop(addr string) *Top {
	t := new(Top)

	t.addr = addr
	t.app = tview.NewApplication()
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	t.network = tview.NewTextView()

	t.suspendAction = NewSelectAction("Suspend", keySuspend, true, false, func() {
		t.suspend = !t.suspend
	})

	bar := NewToolBar(
		t,
		NewSelectAction("Quit", keyQuit, false, false, func() {
			t.app.Stop()
		}),
		t.suspendAction,
		NewSelectAction("↓", 'j', false, false, func() {
			t.Suspend()
			r, c := t.network.GetScrollOffset()
			t.network.ScrollTo(r+1, c)
		}),
		NewSelectAction("↑", 'k', false, false, func() {
			t.Suspend()
			r, c := t.network.GetScrollOffset()
			t.network.ScrollTo(r-1, c)
		}),
		NewSelectGroupAction("|Sort:",
			NewSelectAction("Rx/s", keySortByRxRate, true, true, func() {
				t.sort(keySortByRxRate)
				t.Refresh(false)
			}),
			NewSelectAction("Tx/s", keySortByTxRate, true, false, func() {
				t.sort(keySortByTxRate)
				t.Refresh(false)
			}),
			NewSelectAction("Rx Size", keySortByRxSize, true, false, func() {
				t.sort(keySortByRxSize)
				t.Refresh(false)
			}),
			NewSelectAction("Tx Size", keySortByTxSize, true, false, func() {
				t.sort(keySortByTxSize)
				t.Refresh(false)
			}),
			NewSelectAction("Host", keySortByHost, true, false, func() {
				t.sort(keySortByHost)
				t.Refresh(false)
			}),
			NewSelectAction("Port", keySortByPort, true, false, func() {
				t.sort(keySortByPort)
				t.Refresh(false)
			}),
		),
	)

	layout.AddItem(t.network, 0, 1, false).
		AddItem(bar, 2, 0, true)
	t.app.SetRoot(layout, true)
	return t
}

func (t *Top) sort(key rune) {
	t.sortBy = key
}

func (t *Top) pullMetrics() error {
	r, err := http.Get(t.addr + "/stats")
	if err != nil {
		return err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	t.stats = new(stats.StatsApiModel)
	if err := json.Unmarshal(body, t.stats); err != nil {
		return err
	}
	return nil
}

func (t *Top) Refresh(draw bool) {
	if t.suspend {
		return
	}
	t.refreshLock.Lock()
	defer t.refreshLock.Unlock()

	r := t.stats
	t.network.Clear()
	fmt.Fprintf(t.network, "Uptime: %s, Rx Total: %s, Tx Total: %s\n\n", r.Uptime, hb(r.Total.RxSize), hb(r.Total.TxSize))
	switch t.sortBy {
	case keySortByTxRate:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].TxRate > r.Hosts[j].TxRate
		})
	case keySortByRxRate:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].RxRate > r.Hosts[j].RxRate
		})
	case keySortByTxSize:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].TxSize > r.Hosts[j].TxSize
		})
	case keySortByRxSize:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].RxSize > r.Hosts[j].RxSize
		})
	case keySortByHost:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].Host > r.Hosts[j].Host
		})
	case keySortByPort:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].Port > r.Hosts[j].Port
		})
	}
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Host\tPort\tRX rate\tTX rate\tRX size\tTX size\t")
	for _, h := range r.Hosts {
		host := h.Host
		if t.hostFilter != "" {
			if !strings.Contains(h.Host, t.hostFilter) {
				continue
			} else {
				host = highlight(host, t.hostFilter)
			}
		}
		fmt.Fprintf(w, "%s\t%d\t%s/s\t%s/s\t%s \t%s\t\n",
			host, h.Port, hb(uint64(h.RxRate)), hb(uint64(h.TxRate)), hb(h.RxSize), hb(h.TxSize))
	}
	w.Flush()
	fmt.Fprintf(t.network, "%s", buf.String())
	t.network.ScrollToBeginning()
	if draw {
		t.app.Draw()
	}
}

func (t *Top) Run() {
	if err := t.pullMetrics(); err != nil {
		panic(err)
	}
	go func() {
		for {
			if err := t.pullMetrics(); err != nil {
				t.app.Stop()
				panic(err)
			}
			t.Refresh(true)
			time.Sleep(2 * time.Second)
		}
	}()
	if err := t.app.Run(); err != nil {
		panic(err)
	}
}
