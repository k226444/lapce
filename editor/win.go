package editor

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

// Frame is
type Frame struct {
	vertical bool
	width    int
	height   int
	x        int
	y        int
	editor   *Editor
	children []*Frame
	parent   *Frame
	win      *Window
}

func (f *Frame) split(vertical bool) {
	if f.hasChildren() {
		fmt.Println("split has children already")
		return
	}
	win := f.win
	if win == nil {
		return
	}
	newFrame := &Frame{editor: f.editor}
	newWin := NewWindow(win.editor, newFrame)
	newWin.loadBuffer(win.buffer)
	newWin.row = win.row
	newWin.col = win.col
	newWin.cursorX = win.cursorX
	newWin.cursorY = win.cursorY

	parent := f.parent
	if parent != nil && parent.vertical == vertical {
		newFrame.parent = parent
		children := []*Frame{}
		for _, child := range parent.children {
			if child == f {
				children = append(children, child)
				children = append(children, newFrame)
			} else {
				children = append(children, child)
			}
		}
		parent.children = children
	} else {
		newFrame.parent = f
		frame := &Frame{
			parent: f,
			win:    win,
			editor: f.editor,
		}
		win.frame = frame
		f.children = []*Frame{}
		f.vertical = vertical
		f.win = nil
		f.children = append(f.children, frame, newFrame)
	}
	win.editor.equalWins()
	newWin.view.HorizontalScrollBar().SetValue(win.view.HorizontalScrollBar().Value())
	newWin.view.VerticalScrollBar().SetValue(win.view.VerticalScrollBar().Value())
	newWin.scrollto(newWin.col, newWin.row, false)
	f.setFocus()
}

func (f *Frame) hasChildren() bool {
	return f.children != nil && len(f.children) > 0
}

func (f *Frame) setPos(x, y int) {
	f.x = x
	f.y = y
	if !f.hasChildren() {
		fmt.Println("set x y", x, y)
		return
	}

	for _, child := range f.children {
		child.setPos(x, y)
		if f.vertical {
			x += child.width
		} else {
			y += child.height
		}
	}
}

func (f *Frame) setSize(vertical bool, singleValue int) {
	if !f.hasChildren() {
		if vertical {
			f.width = singleValue
		} else {
			f.height = singleValue
		}
		return
	}

	max := f.countSplits(vertical)
	if vertical {
		f.width = max * singleValue
	} else {
		f.height = max * singleValue
	}

	if f.vertical == vertical {
		for _, child := range f.children {
			child.setSize(vertical, singleValue)
		}
		return
	}

	for _, child := range f.children {
		n := child.countSplits(vertical)
		child.setSize(vertical, singleValue*max/n)
	}
}

func (f *Frame) focusAbove() {
	editor := f.editor
	editor.winsRWMutext.RLock()
	defer editor.winsRWMutext.RUnlock()

	for _, win := range editor.wins {
		y := f.y - (win.frame.y + win.frame.height)
		x1 := f.x - win.frame.x
		x2 := f.x - (win.frame.x + win.frame.width)
		if y < 1 && y >= 0 && x1 >= 0 && x2 < 0 {
			win.frame.setFocus()
			return
		}
	}
}

func (f *Frame) focusBelow() {
	editor := f.editor
	editor.winsRWMutext.RLock()
	defer editor.winsRWMutext.RUnlock()

	for _, win := range editor.wins {
		y := win.frame.y - (f.y + f.height)
		x1 := f.x - win.frame.x
		x2 := f.x - (win.frame.x + win.frame.width)
		if y < 1 && y >= 0 && x1 >= 0 && x2 < 0 {
			win.frame.setFocus()
			return
		}
	}
}

func (f *Frame) focusLeft() {
	editor := f.editor
	editor.winsRWMutext.RLock()
	defer editor.winsRWMutext.RUnlock()

	for _, win := range editor.wins {
		x := f.x - (win.frame.x + win.frame.width)
		y1 := f.y - win.frame.y
		y2 := f.y - (win.frame.y + win.frame.height)
		if x < 1 && x >= 0 && y1 >= 0 && y2 < 0 {
			win.frame.setFocus()
			return
		}
	}
}

func (f *Frame) focusRight() {
	editor := f.editor
	editor.winsRWMutext.RLock()
	defer editor.winsRWMutext.RUnlock()

	for _, win := range editor.wins {
		x := win.frame.x - (f.x + f.width)
		y1 := f.y - win.frame.y
		y2 := f.y - (win.frame.y + win.frame.height)
		if x < 1 && x >= 0 && y1 >= 0 && y2 < 0 {
			win.frame.setFocus()
			return
		}
	}
}

func (f *Frame) exchange() {
	parent := f.parent
	if parent == nil {
		return
	}
	if len(parent.children) == 1 {
		parent.exchange()
		return
	}
	i := 0
	for index, child := range parent.children {
		if child == f {
			i = index
			break
		}
	}

	if i == len(parent.children)-1 {
		parent.children[i], parent.children[i-1] = parent.children[i-1], parent.children[i]
	} else {
		parent.children[i], parent.children[i+1] = parent.children[i+1], parent.children[i]
	}
	f.editor.equalWins()
	parent.children[i].setFocus()
}

func (f *Frame) setFocus() {
	if f.hasChildren() {
		f.children[0].setFocus()
		return
	}
	w := f.win
	w.view.SetFocus2()
	f.editor.activeWin = f.win
	f.editor.cursor.SetParent(f.win.view)
	f.editor.cursor.Move2(w.cursorX, w.cursorY)
	f.editor.cursor.Hide()
	f.editor.cursor.Show()
	w.buffer.xiView.Click(w.row, w.col)
}

func (f *Frame) close() *Frame {
	if f.hasChildren() {
		return nil
	}
	if f.parent == nil {
		return nil
	}
	parent := f.parent
	children := []*Frame{}
	i := 0
	for index, child := range parent.children {
		if child != f {
			children = append(children, child)
		} else {
			i = index
		}
	}
	var newFocus *Frame
	parent.children = children
	if len(children) == 0 {
		newFocus = parent.close()
	} else {
		if i > 0 {
			i--
		}
		newFocus = children[i]
	}
	win := f.win
	if win == nil {
		return newFocus
	}
	editor := win.editor
	editor.winsRWMutext.Lock()
	delete(editor.wins, win.id)
	editor.winsRWMutext.Unlock()
	win.widget.Hide()
	editor.equalWins()
	if newFocus != nil {
		newFocus.setFocus()
	}
	return newFocus
}

func (f *Frame) countSplits(vertical bool) int {
	if !f.hasChildren() {
		return 1
	}
	n := 0
	if f.vertical == vertical {
		for _, child := range f.children {
			n += child.countSplits(vertical)
		}
	} else {
		for _, child := range f.children {
			v := child.countSplits(vertical)
			if v > n {
				n = v
			}
		}
	}
	return n
}

// Window is for displaying a buffer
type Window struct {
	id                        int
	editor                    *Editor
	widget                    *widgets.QWidget
	gutter                    *widgets.QWidget
	gutterWidth               int
	gutterPadding             int
	view                      *widgets.QGraphicsView
	cline                     *widgets.QWidget
	frame                     *Frame
	buffer                    *Buffer
	x                         int
	y                         int
	cursorX                   int
	cursorY                   int
	row                       int
	col                       int
	scrollCol                 int
	start                     int
	end                       int
	scrollChan                chan *Scroll
	scrollWaitChan            chan *SmoothScroll
	scrolling                 bool
	scrollDone                chan struct{}
	scrollMutex               sync.Mutex
	verticalScrollBar         *widgets.QScrollBar
	horizontalScrollBar       *widgets.QScrollBar
	verticalScrollBarWidth    int
	horizontalScrollBarHeight int
	verticalScrollValue       int
	horizontalScrollValue     int
}

// NewWindow creates a new window
func NewWindow(editor *Editor, frame *Frame) *Window {
	editor.winsRWMutext.Lock()
	w := &Window{
		id:             editor.winIndex,
		editor:         editor,
		frame:          frame,
		view:           widgets.NewQGraphicsView(nil),
		cline:          widgets.NewQWidget(nil, 0),
		widget:         widgets.NewQWidget(nil, 0),
		gutter:         widgets.NewQWidget(nil, 0),
		scrollChan:     make(chan *Scroll),
		scrollDone:     make(chan struct{}),
		scrollWaitChan: make(chan *SmoothScroll),
		gutterPadding:  10,
	}
	go w.smoothScrollJob()

	layout := widgets.NewQHBoxLayout()
	layout.SetContentsMargins(0, 0, 0, 0)
	layout.SetSpacing(0)
	layout.AddWidget(w.gutter, 0, 0)
	layout.AddWidget(w.view, 1, 0)
	w.widget.SetContentsMargins(0, 0, 1, 1)
	w.widget.SetLayout(layout)
	w.gutter.SetFixedWidth(30)
	w.gutter.ConnectPaintEvent(w.paintGutter)

	w.view.ConnectEventFilter(func(watched *core.QObject, event *core.QEvent) bool {
		if event.Type() == core.QEvent__MouseButtonPress {
			mousePress := gui.NewQMouseEventFromPointer(event.Pointer())
			w.view.MousePressEvent(mousePress)
			return true
		}
		return w.view.EventFilterDefault(watched, event)
	})
	w.cline.SetParent(w.view)
	w.cline.SetStyleSheet(editor.getClineStylesheet())
	w.cline.SetFocusPolicy(core.Qt__NoFocus)
	w.cline.InstallEventFilter(w.view)
	w.cline.ConnectWheelEvent(func(event *gui.QWheelEvent) {
		w.view.WheelEventDefault(event)
	})
	frame.win = w
	editor.winIndex++
	editor.wins[w.id] = w
	editor.winsRWMutext.Unlock()

	// w.view.SetFrameShape(widgets.QFrame__NoFrame)
	w.view.ConnectMousePressEvent(func(event *gui.QMouseEvent) {
		editor.activeWin = w
		editor.cursor.SetParent(w.view)
		w.view.MousePressEventDefault(event)
	})
	w.view.ConnectKeyPressEvent(func(event *gui.QKeyEvent) {
		if w.buffer == nil {
			return
		}
		state, ok := editor.vimStates[editor.vimMode]
		if !ok {
			return
		}

		key := editor.convertKey(event)
		state.setCmd(key)
		state.execute()
	})
	w.view.ConnectScrollContentsBy(func(dx, dy int) {
		w.view.ScrollContentsByDefault(dx, dy)
		w.verticalScrollValue = w.verticalScrollBar.Value()
		w.horizontalScrollValue = w.horizontalScrollBar.Value()
	})
	w.view.SetFocusPolicy(core.Qt__ClickFocus)
	w.view.SetAlignment(core.Qt__AlignLeft | core.Qt__AlignTop)
	w.view.SetCornerWidget(widgets.NewQWidget(nil, 0))
	w.view.SetFrameStyle(0)
	w.horizontalScrollBar = w.view.HorizontalScrollBar()
	w.verticalScrollBar = w.view.VerticalScrollBar()
	w.widget.SetObjectName("view")
	if editor.theme != nil {
		scrollBarStyleSheet := editor.getScrollbarStylesheet()
		w.widget.SetStyleSheet(scrollBarStyleSheet)
		w.verticalScrollBarWidth = w.verticalScrollBar.Width()
		w.horizontalScrollBarHeight = w.horizontalScrollBar.Height()
	}
	w.widget.SetParent(editor.centralWidget)

	return w
}

func (w *Window) update() {
	w.start = int(float64(w.view.VerticalScrollBar().Value()) / w.buffer.font.lineHeight)
	w.end = w.start + int(float64(w.frame.height)/w.buffer.font.lineHeight+1)
	b := w.buffer
	for i := w.start; i <= w.end; i++ {
		if i >= len(b.lines) {
			break
		}
		if b.lines[i] != nil && b.lines[i].invalid {
			b.lines[i].invalid = false
			b.updateLine(i)
		}
	}
	w.gutterWidth = int(b.font.fontMetrics.Width(strconv.Itoa(len(b.lines)))+0.5) + w.gutterPadding*2
	w.gutter.SetFixedWidth(w.gutterWidth)
	// w.gutter.Update()
}

func (w *Window) charUnderCursor() rune {
	for _, r := range w.buffer.lines[w.row].text[w.col:] {
		return r
	}
	return 0
}

func utfClass(r rune) int {
	if unicode.IsSpace(r) {
		return 0
	}
	if unicode.IsPunct(r) || unicode.IsMark(r) || unicode.IsSymbol(r) {
		return 1
	}
	return 2
}

func (w *Window) wordUnderCursor() string {
	if w.buffer.lines[w.row] == nil {
		return ""
	}
	runeSlice := []rune{}
	nonWordRuneSlice := []rune{}
	stopNonWord := false
	text := w.buffer.lines[w.row].text[w.col:]
	class := 0
	for i, r := range text {
		c := utfClass(r)
		if i == 0 {
			class = c
		}
		if c == 2 {
			runeSlice = append(runeSlice, r)
		} else {
			if len(runeSlice) > 0 {
				break
			}
			if c == 0 {
				if len(nonWordRuneSlice) > 0 {
					stopNonWord = true
				}
			} else {
				if !stopNonWord {
					nonWordRuneSlice = append(nonWordRuneSlice, r)
				}
			}
		}
	}
	if len(runeSlice) == 0 {
		if class == 1 {
			text = w.buffer.lines[w.row].text[:w.col]
			textRune := []rune(text)
			for i := len(textRune) - 1; i >= 0; i-- {
				r := textRune[i]
				c := utfClass(r)
				if c == 1 {
					nonWordRuneSlice = append([]rune{r}, nonWordRuneSlice...)
				} else {
					break
				}
			}
		}
		return string(nonWordRuneSlice)
	}

	if class == 2 {
		text = w.buffer.lines[w.row].text[:w.col]
		textRune := []rune(text)
		for i := len(textRune) - 1; i >= 0; i-- {
			r := textRune[i]
			c := utfClass(r)
			if c == 2 {
				runeSlice = append([]rune{r}, runeSlice...)
			} else {
				break
			}
		}
	}

	return string(runeSlice)
}

func (w *Window) wordEnd() {
	class := 0
	i := 0
	j := 0
	for {
		if w.buffer.lines[w.row] == nil {
			return
		}
		text := w.buffer.lines[w.row].text[w.col:]
		var r rune
		hasBreak := false
		for i, r = range text {
			if j == 0 && i == 0 {
				class = utfClass(r)
				continue
			}
			c := utfClass(r)
			if j == 0 && i == 1 {
				class = c
				continue
			}
			if c == class {
				continue
			}
			if class == 0 {
				class = c
				continue
			}
			hasBreak = true
			break
		}
		if hasBreak {
			w.col += i - 1
			return
		}
		if w.row == len(w.buffer.lines)-1 {
			return
		}
		w.row++
		w.col = 0
		j++
	}
}

func (w *Window) wordForward() {
	class := 0
	j := 0
	for {
		if w.buffer.lines[w.row] == nil {
			return
		}
		if j > 0 {
			w.col = len(w.buffer.lines[w.row].text) - 1
		}
		text := w.buffer.lines[w.row].text[:w.col]
		runeSlice := []rune(text)
		var r rune
		hasBreak := false
		i := -1
		for index := len(runeSlice) - 1; index >= 0; index-- {
			i++
			r = runeSlice[index]
			if j == 0 && i == 0 {
				class = utfClass(r)
				continue
			}
			c := utfClass(r)
			if j == 0 && i == 1 {
				class = c
				continue
			}
			if c == class {
				continue
			}
			if class == 0 {
				class = c
				continue
			}
			hasBreak = true
			break
		}
		if hasBreak {
			w.col -= i
			return
		}
		if len(runeSlice) > 0 && utfClass(runeSlice[0]) > 0 {
			w.col = 0
			return
		}
		if w.row == 0 {
			return
		}
		w.row--
		j++
	}
}

func (w *Window) updateCline() {
	w.cline.Move2(0, w.y)
}

func (w *Window) updateCursor() {
	if w.editor.activeWin != w {
		return
	}
	w.editor.updateCursorShape()
	cursor := w.editor.cursor
	cursor.Move2(w.x, w.y)
	cursor.Hide()
	cursor.Show()
}

func (w *Window) setScroll() {
	w.update()
	w.updateCursorPos()
	w.updateCursor()
	w.updateCline()
	w.buffer.xiView.Scroll(w.start, w.end)
}

func (w *Window) loadBuffer(buffer *Buffer) {
	w.buffer = buffer
	w.view.SetScene(buffer.scence)
}

func (w *Window) updateCursorPos() {
	w.cursorX = w.x - w.view.HorizontalScrollBar().Value()
	w.cursorY = w.y - w.view.VerticalScrollBar().Value()
}

func (w *Window) cursorScroll(row, col int) (int, int) {
	lineHeight := w.buffer.font.lineHeight
	lineHeightInt := int(lineHeight)
	posx, posy := w.getPos(row, col)
	dx := 0
	x := w.horizontalScrollBar.Value()
	verticalScrollBarWidth := 0
	if w.verticalScrollBar.IsVisible() {
		verticalScrollBarWidth = w.verticalScrollBarWidth
	}
	padding := int(w.buffer.font.width*2 + 0.5)
	end := x + w.frame.width + w.gutterWidth - padding - int(w.buffer.font.width+0.5) - verticalScrollBarWidth
	if posx < x+padding {
		dx = posx - (x + padding)
	} else if posx > end {
		dx = posx - end
	}
	if dx < 0 && x == 0 {
		dx = 0
	}

	dy := 0
	y := w.verticalScrollBar.Value()
	horizontalScrollBarHeight := 0
	if w.horizontalScrollBar.IsVisible() {
		horizontalScrollBarHeight = w.horizontalScrollBarHeight
	}
	end = y + w.frame.height - 2*lineHeightInt - horizontalScrollBarHeight
	if posy < y+lineHeightInt {
		dy = posy - (y + lineHeightInt)
	} else if posy > end {
		dy = posy - end
	}
	if dy < 0 && y == 0 {
		dy = 0
	}
	return dx, dy
}

func (w *Window) scrollToCursor() {
	lineHeight := w.buffer.font.lineHeight
	if !w.editor.smoothScroll {
		w.view.EnsureVisible2(
			float64(w.x),
			float64(w.y),
			1,
			lineHeight,
			20,
			20,
		)
		return
	}

	dx, dy := w.cursorScroll(w.row, w.col)
	if dx == 0 && dy == 0 {
		return
	}

	scroll := &Scroll{
		dx: dx,
		dy: dy,
	}
	w.scrollChan <- scroll
}

func (w *Window) scrollRow(rows int, jump bool) {
	y := int(float64(rows)*w.buffer.font.lineHeight + 0.5)
	row := w.row + rows
	if row < 0 {
		row = 0
	} else if row > len(w.buffer.lines)-1 {
		row = len(w.buffer.lines) - 1
	}
	col := w.scrollCol
	if w.buffer.lines[row] != nil {
		maxCol := len(w.buffer.lines[row].text) - 2
		if maxCol < 0 {
			maxCol = 0
		}
		if col > maxCol {
			col = maxCol
		}
	}
	if !jump {
		if w.row < w.start+rows+3 || w.row > w.end+rows-3 {
			jump = true
		}
	}
	go func() {
		w.scrollMutex.Lock()
		if w.scrolling {
			w.scrollMutex.Unlock()
			return
		}
		w.scrolling = true
		w.scrollMutex.Unlock()

		finished, _ := w.smoothScroll(0, y, 0, 0)
		<-finished
		if jump {
			w.row = row
			w.col = col
			w.buffer.xiView.Click(row, col)
		}
		w.scrollMutex.Lock()
		w.scrolling = false
		w.scrollMutex.Unlock()
	}()
}

func (w *Window) smoothScrollJob() {
	go func() {
		lastFinished := true
		finished := make(chan struct{})
		stop := make(chan struct{})
		for {
			select {
			case scroll := <-w.scrollChan:
				if !lastFinished {
					close(stop)
					<-finished
				}
				finished, stop = w.smoothScroll(scroll.dx, scroll.dy, 0, 0)
				lastFinished = false
			case <-finished:
				lastFinished = true
				finished = make(chan struct{})
			}
		}
	}()

	go func() {
		for {
			smoothScroll := <-w.scrollWaitChan
			w.editor.updates <- smoothScroll
			w.editor.signal.UpdateSignal()
			<-w.scrollDone
		}
	}()
}

func (w *Window) smoothScrollNew(s *SmoothScroll) {
	row := w.row + s.rows
	col := w.col + s.cols
	if s.cols == 0 {
		col = w.scrollCol
	}
	maxRow := len(w.buffer.lines) - 1
	if row < 0 {
		row = 0
	} else if row > maxRow {
		row = maxRow
	}
	maxCol := 0
	if w.buffer.lines[row] != nil {
		maxCol = len(w.buffer.lines[row].text) - 1
	}
	if maxCol < 0 {
		maxCol = 0
	}
	if col < 0 {
		col = 0
	} else if col > maxCol {
		col = maxCol
	}
	if w.row == row && w.col == col {
		w.scrollDone <- struct{}{}
		return
	}

	if s.changeScrollCol {
		w.scrollCol = col
	}

	dx, dy := w.cursorScroll(row, col)
	finished, _ := w.smoothScroll(dx, dy, row, col)
	go func() {
		<-finished
		w.editor.updates <- &SetPos{
			row:  row,
			col:  col,
			toXi: true,
		}
		w.editor.signal.UpdateSignal()
		w.scrollDone <- struct{}{}
	}()
}

func (w *Window) smoothScroll(x, y, row, col int) (chan struct{}, chan struct{}) {
	finished := make(chan struct{})
	stop := make(chan struct{})
	if x == 0 && y == 0 {
		close(finished)
		return finished, stop
	}
	total := 10
	if Abs(y) < 100 && Abs(x) < 100 {
		total = 3
	}
	scroll := &Scroll{
		dx: 0,
		dy: 0,
	}
	if Abs(x) < total {
		if x > 0 {
			scroll.dx = 1
		} else if x < 0 {
			scroll.dx = -1
		}
	} else {
		scroll.dx = x / total
	}
	if Abs(y) < total {
		if y > 0 {
			scroll.dy = 1
		} else if y < 0 {
			scroll.dy = -1
		}
	} else {
		scroll.dy = y / total
	}

	go func() {
		defer func() {
			w.row = row
			w.col = col
			w.editor.updates <- "updateXi"
			w.editor.signal.UpdateSignal()
			close(finished)
		}()
		dx := 0
		dy := 0
		xDiff := Abs(x) - dx
		yDiff := Abs(y) - dy
		for {
			if xDiff > 0 && xDiff < Abs(scroll.dx) {
				scroll.dx = xDiff
				if x < 0 {
					scroll.dx = -scroll.dx
				}
			}
			if yDiff > 0 && yDiff < Abs(scroll.dy) {
				scroll.dy = yDiff
				if y < 0 {
					scroll.dy = -scroll.dy
				}
			}
			w.editor.updates <- scroll
			w.editor.signal.UpdateSignal()

			dx += Abs(scroll.dx)
			dy += Abs(scroll.dy)
			xDiff = Abs(x) - dx
			yDiff = Abs(y) - dy

			select {
			case <-time.After(16 * time.Millisecond):
			case <-stop:
				if xDiff <= 0 && yDiff <= 0 {
					return
				}
				scroll.dx = xDiff
				if x < 0 {
					scroll.dx = -scroll.dx
				}
				scroll.dy = yDiff
				if y < 0 {
					scroll.dy = -scroll.dy
				}
				w.editor.updates <- scroll
				w.editor.signal.UpdateSignal()
				return
			}

			if xDiff <= 0 && yDiff <= 0 {
				return
			}
		}
	}()

	return finished, stop
}

func (w *Window) setPos(row, col int, toXi bool) {
	w.row = row
	b := w.buffer
	b.y = row * int(b.font.lineHeight)
	if b.lines[row] == nil {
		b.x = 0
		col = 0
	} else {
		text := b.lines[row].text
		if col > len(text) {
			col = len(text)
		}
		b.x = int(b.font.fontMetrics.Width(text[:col]) + 0.5)
	}
	w.x = b.x - w.horizontalScrollValue
	w.y = b.y - w.verticalScrollValue
	w.col = col
	if toXi {
		b.xiView.Click(w.row, w.col)
	}
	w.updateCursor()
	w.updateCline()
}

func (w *Window) updatePos() {
	b := w.buffer
	row := w.row
	col := w.col
	if b.lines[row] == nil {
		col = 0
		w.x = 0
	} else {
		text := b.lines[row].text
		if col > len(text) {
			col = len(text)
			w.col = col
		}
		w.x = int(b.font.fontMetrics.Width(text[:col]) + 0.5)
	}
	w.y = row * int(b.font.lineHeight)
}

func (w *Window) getPos(row, col int) (int, int) {
	b := w.buffer
	x := 0
	if b.lines[row] != nil {
		text := b.lines[row].text
		if col > len(text) {
			col = len(text)
		}
		x = int(b.font.fontMetrics.Width(text[:col]) + 0.5)
	}
	y := row * int(b.font.lineHeight)
	return x, y
}

func (w *Window) scrollto(col, row int, jump bool) {
	oldRow := w.row
	oldCol := w.col
	w.row = row
	w.col = col
	w.updatePos()
	if oldRow == row && oldCol == col {
		return
	}
	if jump {
		w.scrollToCursor()
	}
	w.updateCursorPos()
	w.updateCursor()
	w.updateCline()
}

func (w *Window) cursorTo(rows, cols int, changeScrollCol bool) {
	s := &SmoothScroll{
		rows:            rows,
		cols:            cols,
		changeScrollCol: changeScrollCol,
	}
	go func() {
		select {
		case w.scrollWaitChan <- s:
		case <-time.After(50 * time.Millisecond):
		}
	}()
}

func (w *Window) paintGutter(event *gui.QPaintEvent) {
	p := gui.NewQPainter2(w.gutter)
	defer p.DestroyQPainter()
	p.SetFont(w.buffer.font.font)
	fg := w.editor.theme.Theme.Selection
	fgColor := gui.NewQColor3(fg.R, fg.G, fg.B, fg.A)
	clineFg := w.editor.theme.Theme.Foreground
	clineColor := gui.NewQColor3(clineFg.R, clineFg.G, clineFg.B, clineFg.A)
	shift := int(w.buffer.font.shift+0.5) - (w.view.VerticalScrollBar().Value() - w.start*int(w.buffer.font.lineHeight))
	for i := w.start; i < w.end; i++ {
		if i >= len(w.buffer.lines) {
			return
		}
		if i == w.row {
			p.SetPen2(clineColor)
		} else {
			p.SetPen2(fgColor)
		}

		n := i + 1
		// if relative {
		if w.row != i {
			n = Abs(i - w.row)
		}
		// }
		padding := w.gutterPadding + int((w.buffer.font.fontMetrics.Width(strconv.Itoa(len(w.buffer.lines)))-w.buffer.font.fontMetrics.Width(strconv.Itoa(n)))+0.5)
		p.DrawText3(padding, (i-w.start)*int(w.buffer.font.lineHeight)+shift, strconv.Itoa(n))
	}
}