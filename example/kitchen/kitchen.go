// SPDX-License-Identifier: Unlicense OR MIT

package main

// A Gio program that demonstrates Gio widgets. See https://gioui.org for more information.

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/app/headless"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"golang.org/x/exp/shiny/materialdesign/icons"
)

var screenshot = flag.String("screenshot", "", "save a screenshot to a file and exit")

type scaledConfig struct {
	Scale float32
}

type iconAndTextButton struct {
	theme  *material.Theme
	button *widget.Clickable
	icon   *widget.Icon
	word   string
}

func main() {
	flag.Parse()
	editor.SetText(longText)
	ic, err := widget.NewIcon(icons.ContentAdd)
	if err != nil {
		log.Fatal(err)
	}
	icon = ic
	progressIncrementer = make(chan int)
	gofont.Register()
	if *screenshot != "" {
		if err := saveScreenshot(*screenshot); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save screenshot: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	go func() {
		for {
			time.Sleep(time.Second)
			progressIncrementer <- 10
		}
	}()

	go func() {
		w := app.NewWindow(app.Size(unit.Dp(800), unit.Dp(650)))
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()
}

func saveScreenshot(f string) error {
	const scale = 1.5
	sz := image.Point{X: 800 * scale, Y: 600 * scale}
	w, err := headless.NewWindow(sz.X, sz.Y)
	if err != nil {
		return err
	}
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Config:      &scaledConfig{scale},
		Constraints: layout.Exact(sz),
	}
	th := material.NewTheme()
	kitchen(gtx, th)
	w.Frame(gtx.Ops)
	img, err := w.Screenshot()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return ioutil.WriteFile(f, buf.Bytes(), 0666)
}

func loop(w *app.Window) error {
	th := material.NewTheme()

	var ops op.Ops
	for {
		select {
		case e := <-w.Events():
			switch e := e.(type) {
			case system.ClipboardEvent:
				lineEditor.SetText(e.Text)
			case system.DestroyEvent:
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e.Queue, e.Config, e.Size)
				for iconButton.Clicked() {
					w.WriteClipboard(lineEditor.Text())
				}
				for flatBtn.Clicked() {
					w.ReadClipboard()
				}
				kitchen(gtx, th)
				e.Frame(gtx.Ops)
			}
		case p := <-progressIncrementer:
			_ = p
			// progress += p
			// if progress > 100 {
			// 	progress = 0
			// }
			// w.Invalidate()
		}
	}
}

var (
	editor     = new(widget.Editor)
	lineEditor = &widget.Editor{
		SingleLine: true,
		Submit:     true,
	}
	button            = new(widget.Clickable)
	greenButton       = new(widget.Clickable)
	iconTextButton    = new(widget.Clickable)
	iconButton        = new(widget.Clickable)
	flatBtn           = new(widget.Clickable)
	radioButtonsGroup = new(widget.Enum)
	list              = &layout.List{
		Axis: layout.Vertical,
	}
	progress            = 0
	progressIncrementer chan int
	green               = true
	topLabel            = "Hello, Gio"
	icon                *widget.Icon
	checkbox            = new(widget.Bool)
	swtch               = new(widget.Bool)
)

type (
	D = layout.Dimensions
	C = layout.Context
)

func (b iconAndTextButton) Layout(gtx layout.Context) layout.Dimensions {
	return material.ButtonLayout(b.theme, b.button).Layout(gtx, func(gtx C) D {
		iconAndLabel := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}
		textIconSpacer := unit.Dp(5)

		layIcon := layout.Rigid(func(gtx C) D {
			return layout.Inset{Right: textIconSpacer}.Layout(gtx, func(gtx C) D {
				var d D
				if icon != nil {
					size := gtx.Px(unit.Dp(56)) - 2*gtx.Px(unit.Dp(16))
					b.icon.Layout(gtx, unit.Px(float32(size)))
					d = layout.Dimensions{
						Size: image.Point{X: size, Y: size},
					}
				}
				return d
			})
		})

		layLabel := layout.Rigid(func(gtx C) D {
			return layout.Inset{Left: textIconSpacer}.Layout(gtx, func(gtx C) D {
				l := material.Body1(b.theme, b.word)
				l.Color = b.theme.Color.InvText
				return l.Layout(gtx)
			})
		})

		return iconAndLabel.Layout(gtx, layIcon, layLabel)
	})
}

func kitchen(gtx layout.Context, th *material.Theme) layout.Dimensions {
	for _, e := range lineEditor.Events() {
		if e, ok := e.(widget.SubmitEvent); ok {
			topLabel = e.Text
			lineEditor.SetText("")
		}
	}
	widgets := []layout.Widget{
		material.H3(th, topLabel).Layout,
		func(gtx C) D {
			gtx.Constraints.Max.Y = gtx.Px(unit.Dp(200))
			return material.Editor(th, editor, "Hint").Layout(gtx)
		},
		func(gtx C) D {
			e := material.Editor(th, lineEditor, "Hint")
			e.Font.Style = text.Italic
			return e.Layout(gtx)
		},
		func(gtx C) D {
			in := layout.UniformInset(unit.Dp(8))
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return in.Layout(gtx, material.IconButton(th, iconButton, icon).Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return in.Layout(gtx, iconAndTextButton{theme: th, icon: icon, word: "Icon", button: iconTextButton}.Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return in.Layout(gtx, func(gtx C) D {
						for button.Clicked() {
							green = !green
						}
						return material.Button(th, button, "Click me!").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return in.Layout(gtx, func(gtx C) D {
						l := "Green"
						if !green {
							l = "Blue"
						}
						btn := material.Button(th, greenButton, l)
						if green {
							btn.Background = color.RGBA{A: 0xff, R: 0x9e, G: 0x9d, B: 0x24}
						}
						return btn.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return in.Layout(gtx, func(gtx C) D {
						return material.Clickable(gtx, flatBtn, func(gtx C) D {
							return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
								return layout.Center.Layout(gtx, material.Body1(th, "Flat").Layout)
							})
						})
					})
				}),
			)
		},
		material.ProgressBar(th, progress).Layout,
		func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(
					material.CheckBox(th, checkbox, "Checkbox").Layout,
				),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Left: unit.Dp(16)}.Layout(gtx,
						material.Switch(th, swtch).Layout,
					)
				}),
			)
		},
		func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(material.RadioButton(th, radioButtonsGroup, "r1", "RadioButton1").Layout),
				layout.Rigid(material.RadioButton(th, radioButtonsGroup, "r2", "RadioButton2").Layout),
				layout.Rigid(material.RadioButton(th, radioButtonsGroup, "r3", "RadioButton3").Layout),
			)
		},
	}

	return list.Layout(gtx, len(widgets), func(gtx C, i int) D {
		return layout.UniformInset(unit.Dp(16)).Layout(gtx, widgets[i])
	})
}

func (s *scaledConfig) Now() time.Time {
	return time.Now()
}

func (s *scaledConfig) Px(v unit.Value) int {
	scale := s.Scale
	if v.U == unit.UnitPx {
		scale = 1
	}
	return int(math.Round(float64(scale * v.V)))
}

const longText = `1. I learned from my grandfather, Verus, to use good manners, and to
put restraint on anger. 2. In the famous memory of my father I had a
pattern of modesty and manliness. 3. Of my mother I learned to be
pious and generous; to keep myself not only from evil deeds, but even
from evil thoughts; and to live with a simplicity which is far from
customary among the rich. 4. I owe it to my great-grandfather that I
did not attend public lectures and discussions, but had good and able
teachers at home; and I owe him also the knowledge that for things of
this nature a man should count no expense too great.

5. My tutor taught me not to favour either green or blue at the
chariot races, nor, in the contests of gladiators, to be a supporter
either of light or heavy armed. He taught me also to endure labour;
not to need many things; to serve myself without troubling others; not
to intermeddle in the affairs of others, and not easily to listen to
slanders against them.

6. Of Diognetus I had the lesson not to busy myself about vain things;
not to credit the great professions of such as pretend to work
wonders, or of sorcerers about their charms, and their expelling of
Demons and the like; not to keep quails (for fighting or divination),
nor to run after such things; to suffer freedom of speech in others,
and to apply myself heartily to philosophy. Him also I must thank for
my hearing first Bacchius, then Tandasis and Marcianus; that I wrote
dialogues in my youth, and took a liking to the philosopher's pallet
and skins, and to the other things which, by the Grecian discipline,
belong to that profession.

7. To Rusticus I owe my first apprehensions that my nature needed
reform and cure; and that I did not fall into the ambition of the
common Sophists, either by composing speculative writings or by
declaiming harangues of exhortation in public; further, that I never
strove to be admired by ostentation of great patience in an ascetic
life, or by display of activity and application; that I gave over the
study of rhetoric, poetry, and the graces of language; and that I did
not pace my house in my senatorial robes, or practise any similar
affectation. I observed also the simplicity of style in his letters,
particularly in that which he wrote to my mother from Sinuessa. I
learned from him to be easily appeased, and to be readily reconciled
with those who had displeased me or given cause of offence, so soon as
they inclined to make their peace; to read with care; not to rest
satisfied with a slight and superficial knowledge; nor quickly to
assent to great talkers. I have him to thank that I met with the
discourses of Epictetus, which he furnished me from his own library.

8. From Apollonius I learned true liberty, and tenacity of purpose; to
regard nothing else, even in the smallest degree, but reason always;
and always to remain unaltered in the agonies of pain, in the losses
of children, or in long diseases. He afforded me a living example of
how the same man can, upon occasion, be most yielding and most
inflexible. He was patient in exposition; and, as might well be seen,
esteemed his fine skill and ability in teaching others the principles
of philosophy as the least of his endowments. It was from him that I
learned how to receive from friends what are thought favours without
seeming humbled by the giver or insensible to the gift.`
