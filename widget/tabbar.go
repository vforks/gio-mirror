// SPDX-License-Identifier: Unlicense OR MIT

package widget

import (
	"gioui.org/layout"
)

type (
	Tabbar struct {
		Tabs      []*Tab
		byAddress map[interface{}]*Tab
		Active    *Tab
	}

	Tab struct {
		Label  string
		w      layout.Layouter
		button Button
	}

	Activater interface {
		Activate()
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
		if act, ok := key.(Activater); ok {
			act.Activate()
		}
	}
}

func (tb *Tabbar) Prev() {
	for i, tab := range tb.Tabs {
		if tab == tb.Active {
			if i == 0 {
				tb.Activate(tb.Tabs[len(tb.Tabs)-1].w)
			} else {
				tb.Activate(tb.Tabs[i-1].w)
			}
			return
		}
	}
}

func (tb *Tabbar) Next() {
	for i, tab := range tb.Tabs {
		if tab == tb.Active {
			if i < len(tb.Tabs)-1 {
				tb.Activate(tb.Tabs[i+1].w)
			} else {
				tb.Activate(tb.Tabs[0].w)
			}
			return
		}
	}
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
