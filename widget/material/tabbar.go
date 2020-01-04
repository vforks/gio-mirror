// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type (
	// See https://material.io/components/tabs
	Tabbar struct {
		th             *Theme
		Scrollable     bool
		ClusteredFixed bool
		Alignment      layout.Alignment // only for ClusteredFixed
		Color          struct {
			// Active: use Theme.Color.Text
			// Container: use Theme.Color.Primary
			Active    color.RGBA
			Inactive  color.RGBA
			Container color.RGBA
			Divider   color.RGBA
		}
		Font     text.Font
		IconType IconType
		Buttons  layout.List
		Tabs     []Tab
	}

	Tab interface {
		Layout(*layout.Context)
		Label() string
		Icon() *Icon
	}

	IconType int
)

const (
	IconNone IconType = iota
	IconLeader
	IconTop
)

// Tabbar creates a new tab bar.  You should store it and reuse it for each
// layout, because the embedded layout.List has some state you need to keep
// around.
func (th *Theme) Tabbar() *Tabbar {
	tb := Tabbar{
		th:         th,
		Scrollable: false,
		Font: text.Font{
			Size: th.TextSize.Scale(14.0 / 16.0),
		},
		Buttons: layout.List{Axis: layout.Horizontal, Alignment: layout.Middle},
	}
	tb.Color.Active = th.Color.Text
	tb.Color.Inactive = rgb(0xffffff) // FIXME
	tb.Color.Container = th.Color.Primary
	tb.Color.Divider = rgb(0xffffff) // FIXME
	return &tb
}

func (tb *Tabbar) Layout(gtx *layout.Context, wtb *widget.Tabbar) {
	wtb.ProcessEvents(gtx)

	in := layout.Inset{Top: unit.Dp(12), Right: unit.Dp(16), Bottom: unit.Dp(12), Left: unit.Dp(16)}
	layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.
		Layout(gtx,
			layout.Rigid(func() {
				tb.Buttons.Layout(gtx, len(wtb.Tabs), func(i int) {
					// From https://material.io/components/tabs/#specs
					gtx.Constraints = layout.Constraints{
						Width:  layout.Constraint{Min: gtx.Px(unit.Dp(90)), Max: gtx.Px(unit.Dp(360))},
						Height: layout.Constraint{Min: gtx.Px(unit.Dp(48)), Max: gtx.Px(unit.Dp(48))},
					}
					in.Layout(gtx, func() {
						layout.Align(layout.Center).Layout(gtx, func() {
							tb.th.Body2(wtb.Tabs[i].Label).Layout(gtx)
						})
					})
					dims := gtx.Dimensions
					pointer.Rect(image.Rectangle{Max: gtx.Dimensions.Size}).Add(gtx.Ops)
					wtb.Tabs[i].LayoutButton(gtx)

					// Underline the active item
					if wtb.Tabs[i] == wtb.Active {
						paint.ColorOp{Color: color.RGBA{
							A: 0xff, B: 0xff,
						}}.Add(gtx.Ops)
						paint.PaintOp{
							Rect: f32.Rectangle{
								Min: f32.Point{Y: float32(dims.Size.Y - gtx.Px(unit.Dp(2)))},
								Max: f32.Point{X: float32(dims.Size.X), Y: float32(dims.Size.Y)},
							},
						}.Add(gtx.Ops)
					}
				})
			}),
			layout.Rigid(func() {
				wtb.Active.Layout(gtx)
			}),
		)
}
