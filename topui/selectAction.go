package topui

import (
	"fmt"
	"unicode/utf8"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type SelectAction struct {
	*tview.Box
	Label      string
	Key        rune
	selected   bool
	selectable bool
	cb         func()
}

func NewSelectAction(label string, key rune, selectable, selected bool, cb func()) *SelectAction {
	return &SelectAction{
		Box:        tview.NewBox(),
		Label:      label,
		Key:        key,
		selectable: selectable,
		selected:   selected,
		cb:         cb}
}

func (s *SelectAction) Selectable() bool {
	return s.selectable
}

func (s *SelectAction) Selected() bool {
	return s.selected
}

func (s *SelectAction) Do() {
	if s.cb != nil {
		s.cb()
	}
}

func (s *SelectAction) SetLabel(label string) {
	s.Label = label
}

func (s *SelectAction) TextLen() int {
	return utf8.RuneCountInString(s.Label) + 3
}

func (s *SelectAction) Select() {
	s.selected = true
	if s.cb != nil {
		s.cb()
	}
}

func (s *SelectAction) UnSelect() {
	s.selected = false
}

func (s *SelectAction) Toggle() {
	s.selected = !s.selected
	if s.cb != nil {
		s.cb()
	}
}

func (s *SelectAction) Draw(screen tcell.Screen) {
	s.Box.Draw(screen)
	x, y, w, _ := s.GetInnerRect()
	var r string
	if s.selected {
		r = fmt.Sprintf(`[black:white]%s(%c)[white:black]`, s.Label, s.Key)
	} else {
		r = fmt.Sprintf("%s([red]%c[white:black])", s.Label, s.Key)
	}
	tview.Print(screen, r, x, y, w, tview.AlignLeft, tcell.ColorWhite)
}

type SelectGroupAction struct {
	*tview.Flex
	actions []*SelectAction
	keys    []rune
}

func NewSelectGroupAction(name string, actions ...*SelectAction) *SelectGroupAction {
	flex := tview.NewFlex()
	flex.AddItem(tview.NewTextView().SetText(name), len(name)+1, 0, false)
	keys := make([]rune, 0, len(actions))
	for _, a := range actions {
		keys = append(keys, a.Key)
		flex.AddItem(a, a.TextLen()+1, 0, false)
	}
	flex.AddItem(tview.NewBox(), 0, 1, false)
	return &SelectGroupAction{Flex: flex, actions: actions, keys: keys}
}

func (g *SelectGroupAction) Draw(screen tcell.Screen) {
	g.Flex.Draw(screen)
}

func (g *SelectGroupAction) Select(key rune) {
	for _, a := range g.actions {
		if a.Key == key {
			a.Select()
		} else {
			a.UnSelect()
		}
	}
}
