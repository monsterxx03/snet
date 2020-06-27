package topui

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"fmt"
)

type FilterAction struct {
	*tview.Box
	Label    string
	Key      rune
	inputing bool
}

func NewFilterAction(label string, key rune) *FilterAction {
	return &FilterAction{
		Box:   tview.NewBox(),
		Label: label,
		Key:   key,
	}
}

func (s *FilterAction) Draw(screen tcell.Screen) {
	if s.inputing {
	} else {
		s.Box.Draw(screen)
		x, y, w, _ := s.GetInnerRect()
		r := fmt.Sprintf(`%s[red]%c[white:black]`, s.Label, s.Key)
		tview.Print(screen, r, x, y, w, tview.AlignLeft, tcell.ColorWhite)
	}
}

func (s *FilterAction) Size() int {
	if s.inputing {
		return 10
	}
	return len(s.Label) + 3
}
