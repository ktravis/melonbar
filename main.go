package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

var (
	//box = packr.New("box", "./box")

	// Connection to the X server.
	X *xgbutil.XUtil

	// The font that should be used.
	face font.Face
)

func main() {
	// Initialize X.
	if err := initX(); err != nil {
		log.Fatalf("cannot initialize X: %v", err)
	}

	// Initialize font.
	if err := initFont(); err != nil {
		log.Fatalf("cannot initialize font: %v", err)
	}

	// Initialize bar.
	bar, err := initBar(0, 0, 1920, 26)
	if err != nil {
		log.Fatalf("cannot initialize bar: %v", err)
	}

	bar.SetGroups(
		Group{
			Align:  AlignLeft,
			Blocks: append(bar.workspace(), bar.window()),
		},
		Group{
			Align:  AlignCenter,
			Blocks: []*Block{bar.clock()},
		},
		Group{
			Align:  AlignRight,
			Blocks: []*Block{bar.battery(), bar.wifi()},
		},
	)
	// Initialize blocks.
	//go bar.initBlocks([]func(){
	//bar.workspace,
	//bar.window,
	//bar.clock,
	//bar.battery,
	//bar.wifi,
	////bar.music,
	////bar.todo,
	//})

	// Listen for redraw events.
	bar.drawLoop()
}

func initX() error {
	// Set up a connection to the X server.
	var err error
	X, err = xgbutil.NewConn()
	if err != nil {
		return err
	}

	// Run the main X event loop, this is used to catch events.
	go xevent.Main(X)

	// Listen to the root window for property change events, used to check if
	// the user changed the focused window or active workspace for example.
	return xwindow.New(X, X.RootWin()).Listen(xproto.EventMaskPropertyChange,
		xproto.EventMaskEnterWindow,
		xproto.EventMaskLeaveWindow,
		xproto.EventMaskPointerMotion,
		xproto.EventMaskPointerMotionHint,
	)
}

func initFont() error {
	//fr := func(name string) ([]byte, error) {
	//return box.Find(path.Join("fonts", name))
	//}
	//fp, err := box.Find("fonts/cure.font")
	//if err != nil {
	//return err
	//}
	//face, err = plan9font.ParseFont(fp, fr)
	//if err != nil {
	//return err
	//}
	//b, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), "fonts/Inconsolata/Inconsolata Bold for Powerline.ttf"))
	b, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), "fonts/FiraCode/Fura Code Medium Nerd Font Complete Mono.ttf"))
	if err != nil {
		return err
	}

	tt, err := truetype.Parse(b)
	if err != nil {
		return err
	}
	face = truetype.NewFace(tt, &truetype.Options{
		Size:    16,
		Hinting: font.HintingFull,
	})

	return nil
}
