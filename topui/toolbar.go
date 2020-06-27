package topui

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	keyQuit         = 'q'
	keySuspend      = 's'
	keySortByRxRate = 'r'
	keySortByRxSize = 'R'
	keySortByTxRate = 't'
	keySortByTxSize = 'T'
	keySortByHost   = 'h'
	keySortByPort   = 'p'
	keyFilter       = '/'
)

type ToolBar struct {
	*tview.Flex
	keyActionMap map[rune]tview.Primitive
}

func NewToolBar(actions ...tview.Primitive) *ToolBar {
	flex := tview.NewFlex()
	m := make(map[rune]tview.Primitive)
	for _, a := range actions {
		if s, ok := a.(*SelectAction); ok {
			flex.AddItem(a, s.TextLen()+1, 0, false)
			m[s.Key] = a
		} else if s, ok := a.(*SelectGroupAction); ok {
			flex.AddItem(a, 0, 1, false)
			for _, k := range s.keys {
				m[k] = a
			}
		}
	}
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Rune()
		if a, ok := m[key]; ok {
			if a1, ok := a.(*SelectAction); ok {
				if a1.Selectable() {
					a1.Toggle()
				} else {
					a1.Do()
				}
			} else if a1, ok := a.(*SelectGroupAction); ok {
				a1.Select(key)
			}
			return nil
		}
		return event
	})
	return &ToolBar{
		Flex:         flex,
		keyActionMap: m,
	}
}

func (t *ToolBar) Draw(screen tcell.Screen) {
	t.Flex.Draw(screen)
}
