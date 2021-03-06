// SPDX-License-Identifier: Unlicense OR MIT

// +build darwin,!ios

package window

import (
	"errors"
	"image"
	"runtime"
	"sync"
	"time"
	"unicode"
	"unicode/utf16"
	"unsafe"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"

	_ "gioui.org/app/internal/cocoainit"
)

/*
#cgo CFLAGS: -DGL_SILENCE_DEPRECATION -Werror -Wno-deprecated-declarations -fmodules -fobjc-arc -x objective-c

#include <AppKit/AppKit.h>

#define GIO_MOUSE_MOVE 1
#define GIO_MOUSE_UP 2
#define GIO_MOUSE_DOWN 3

__attribute__ ((visibility ("hidden"))) void gio_main(CFTypeRef viewRef, const char *title, CGFloat width, CGFloat height);
__attribute__ ((visibility ("hidden"))) CGFloat gio_viewWidth(CFTypeRef viewRef);
__attribute__ ((visibility ("hidden"))) CGFloat gio_viewHeight(CFTypeRef viewRef);
__attribute__ ((visibility ("hidden"))) CGFloat gio_getViewBackingScale(CFTypeRef viewRef);
__attribute__ ((visibility ("hidden"))) CFTypeRef gio_readClipboard(void);
__attribute__ ((visibility ("hidden"))) void gio_writeClipboard(unichar *chars, NSUInteger length);
*/
import "C"

func init() {
	// Darwin requires that UI operations happen on the main thread only.
	runtime.LockOSThread()
}

type window struct {
	view        C.CFTypeRef
	w           Callbacks
	stage       system.Stage
	displayLink *displayLink

	// mu protect the following fields
	mu            sync.Mutex
	scale         float32
	width, height float32
}

// viewMap is the mapping from Cocoa NSViews to Go windows.
var viewMap sync.Map

var mainWindow = newWindowRendezvous()

var viewFactory func() C.CFTypeRef

// mustView is like lookoupView, except that it panics
// if the view isn't mapped.
func mustView(view C.CFTypeRef) *window {
	w, ok := lookupView(view)
	if !ok {
		panic("no window view view")
	}
	return w
}

func lookupView(view C.CFTypeRef) (*window, bool) {
	w, exists := viewMap.Load(view)
	if !exists {
		return nil, false
	}
	return w.(*window), true
}

func deleteView(view C.CFTypeRef) {
	viewMap.Delete(view)
}

func insertView(view C.CFTypeRef, w *window) {
	viewMap.Store(view, w)
}

func (w *window) contextView() C.CFTypeRef {
	return w.view
}

func (w *window) ReadClipboard() {
	runOnMain(func() {
		content := nsstringToString(C.gio_readClipboard())
		w.w.Event(system.ClipboardEvent{Text: content})
	})
}

func (w *window) WriteClipboard(s string) {
	u16 := utf16.Encode([]rune(s))
	runOnMain(func() {
		var chars *C.unichar
		if len(u16) > 0 {
			chars = (*C.unichar)(unsafe.Pointer(&u16[0]))
		}
		C.gio_writeClipboard(chars, C.NSUInteger(len(u16)))
	})
}

func (w *window) ShowTextInput(show bool) {}

func (w *window) SetAnimating(anim bool) {
	if anim {
		w.displayLink.Start()
	} else {
		w.displayLink.Stop()
	}
}

func (w *window) setStage(stage system.Stage) {
	if stage == w.stage {
		return
	}
	w.stage = stage
	w.w.Event(system.StageEvent{Stage: stage})
}

//export gio_onKeys
func gio_onKeys(view C.CFTypeRef, cstr *C.char, ti C.double, mods C.NSUInteger) {
	str := C.GoString(cstr)
	kmods := convertMods(mods)
	w := mustView(view)
	for _, k := range str {
		if n, ok := convertKey(k); ok {
			w.w.Event(key.Event{
				Name:      n,
				Modifiers: kmods,
			})
		}
	}
}

//export gio_onText
func gio_onText(view C.CFTypeRef, cstr *C.char) {
	str := C.GoString(cstr)
	w := mustView(view)
	w.w.Event(key.EditEvent{Text: str})
}

//export gio_onMouse
func gio_onMouse(view C.CFTypeRef, cdir C.int, cbtns C.NSUInteger, x, y, dx, dy C.CGFloat, ti C.double, mods C.NSUInteger) {
	var typ pointer.Type
	switch cdir {
	case C.GIO_MOUSE_MOVE:
		typ = pointer.Move
	case C.GIO_MOUSE_UP:
		typ = pointer.Release
	case C.GIO_MOUSE_DOWN:
		typ = pointer.Press
	default:
		panic("invalid direction")
	}
	var btns pointer.Buttons
	if cbtns&(1<<0) != 0 {
		btns |= pointer.ButtonLeft
	}
	if cbtns&(1<<1) != 0 {
		btns |= pointer.ButtonRight
	}
	if cbtns&(1<<2) != 0 {
		btns |= pointer.ButtonMiddle
	}
	t := time.Duration(float64(ti)*float64(time.Second) + .5)
	w := mustView(view)
	xf, yf := float32(x)*w.scale, float32(y)*w.scale
	dxf, dyf := float32(dx)*w.scale, float32(dy)*w.scale
	w.w.Event(pointer.Event{
		Type:      typ,
		Source:    pointer.Mouse,
		Time:      t,
		Buttons:   btns,
		Position:  f32.Point{X: xf, Y: yf},
		Scroll:    f32.Point{X: dxf, Y: dyf},
		Modifiers: convertMods(mods),
	})
}

//export gio_onDraw
func gio_onDraw(view C.CFTypeRef) {
	w := mustView(view)
	scale := float32(C.gio_getViewBackingScale(w.view))
	width, height := float32(C.gio_viewWidth(w.view)), float32(C.gio_viewHeight(w.view))
	w.mu.Lock()
	w.scale = scale
	w.width, w.height = width, height
	w.mu.Unlock()
	w.draw(true)
}

//export gio_onFocus
func gio_onFocus(view C.CFTypeRef, focus C.BOOL) {
	w := mustView(view)
	w.w.Event(key.FocusEvent{Focus: focus == C.YES})
}

//export gio_onChangeScreen
func gio_onChangeScreen(view C.CFTypeRef, did uint64) {
	w := mustView(view)
	w.displayLink.SetDisplayID(did)
}

func (w *window) draw(sync bool) {
	w.mu.Lock()
	wf, hf, scale := w.width, w.height, w.scale
	w.mu.Unlock()
	if wf == 0 || hf == 0 {
		return
	}
	width := int(wf*scale + .5)
	height := int(hf*scale + .5)
	cfg := configFor(scale)
	cfg.now = time.Now()
	w.setStage(system.StageRunning)
	w.w.Event(FrameEvent{
		FrameEvent: system.FrameEvent{
			Size: image.Point{
				X: width,
				Y: height,
			},
			Config: &cfg,
		},
		Sync: sync,
	})
}

func configFor(scale float32) config {
	return config{
		pxPerDp: scale,
		pxPerSp: scale,
	}
}

//export gio_onTerminate
func gio_onTerminate(view C.CFTypeRef) {
	w := mustView(view)
	w.displayLink.Close()
	deleteView(view)
	w.w.Event(system.DestroyEvent{})
}

//export gio_onHide
func gio_onHide(view C.CFTypeRef) {
	w := mustView(view)
	w.setStage(system.StagePaused)
}

//export gio_onShow
func gio_onShow(view C.CFTypeRef) {
	w := mustView(view)
	w.setStage(system.StageRunning)
}

//export gio_onCreate
func gio_onCreate(view C.CFTypeRef) {
	scale := float32(C.gio_getViewBackingScale(view))
	w := &window{
		view:  view,
		scale: scale,
	}
	dl, err := NewDisplayLink(func() {
		w.draw(false)
	})
	if err != nil {
		panic(err)
	}
	w.displayLink = dl
	wopts := <-mainWindow.out
	w.w = wopts.window
	w.w.SetDriver(w)
	insertView(view, w)
}

func NewWindow(win Callbacks, opts *Options) error {
	mainWindow.in <- windowAndOptions{win, opts}
	return <-mainWindow.errs
}

func Main() {
	wopts := <-mainWindow.out
	view := viewFactory()
	if view == 0 {
		// TODO: return this error from CreateWindow.
		panic(errors.New("CreateWindow: failed to create view"))
	}
	// Window sizes is in unscaled screen coordinates, not device pixels.
	cfg := configFor(1.0)
	opts := wopts.opts
	w := cfg.Px(opts.Width)
	h := cfg.Px(opts.Height)
	w = int(float32(w))
	h = int(float32(h))
	title := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(title))
	C.gio_main(view, title, C.CGFloat(w), C.CGFloat(h))
}

func convertKey(k rune) (string, bool) {
	var n string
	switch k {
	case 0x1b:
		n = key.NameEscape
	case C.NSLeftArrowFunctionKey:
		n = key.NameLeftArrow
	case C.NSRightArrowFunctionKey:
		n = key.NameRightArrow
	case C.NSUpArrowFunctionKey:
		n = key.NameUpArrow
	case C.NSDownArrowFunctionKey:
		n = key.NameDownArrow
	case 0xd:
		n = key.NameReturn
	case 0x3:
		n = key.NameEnter
	case C.NSHomeFunctionKey:
		n = key.NameHome
	case C.NSEndFunctionKey:
		n = key.NameEnd
	case 0x7f:
		n = key.NameDeleteBackward
	case C.NSDeleteFunctionKey:
		n = key.NameDeleteForward
	case C.NSPageUpFunctionKey:
		n = key.NamePageUp
	case C.NSPageDownFunctionKey:
		n = key.NamePageDown
	case C.NSF1FunctionKey:
		n = "F1"
	case C.NSF2FunctionKey:
		n = "F2"
	case C.NSF3FunctionKey:
		n = "F3"
	case C.NSF4FunctionKey:
		n = "F4"
	case C.NSF5FunctionKey:
		n = "F5"
	case C.NSF6FunctionKey:
		n = "F6"
	case C.NSF7FunctionKey:
		n = "F7"
	case C.NSF8FunctionKey:
		n = "F8"
	case C.NSF9FunctionKey:
		n = "F9"
	case C.NSF10FunctionKey:
		n = "F10"
	case C.NSF11FunctionKey:
		n = "F11"
	case C.NSF12FunctionKey:
		n = "F12"
	case 0x09, 0x19:
		n = key.NameTab
	case 0x20:
		n = "Space"
	default:
		k = unicode.ToUpper(k)
		if !unicode.IsPrint(k) {
			return "", false
		}
		n = string(k)
	}
	return n, true
}

func convertMods(mods C.NSUInteger) key.Modifiers {
	var kmods key.Modifiers
	if mods&C.NSAlternateKeyMask != 0 {
		kmods |= key.ModAlt
	}
	if mods&C.NSControlKeyMask != 0 {
		kmods |= key.ModCtrl
	}
	if mods&C.NSCommandKeyMask != 0 {
		kmods |= key.ModCommand
	}
	if mods&C.NSShiftKeyMask != 0 {
		kmods |= key.ModShift
	}
	return kmods
}
