// SPDX-License-Identifier: Unlicense OR MIT

package layout

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
)

type scrollChild struct {
	index int
	size  image.Point
	call  op.CallOp
}

// List displays a subsection of a potentially infinitely
// large underlying list. List accepts user input to scroll
// the subsection.
type List struct {
	Axis Axis
	// ScrollToEnd instructs the list to stay scrolled to the far end position
	// once reached. A List with ScrollToEnd == true and Position.BeforeEnd ==
	// false draws its content with the last item at the bottom of the list
	// area.
	ScrollToEnd bool
	// Alignment is the cross axis alignment of list elements.
	Alignment Alignment

	ctx         Context
	macro       op.MacroOp
	child       op.MacroOp
	scroll      gesture.Scroll
	scrollDelta int

	// Position is updated during Layout. To save the list scroll position,
	// just save Position after Layout finishes. To scroll the list
	// programatically, update Position (e.g. restore it from a saved value)
	// before calling Layout.
	Position Position

	doScrollTo bool
	scrollTo   int
	fromEnd    bool

	len        int
	FirstDrawn int
	lastDrawn  int

	// maxSize is the total size of visible children.
	maxSize  int
	children []scrollChild
	dir      iterationDir

	// height is the height in pixels at the last layout, used for PgUp & PgDn.
	height int
}

// ListElement is a function that computes the dimensions of
// a list element.
type ListElement func(gtx Context, index int) Dimensions

type iterationDir uint8

// Position is a List scroll offset represented as an offset from the top edge
// of a child element.
type Position struct {
	// BeforeEnd tracks whether the List position is before the very end. We
	// use "before end" instead of "at end" so that the zero value of a
	// Position struct is useful.
	//
	// When laying out a list, if ScrollToEnd is true and BeforeEnd is false,
	// then First and Offset are ignored, and the list is drawn with the last
	// item at the bottom. If ScrollToEnd is false then BeforeEnd is ignored.
	BeforeEnd bool
	// First is the index of the first visible child.
	First int
	// Offset is the distance in pixels from the top edge to the child at index
	// First.  Positive offsets are above the edge.
	Offset int
}

const (
	iterateNone iterationDir = iota
	iterateForward
	iterateBackward
)

const inf = 1e6

// init prepares the list for iterating through its children with next.
func (l *List) init(gtx Context, len int) {
	if l.more() {
		panic("unfinished child")
	}
	l.ctx = gtx
	l.maxSize = 0
	l.children = l.children[:0]
	l.len = len
	l.update()
	if l.doScrollTo {
		if l.scrollTo >= len {
			l.scrollTo = len - 1
		}
		// If l.scrollTo is already in view, do nothing.
		if l.FirstDrawn < l.scrollTo &&
			l.scrollTo < l.LastDrawn {
			l.doScrollTo = false
		} else {
			l.Position.Offset = 0
			l.Position.First = l.scrollTo
			if l.LastDrawn > 0 && l.scrollTo >= l.LastDrawn {
				l.fromEnd = true
				l.Position.First++
			} else {
				l.fromEnd = false
			}
		}
	} else if l.scrollToEnd() || l.Position.First > len {
		l.Position.Offset = 0
		l.Position.First = len
	}
	l.macro = op.Record(gtx.Ops)
	l.next()
}

// Layout the List.
func (l *List) Layout(gtx Context, len int, w ListElement) Dimensions {
	for l.init(gtx, len); l.more(); l.next() {
		crossMin, crossMax := axisCrossConstraint(l.Axis, l.ctx.Constraints)
		cs := axisConstraints(l.Axis, 0, inf, crossMin, crossMax)
		i := l.index()
		gtx.Constraints = cs
		l.end(i, w(gtx, i))
	}
	return l.layout()
}

func (l *List) scrollToEnd() bool {
	return (l.doScrollTo && l.fromEnd) || (l.ScrollToEnd && !l.Position.BeforeEnd)
}

// Dragging reports whether the List is being dragged.
func (l *List) Dragging() bool {
	return l.scroll.State() == gesture.StateDragging
}

// ScrollTo makes sure list index item i is in view.  If it's above the top,
// it becomes the top item.  If it's below the bottom, it becomes the bottom
// item.  If i < 0, uses 0.  If you ScrollTo(n) and then later layout a list
// shorter than n, Layout scrolls to the end of the list.
func (l *List) ScrollTo(i int) {
	l.doScrollTo = true
	if i < 0 {
		i = 0
	}
	l.scrollTo = i
}

func (l *List) PageUp() {
	l.Position.Offset -= l.height
	// If you don't do this and l.ScrollToEnd == true, Position.Offset is
	// ignored.
	l.Position.BeforeEnd = true
}

func (l *List) PageDown() {
	l.Position.Offset += l.height
	l.Position.BeforeEnd = true
}

func (l *List) update() {
	d := l.scroll.Scroll(l.ctx, l.ctx, l.ctx.Now(), gesture.Axis(l.Axis))
	l.scrollDelta = d
	l.Position.Offset += d
}

// next advances to the next child.
func (l *List) next() {
	l.dir = l.nextDir()
	// The user scroll offset is applied after scrolling to
	// list end.
	if l.scrollToEnd() && !l.more() && l.scrollDelta < 0 {
		l.Position.BeforeEnd = true
		l.Position.Offset += l.scrollDelta
		l.dir = l.nextDir()
	}
	if l.more() {
		l.child = op.Record(l.ctx.Ops)
	}
}

// index is current child's position in the underlying list.
func (l *List) index() int {
	switch l.dir {
	case iterateBackward:
		return l.Position.First - 1
	case iterateForward:
		return l.Position.First + len(l.children)
	default:
		panic("Index called before Next")
	}
}

// more reports whether more children are needed.
func (l *List) more() bool {
	return l.dir != iterateNone
}

func (l *List) nextDir() iterationDir {
	_, vsize := axisMainConstraint(l.Axis, l.ctx.Constraints)
	last := l.Position.First + len(l.children)
	// Clamp offset.
	if l.maxSize-l.Position.Offset < vsize &&
		(last == l.len ||
			(l.doScrollTo &&
				l.fromEnd &&
				last == l.scrollTo+1)) {
		l.Position.Offset = l.maxSize - vsize
	}
	if l.Position.Offset < 0 && l.Position.First == 0 {
		l.Position.Offset = 0
	}
	switch {
	case len(l.children) == l.len:
		return iterateNone
	case l.maxSize-l.Position.Offset < vsize:
		return iterateForward
	case l.Position.Offset < 0:
		return iterateBackward
	}
	return iterateNone
}

// End the current child by specifying its dimensions.
func (l *List) end(i int, dims Dimensions) {
	call := l.child.Stop()
	child := scrollChild{i, dims.Size, call}
	mainSize := axisMain(l.Axis, child.size)
	l.maxSize += mainSize
	switch l.dir {
	case iterateForward:
		l.children = append(l.children, child)
	case iterateBackward:
		l.children = append([]scrollChild{child}, l.children...)
		l.Position.First--
		l.Position.Offset += mainSize
	default:
		panic("call Next before End")
	}
	l.dir = iterateNone
}

// Layout the List and return its dimensions.
func (l *List) layout() Dimensions {
	if l.more() {
		panic("unfinished child")
	}
	mainMin, mainMax := axisMainConstraint(l.Axis, l.ctx.Constraints)
	children := l.children
	// Skip invisible children
	for len(children) > 0 {
		sz := children[0].size
		mainSize := axisMain(l.Axis, sz)
		if l.Position.Offset <= mainSize {
			break
		}
		l.Position.First++
		l.Position.Offset -= mainSize
		children = children[1:]
	}
	size := -l.Position.Offset
	var maxCross int
	for i, child := range children {
		sz := child.size
		if c := axisCross(l.Axis, sz); c > maxCross {
			maxCross = c
		}
		size += axisMain(l.Axis, sz)
		if size >= mainMax {
			children = children[:i+1]
			break
		}
	}
	ops := l.ctx.Ops
	pos := -l.Position.Offset
	// ScrollToEnd lists are end aligned.
	if space := mainMax - size; l.ScrollToEnd && space > 0 {
		pos += space
	}
	if len(children) > 0 {
		l.FirstDrawn = children[0].index
		l.LastDrawn = children[len(children)-1].index
	}
	for _, child := range children {
		sz := child.size
		var cross int
		switch l.Alignment {
		case End:
			cross = maxCross - axisCross(l.Axis, sz)
		case Middle:
			cross = (maxCross - axisCross(l.Axis, sz)) / 2
		}
		childSize := axisMain(l.Axis, sz)
		max := childSize + pos
		if max > mainMax {
			max = mainMax
		}
		min := pos
		if min < 0 {
			min = 0
		}
		r := image.Rectangle{
			Min: axisPoint(l.Axis, min, -inf),
			Max: axisPoint(l.Axis, max, inf),
		}
		stack := op.Push(ops)
		clip.Rect{Rect: FRect(r)}.Op(ops).Add(ops)
		op.TransformOp{}.Offset(FPt(axisPoint(l.Axis, pos, cross))).Add(ops)
		child.call.Add(ops)
		stack.Pop()
		pos += childSize
	}
	atStart := l.Position.First == 0 && l.Position.Offset <= 0
	atEnd := l.Position.First+len(children) == l.len && mainMax >= pos
	if atStart && l.scrollDelta < 0 || atEnd && l.scrollDelta > 0 {
		l.scroll.Stop()
	}
	l.Position.BeforeEnd = !atEnd
	if pos < mainMin {
		pos = mainMin
	}
	if pos > mainMax {
		pos = mainMax
	}
	dims := axisPoint(l.Axis, pos, maxCross)
	call := l.macro.Stop()
	defer op.Push(l.ctx.Ops).Pop()
	pointer.Rect(image.Rectangle{Max: dims}).Add(ops)
	l.scroll.Add(ops)
	call.Add(ops)
	l.doScrollTo = false
	l.height = dims.Y
	return Dimensions{Size: dims}
}
