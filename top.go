package main

import (
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"io/ioutil"
	"net/http"
	"sort"
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

const (
	sortByRxRate = 'r'
	sortByRxSize = 'R'
	sortByTxRate = 't'
	sortByTxSize = 'T'
	sortByHost   = 'h'
	sortByPort   = 'p'
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

type Action struct {
	Label    string
	Key      rune
	selected bool
}

func NewAction(label string, key rune) *Action {
	return &Action{Label: label, Key: key, selected: false}
}

func (a *Action) String() string {
	if a.selected {
		return fmt.Sprintf(`[black:white]%s(%c)[white:black]`, a.Label, a.Key)
	}
	return fmt.Sprintf("%s([red]%c[white:black])", a.Label, a.Key)
}

func (a *Action) Select(t bool) *Action {
	a.selected = t
	return a
}

func (a *Action) Toggle() *Action {
	a.selected = !a.selected
	return a
}

type ActionGroup struct {
	name    string
	actions []*Action
}

func NewActionGroup(name string, actions []*Action) *ActionGroup {
	g := &ActionGroup{name: name, actions: actions}
	return g
}

func (g *ActionGroup) Select(key rune) {
	for _, a := range g.actions {
		a.Select(a.Key == key)
	}
}

func (g *ActionGroup) String() string {
	s := fmt.Sprintf("%s ", g.name)
	for _, a := range g.actions {
		s += a.String() + " "
	}
	return s
}

type ToolBar struct {
	*tview.Box
	quitAction    *Action
	suspendAction *Action
	sortGroup     *ActionGroup
}

func NewToolBar() *ToolBar {
	return &ToolBar{
		Box:           tview.NewBox(),
		quitAction:    NewAction("Quit", 'q'),
		suspendAction: NewAction("Suspend", 's'),
		sortGroup: NewActionGroup("Sort By:", []*Action{
			NewAction("RX rate", sortByRxRate),
			NewAction("TX rate", sortByTxRate),
			NewAction("RX size", sortByRxSize),
			NewAction("TX size", sortByTxSize),
			NewAction("Host", sortByHost),
			NewAction("Port", sortByPort),
		}),
	}
}
func (t *ToolBar) Do(key rune) {
	switch key {
	case 'q':
		t.quitAction.Select(true)
	case 's':
		t.suspendAction.Toggle()
	case sortByRxSize, sortByRxRate, sortByTxSize, sortByTxRate, sortByHost, sortByPort:
		t.sortGroup.Select(key)
	}
}

func (t *ToolBar) Draw(screen tcell.Screen) {
	t.Box.Draw(screen)
	x, y, width, _ := t.GetInnerRect()

	tview.Print(screen, fmt.Sprintf("%s %s|%s", t.quitAction, t.suspendAction, t.sortGroup),
		x, y, width, tview.AlignLeft, tcell.ColorWhite)
}

type Top struct {
	addr    string
	app     *tview.Application
	network *tview.TextView
	dns     *tview.TextView
	toolBar *ToolBar
	stats   *stats.StatsApiModel
	suspend bool
	sortBy  rune
}

func NewTop(addr string) *Top {
	bar := NewToolBar()
	t := new(Top)
	t.addr = addr
	t.toolBar = bar
	t.app = tview.NewApplication()
	t.sort(sortByRxRate)
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 's':
			t.suspend = !t.suspend
			bar.Do('s')
			return nil
		case 'q':
			t.app.Stop()
			return nil
		case sortByRxRate, sortByRxSize, sortByTxRate, sortByTxSize, sortByHost, sortByPort:
			t.sortBy = event.Rune()
			bar.Do(event.Rune())
			t.Refresh(false)
			return nil
		}
		return event
	})
	layoutUp := tview.NewFlex()
	t.network = tview.NewTextView()
	t.dns = tview.NewTextView()
	t.dns.SetTitle("DNS Status")
	layoutUp.AddItem(t.network, 0, 1, false).
		AddItem(t.dns, 0, 1, false)

	layout.AddItem(layoutUp, 0, 1, false).
		AddItem(bar, 2, 0, false)
	t.app.SetRoot(layout, true)
	return t
}

func (t *Top) sort(key rune) {
	t.sortBy = key
	t.toolBar.Do(key)
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
	r := t.stats
	t.network.Clear()
	fmt.Fprintf(t.network, "Uptime: %s, Rx Total: %s, Tx Total: %s\n\n", r.Uptime, hb(r.Total.RxSize), hb(r.Total.TxSize))
	switch t.sortBy {
	case sortByTxRate:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].TxRate > r.Hosts[j].TxRate
		})
	case sortByRxRate:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].RxRate > r.Hosts[j].RxRate
		})
	case sortByTxSize:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].TxSize > r.Hosts[j].TxSize
		})
	case sortByRxSize:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].RxSize > r.Hosts[j].RxSize
		})
	case sortByHost:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].Host > r.Hosts[j].Host
		})
	case sortByPort:
		sort.Slice(r.Hosts, func(i, j int) bool {
			return r.Hosts[i].Port > r.Hosts[j].Port
		})
	}
	w := tabwriter.NewWriter(t.network, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "Host\tPort\tRX rate\tTX rate\tRX size\tTX size\t")
	for _, h := range r.Hosts {
		fmt.Fprintf(w, "%s\t%d\t%s/s\t%s/s\t%s \t%s\t\n",
			h.Host, h.Port, hb(uint64(h.RxRate)), hb(uint64(h.TxRate)), hb(h.RxSize), hb(h.TxSize))
	}
	w.Flush()
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
