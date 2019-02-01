package main

import (
	"fmt"
	"image"
	"log"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Alignment int

const (
	AlignCenter Alignment = iota
	AlignLeft
	AlignRight

	DefaultBGColor = "#222222"
	DefaultFGColor = "#cccccc"
)

// Bar is a struct with information about the bar.
type Bar struct {
	// Bar window, and bar image.
	win *xwindow.Window
	img *xgraphics.Image

	// The width and height of the bar.
	w, h int

	// Text drawer.
	drawer *font.Drawer

	groups []Group

	redraw chan *Block
}

type Group struct {
	Align  Alignment
	Blocks []*Block
}

func (g *Group) Width() int {
	var w int
	for _, b := range g.Blocks {
		w += b.Width
	}
	return w
}

func initBar(x, y, w, h int) (*Bar, error) {
	bar := new(Bar)
	var err error

	// Create a window for the bar. This window listens to button press events
	// in order to respond to them.
	bar.win, err = xwindow.Generate(X)
	if err != nil {
		return nil, err
	}
	bar.win.Create(X.RootWin(), x, y, w, h, xproto.CwBackPixel|xproto.CwEventMask, 0x000000, xproto.EventMaskButtonPress)
	bar.win.Listen(
		xproto.EventMaskButtonPress,
		xproto.EventMaskEnterWindow,
		xproto.EventMaskLeaveWindow,
		xproto.EventMaskPointerMotion,
		xproto.EventMaskPointerMotionHint,
	)

	// EWMH stuff to make the window behave like an actual bar.
	// XXX: `WmStateSet` and `WmDesktopSet` are basically here to keep OpenBox
	// happy, can I somehow remove them and just use `_NET_WM_WINDOW_TYPE_DOCK`
	// like I can with WindowChef?
	if err := ewmh.WmWindowTypeSet(X, bar.win.Id, []string{
		"_NET_WM_WINDOW_TYPE_DOCK"}); err != nil {
		return nil, err
	}
	if err := ewmh.WmStateSet(X, bar.win.Id, []string{
		"_NET_WM_STATE_STICKY"}); err != nil {
		return nil, err
	}
	if err := ewmh.WmDesktopSet(X, bar.win.Id, ^uint(0)); err != nil {
		return nil, err
	}
	if err := ewmh.WmNameSet(X, bar.win.Id, "melonbar"); err != nil {
		return nil, err
	}

	// Map window.
	bar.win.Map()

	// XXX: Moving the window is again a hack to keep OpenBox happy.
	bar.win.Move(x, y)

	// Create the bar image.
	bar.img = xgraphics.New(X, image.Rect(0, 0, w, h))
	if err := bar.img.XSurfaceSet(bar.win.Id); err != nil {
		return nil, err
	}
	// draw the background - but this will need to be cleared if other things move? Can other things move?
	bg := hexToBGRA(DefaultBGColor)
	bar.img.For(func(cx, cy int) xgraphics.BGRA {
		return bg
	})
	bar.img.XDraw()

	bar.w = w
	bar.h = h

	bar.drawer = &font.Drawer{
		Dst:  bar.img,
		Face: face,
	}

	//bar.blocks = new(sync.Map)
	bar.redraw = make(chan *Block)

	xevent.EnterNotifyFun(func(_ *xgbutil.XUtil, ev xevent.EnterNotifyEvent) {
		fmt.Printf("enter = %+v\n", ev)
	}).Connect(X, bar.win.Id)

	xevent.MotionNotifyFun(func(_ *xgbutil.XUtil, ev xevent.MotionNotifyEvent) {
		fmt.Printf("motion = %+v\n", ev)
	}).Connect(X, bar.win.Id)

	xevent.LeaveNotifyFun(func(_ *xgbutil.XUtil, ev xevent.LeaveNotifyEvent) {
		fmt.Printf("leave = %+v\n", ev)
	}).Connect(X, bar.win.Id)

	// Listen to mouse events and execute the required function.
	xevent.ButtonPressFun(func(_ *xgbutil.XUtil, ev xevent.ButtonPressEvent) {

		bar.eachBlock(func(xoff int, block *Block) bool {

			ex := int(ev.EventX)
			// XXX: Hack for music block.
			if block.ID == "music" {
				tw := font.MeasureString(face, block.Text).Ceil()
				if ex >= xoff+(block.Width-tw+(block.xoff*2)) && ex < xoff+block.Width {
					if err := block.RunAction(fmt.Sprintf("button%d", ev.Detail)); err != nil {
						log.Println(err)
					}
					return false
				}
			}

			if ex >= xoff && ex < xoff+block.Width {
				if err := block.RunAction(fmt.Sprintf("button%d", ev.Detail)); err != nil {
					log.Println(err)
				}
				return false
			}
			return true
		})

	}).Connect(X, bar.win.Id)

	return bar, nil
}

func (bar *Bar) draw(xoff int, block *Block) error {
	// Color the background.
	block.img.For(func(cx, cy int) xgraphics.BGRA {
		// XXX: Hack for music block.
		bg := block.BGColor
		if bg == "" {
			bg = DefaultBGColor
		}
		if block.ID == "music" {
			if cx < xoff+block.xoff {
				return hexToBGRA("#445967")
			}
			return hexToBGRA(bg)
		}

		return hexToBGRA(bg)
	})

	// Set text color.
	fg := block.FGColor
	if fg == "" {
		fg = DefaultFGColor
	}
	bar.drawer.Src = image.NewUniform(hexToBGRA(fg))

	txt := block.Text
	quota := block.Width - 2*block.TextPad
	for n := len(txt); n > 0; n-- {
		tw := font.MeasureString(face, txt).Ceil()
		if tw < quota {
			break
		}
		txt = txt[:n] + "..."
	}

	tw := bar.drawer.MeasureString(txt).Ceil()
	var tx int
	switch block.TextAlign {
	case AlignLeft:
		tx = xoff + block.TextPad
	case AlignCenter:
		tx = xoff + ((block.Width / 2) - (tw / 2))
	case AlignRight:
		tx = (xoff + block.Width) - tw
	default:
		return fmt.Errorf("draw (%v): Not a valid aligment type", block.TextAlign)
	}

	// swap this for the xgbutil text stuff
	bar.drawer.Dot = fixed.P(tx, 18)
	bar.drawer.DrawString(txt)

	// Redraw the bar.
	block.img.XDraw()
	bar.img.XPaint(bar.win.Id)

	return nil
}

func (bar *Bar) SetGroups(gg ...Group) {
	bar.groups = gg
	bar.eachBlock(func(xoff int, b *Block) bool {
		b.img = bar.img.SubImage(image.Rect(xoff, 0, xoff+b.Width, bar.h)).(*xgraphics.Image)
		b.Dirty = true
		return true
	})
}

func (bar *Bar) eachBlock(fn func(int, *Block) bool) {
	for _, g := range bar.groups {
		xoff := 0
		goff := 0
		switch g.Align {
		case AlignLeft:
			xoff = 0
		case AlignCenter:
			xoff = bar.w/2 - g.Width()/2
		case AlignRight:
			xoff = bar.w - g.Width()
		}
		for _, b := range g.Blocks {
			if !fn(xoff+goff, b) {
				break
			}
			goff += b.Width
		}
	}
}

func (bar *Bar) drawLoop() {
	for {
		bar.eachBlock(func(xoff int, b *Block) bool {
			if b.Dirty {
				if err := bar.draw(xoff, b); err != nil {
					log.Fatalln(err)
				}
				b.Dirty = false
			}
			return true
		})
		<-bar.redraw
	}
}
