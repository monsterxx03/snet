package main

import (
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"io/ioutil"
	"net/http"
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
		return fmt.Sprintf(`[:red]%s(%c)[-:-:-]`, a.Label, a.Key)
	}
	return fmt.Sprintf("%s([red]%c[-:-:-])", a.Label, a.Key)
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
	s := fmt.Sprintf("|%s ", g.name)
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
			NewAction("RX rate", 'r').Select(true),
			NewAction("TX rate", 't'),
			NewAction("RX size", 'R'),
			NewAction("TX size", 'T'),
			NewAction("Host", 'h'),
		}),
	}
}
func (t *ToolBar) Do(key rune) {
	switch key {
	case 'q':
		t.quitAction.Select(true)
	case 's':
		t.suspendAction.Toggle()
	case 'r', 't', 'R', 'T', 'h':
		t.sortGroup.Select(key)
	}
}


func (t *ToolBar) Draw(screen tcell.Screen) {
	t.Box.Draw(screen)
	x, y, width, _ := t.GetInnerRect()

	tview.Print(screen, fmt.Sprintf("%s %s %s", t.quitAction, t.suspendAction, t.sortGroup),
		x, y, width, tview.AlignLeft, tcell.ColorWhite)
}

type Top struct {
	addr    string
	app     *tview.Application
	network *tview.TextView
	dns     *tview.TextView
	suspend bool
}

func NewTop(addr string) *Top {
	t := new(Top)
	t.addr = addr
	t.app = tview.NewApplication()
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	bar := NewToolBar()
	layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 's':
			t.suspend = !t.suspend
			bar.Do('s')
			return nil
		case 'q':
			t.app.Stop()
			return nil
		case 'r', 't', 'R', 'T', 'h':
			bar.Do(event.Rune())
			return nil
		}
		return event
	})
	layoutUp := tview.NewFlex()
	t.network = tview.NewTextView()
	t.network.SetBorder(true).SetTitle("Network Status")
	t.dns = tview.NewTextView()
	t.dns.SetBorder(true).SetTitle("DNS Status")
	layoutUp.AddItem(t.network, 0, 1, false).
		AddItem(t.dns, 0, 1, false)

	layout.AddItem(layoutUp, 0, 1, false).
		AddItem(bar, 2, 0, false)
	t.app.SetRoot(layout, true)
	return t
}

func (t *Top) pullMetrics() (*stats.StatsApiModel, error) {
	r, err := http.Get(t.addr + "/stats")
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r1 := new(stats.StatsApiModel)
	if err := json.Unmarshal(body, r1); err != nil {
		return nil, err
	}
	return r1, nil
}

func (t *Top) Refresh() {
	if t.suspend {
		return
	}
	r, err := t.pullMetrics()
	if err != nil {
		panic(err)
	}
	t.network.Clear()
	fmt.Fprintf(t.network, "Rx Total: %s, Tx Total: %s\n\n", hb(r.Total.RxSize), hb(r.Total.TxSize))
	for _, h := range r.Hosts {
		fmt.Fprintf(t.network, "%s\n", h.Host)
		fmt.Fprintf(t.network, "\t rx rate: %s/s tx rate: %s/s, rx size: %s, tx size: %s\n",
			hb(uint64(h.RxRate)), hb(uint64(h.TxRate)), hb(h.RxSize), hb(h.TxSize))
	}
	//t.network.ScrollToBeginning()
	t.app.Draw()
}

func (t *Top) Run() {
	go func() {
		for {
			t.Refresh()
			time.Sleep(2 * time.Second)
		}
	}()
	if err := t.app.Run(); err != nil {
		panic(err)
	}
}
