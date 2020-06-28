package topui

import (
	"fmt"
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
	top          *Top
	keyActionMap map[rune]tview.Primitive
}

func NewToolBar(top *Top, actions ...tview.Primitive) *ToolBar {
	flex := tview.NewFlex()
	bar := &ToolBar{Flex: flex, top: top}
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
	bar.keyActionMap = m

	filterAction := NewSelectAction("Filter:", keyFilter, false, false, nil)
	filterInput := tview.NewInputField()
	filterInput.SetChangedFunc(func(text string) {
		top.FilterHost(text)
	})
	filterInput.SetDoneFunc(func(key tcell.Key) {
		// exit searching state
		if key == tcell.KeyEscape || key == tcell.KeyEnter {
			flex.RemoveItem(filterInput)
			flex.AddItem(filterAction, 0, 1, false)
			top.app.SetFocus(bar)
			top.UnSuspend()
			filterAction.SetLabel(fmt.Sprintf("Filter:[yellow]%s[white]", filterInput.GetText()))
			top.Refresh(false)
		}
	})
	flex.AddItem(filterAction, 0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Rune()
		if a, ok := m[key]; ok {
			if s, ok := a.(*SelectAction); ok {
				if s.Selectable() {
					s.Toggle()
				} else {
					s.Do()
				}
			} else if s, ok := a.(*SelectGroupAction); ok {
				s.Select(key)
			}
			return nil
		} else if key == keyFilter {
			// changing to search state
			flex.RemoveItem(filterAction)
			flex.AddItem(filterInput, 0, 1, false)
			top.app.SetFocus(filterInput)
			top.Suspend()
		}
		return event
	})
	return bar
}

func (t *ToolBar) Draw(screen tcell.Screen) {
	t.Flex.Draw(screen)
}
