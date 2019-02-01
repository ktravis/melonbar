package main

import (
	"github.com/BurntSushi/xgbutil/xgraphics"
)

// Block is a struct with information about a block.
type Block struct {
	// The sub-image that represents the block.
	img *xgraphics.Image

	// The text the block should display.
	ID      string
	Text    string
	Width   int
	TextPad int
	Dirty   bool

	// The x coordinate and width of the block.
	//x, w int
	x int

	// Additional x offset to further tweak the location of the text.
	xoff int

	// The aligment of the text, this can be `l` for left aligment, `c` for
	// center aligment, `r` for right aligment and `a` for absolute center
	// aligment.
	TextAlign Alignment

	// The foreground and background colors in hex.
	FGColor string
	BGColor string

	// A map with functions to execute on button events. Accepted button strings
	// are `button0` to `button5`
	actions map[string]func() error

	// Block popup..
	popup *Popup
}

func (b *Block) AddAction(event string, action func() error) {
	if b.actions == nil {
		b.actions = make(map[string]func() error)
	}
	b.actions[event] = action
}

func (b *Block) OnClick(action func() error) {
	b.AddAction("button1", action)
}

func (b *Block) RunAction(event string) error {
	if b.actions == nil {
		return nil
	}
	if fn, ok := b.actions[event]; ok {
		return fn()
	}
	return nil
}

//func (bar *Bar) initBlock(name, txt string, w int, align Alignment, xoff int, bg, fg string) *Block {
//block := new(Block)

////block.x = bar.xsum
//block.Width = w
//block.xoff = xoff
//block.Text = txt
//block.TextAlign = align
//block.BGColor = bg
//block.Dirty = true
////block.FGColor = fg

//// Add the width of this block to the xsum.
////bar.xsum += w

//// Store the block in map.
////bar.blocks = append(bar.blocks, block)

//// Draw block.
////block.redraw(bar)

//return block
//}

func (b *Block) redraw(bar *Bar) {
	b.Dirty = true
	bar.redraw <- b
}

// TODO: Make this function more versatile by allowing different and multiple
// properties to be checked.
func (block *Block) diff(txt string) bool {
	if block.Text == txt {
		return false
	}
	block.Text = txt
	return true
}
