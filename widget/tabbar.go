// SPDX-License-Identifier: Unlicense OR MIT

package widget

import (
	"gioui.org/layout"
)

type (
	Tabbar struct {
		Tabs         []*Tab
		byAddress    map[interface{}]*Tab
		Active       *Tab
		becameActive *Tab
	}

	Tab struct {
		Label  string
		w      layout.Layouter
		button Button
	}
)

func NewTabbar(tabs ...*Tab) *Tabbar {
	tb := Tabbar{
		Tabs:      tabs,
		byAddress: map[interface{}]*Tab{},
	}
	for _, tab := range tabs {
		tb.byAddress[tab.w] = tab
	}
	return &tb
}

func (tb *Tabbar) ProcessEvents(gtx *layout.Context) {
	for _, tab := range tb.Tabs {
		if tab.button.Clicked(gtx) {
			tb.Activate(tab.w)
		}
	}
}

func (tb *Tabbar) Activate(key interface{}) {
	if tab, ok := tb.byAddress[key]; ok {
		tb.Active = tab
		tb.becameActive = tab
	}
}

// BecameActive returns true if the given tab has become active since the last time
// BecameActive was called.  Use it in your Layout function to do things like
// request focus when the tab becomes active.
func (tb *Tabbar) BecameActive(key interface{}) bool {
	if tab, ok := tb.byAddress[key]; ok && tb.Active == tab && tb.becameActive == tab {
		tb.becameActive = nil
		return true
	}
	return false
}

func NewTab(label string, w layout.Layouter) *Tab {
	return &Tab{Label: label, w: w}
}

func (t *Tab) LayoutButton(gtx *layout.Context) {
	t.button.Layout(gtx)
}

func (t *Tab) Layout(gtx *layout.Context) {
	t.w.Layout(gtx)
}
