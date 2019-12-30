// SPDX-License-Identifier: Unlicense OR MIT

package widget

import (
	"gioui.org/layout"
)

type Tabbar struct {
	Tabs   []*Tab
	Active int
}

type Tab struct {
	Label        string
	button       Button
	becameActive bool
}

func NewTabbar() *Tabbar {
	return &Tabbar{}
}

func (tb *Tabbar) Add(label string) {
	tb.Tabs = append(tb.Tabs, &Tab{Label: label})
}

func (tb *Tabbar) Layout(gtx *layout.Context) {
	tb.processEvents(gtx)
}

func (tb *Tabbar) processEvents(gtx *layout.Context) {
	for i := range tb.Tabs {
		if tb.Tabs[i].button.Clicked(gtx) {
			// log.Printf("Setting active tab to %d", i)
			tb.SetActiveIndex(i)
		}
	}
}

func (tb *Tabbar) SetActiveLabel(label string) {
	for i, tab := range tb.Tabs {
		if tab.Label == label && tb.Active != i {
			tab.becameActive = true
			tb.Active = i
		} else {
			tab.becameActive = false
		}
	}
}

func (tb *Tabbar) SetActiveIndex(index int) {
	for i, tab := range tb.Tabs {
		if i == index && tb.Active != i {
			tab.becameActive = true
			tb.Active = i
		} else {
			tab.becameActive = false
		}
	}
}

func (t *Tab) Layout(gtx *layout.Context) {
	t.button.Layout(gtx)
}

// BecameActive returns true if this tab has become active since the last time
// BecameActive was called.  Use it in your Layout function to do things like
// request focus when the tab becomes active.
func (t *Tab) BecameActive() bool {
	ba := t.becameActive
	t.becameActive = false
	return ba
}
