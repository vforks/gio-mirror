// SPDX-License-Identifier: Unlicense OR MIT

// Package system contains events usually handled at the top-level
// program level.
package system

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/op"
	"gioui.org/unit"
)

// A FrameEvent requests a new frame in the form of a list of
// operations that describes what to display and how to handle
// input.
type FrameEvent struct {
	Config Config
	// Size is the dimensions of the window.
	Size image.Point
	// Insets is the insets to apply.
	Insets Insets
	// Frame is the callback to supply the list of
	// operations to complete the FrameEvent.
	//
	// Note that the operation list and the operations themselves
	// may not be mutated until another FrameEvent is received from
	// the same event source.
	// That means that calls to frame.Reset and changes to referenced
	// data such as ImageOp backing images should happen between
	// receiving a FrameEvent and calling Frame.
	//
	// Example:
	//
	//  var w *app.Window
	//  var frame *op.Ops
	//  for e := range w.Events() {
	//      if e, ok := e.(system.FrameEvent); ok {
	//          // Call frame.Reset and manipulate images for ImageOps
	//          // here.
	//          e.Frame(frame)
	//      }
	//  }
	Frame func(frame *op.Ops)
	// Queue supplies the events for event handlers.
	Queue event.Queue
}

// Config defines the essential properties of
// the environment.
type Config interface {
	// Now returns the current animation time.
	Now() time.Time

	unit.Converter
}

// DestroyEvent is the last event sent through
// a window event channel.
type DestroyEvent struct {
	// Err is nil for normal window closures. If a
	// window is prematurely closed, Err is the cause.
	Err error
}

// ClipboardEvent is sent once for each request for the
// clipboard content.
type ClipboardEvent struct {
	Text string
}

// Insets is the space taken up by
// system decoration such as translucent
// system bars and software keyboards.
type Insets struct {
	Top, Bottom, Left, Right unit.Value
}

// A StageEvent is generated whenever the stage of a
// Window changes.
type StageEvent struct {
	Stage Stage
}

// CommandEvent is a system event.
type CommandEvent struct {
	Type CommandType
	// Suppress the default action of the command.
	Cancel bool
}

// Stage of a Window.
type Stage uint8

// CommandType is the type of a CommandEvent.
type CommandType uint8

const (
	// StagePaused is the Stage for inactive Windows.
	// Inactive Windows don't receive FrameEvents.
	StagePaused Stage = iota
	// StateRunning is for active Windows.
	StageRunning
)

const (
	// CommandBack is the command for a back action
	// such as the Android back button.
	CommandBack CommandType = iota
)

func (l Stage) String() string {
	switch l {
	case StagePaused:
		return "StagePaused"
	case StageRunning:
		return "StageRunning"
	default:
		panic("unexpected Stage value")
	}
}

func (FrameEvent) ImplementsEvent()     {}
func (StageEvent) ImplementsEvent()     {}
func (*CommandEvent) ImplementsEvent()  {}
func (DestroyEvent) ImplementsEvent()   {}
func (ClipboardEvent) ImplementsEvent() {}
