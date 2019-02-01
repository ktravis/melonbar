package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xprop"
	"github.com/fhs/gompd/mpd"
	"github.com/fsnotify/fsnotify"
	"github.com/rkoesters/xdg/basedir"
	"golang.org/x/image/font"
)

func (bar *Bar) clock() *Block {
	// Initialize block.
	block := &Block{
		ID:        "clock",
		Width:     400,
		TextAlign: AlignCenter,
	}

	// Show popup on clicking the left mouse button.
	//block.OnClick(func() error {
	//if block.popup != nil {
	//block.popup = block.popup.destroy()
	//return nil
	//}

	//var err error
	//block.popup, err = initPopup((bar.w/2)-(178/2), 29, 178, 129, "#EEEEEE", "#021B21")
	//if err != nil {
	//return err
	//}

	//return block.popup.clock()
	//})

	go func() {
		for {
			// Compose block text.
			txt := time.Now().Format("Monday, January 2th 03:04 PM")

			// Redraw block.
			if block.diff(txt) {
				block.redraw(bar)
			}

			// Update every 45 seconds.
			time.Sleep(30 * time.Second)
		}
	}()
	return block
}

func (bar *Bar) music() {
	// Initialize block.
	block := &Block{
		ID:        "music",
		Width:     660,
		TextAlign: AlignRight,
	}

	// Notify that the next block can be initialized.
	//bar.ready <- true

	// Connect to MPD.
	c, err := mpd.Dial("tcp", ":6600")
	if err != nil {
		log.Fatalf("unable to connect to mpd: %v", err)
	}

	// Keep connection alive by pinging ever 45 seconds.
	go func() {
		for {
			time.Sleep(time.Second * 45)

			if err := c.Ping(); err != nil {
				c, err = mpd.Dial("tcp", ":6600")
				if err != nil {
					log.Fatalf("unable to connect to mpd: %v", err)
				}
			}
		}
	}()

	// Show popup on clicking the left mouse button.
	block.OnClick(func() error {
		if block.popup != nil {
			block.popup = block.popup.destroy()
			return nil
		}

		block.popup, err = initPopup(1920-304-29, 29, 304, 148, "#EEEEEE",
			"#021B21")
		if err != nil {
			return err
		}

		return block.popup.music(c)
	})

	// Toggle play/pause on clicking the right mouse button.
	block.AddAction("button3", func() error {
		status, err := c.Status()
		if err != nil {
			return err
		}

		return c.Pause(status["state"] != "pause")
	})

	// Previous song on scrolling up.
	block.AddAction("button4", c.Previous)

	// Next song on on scrolling down..
	block.AddAction("button5", c.Next)

	// Watch MPD for events.
	w, err := mpd.NewWatcher("tcp", ":6600", "", "player")
	if err != nil {
		log.Fatalln(err)
	}
	for {
		cur, err := c.CurrentSong()
		if err != nil {
			log.Println(err)
		}
		sts, err := c.Status()
		if err != nil {
			log.Println(err)
		}

		// Compose text.
		var s string
		if sts["state"] == "pause" {
			s = "[paused] "
		}
		txt := "»      " + s + cur["Artist"] + " - " + cur["Title"]

		// Redraw block.
		if block.diff(txt) {
			block.redraw(bar)

			// Redraw popup.
			if block.popup != nil {
				if err := block.popup.music(c); err != nil {
					log.Println(err)
				}
			}
		}

		<-w.Event
	}
}

func (bar *Bar) todo() {
	// Initialize block.
	block := &Block{
		ID:        "todo",
		Text:      "¢",
		Width:     29,
		TextAlign: AlignCenter,
	}

	// Notify that the next block can be initialized.
	//bar.ready <- true

	// Show popup on clicking the left mouse button.
	block.OnClick(func() error {
		cmd := exec.Command("st", "micro", "-savecursor", "false", path.Join(
			basedir.Home, ".todo"))
		cmd.Stdout = os.Stdout
		return cmd.Run()
	})

	// Watch file for events.
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln(err)
	}
	if err := w.Add(path.Join(basedir.Home, ".todo")); err != nil {
		log.Fatalln(err)
	}
	f, err := os.Open(path.Join(basedir.Home, ".todo"))
	if err != nil {
		log.Fatalln(err)
	}
	for {
		// Count file lines.
		s := bufio.NewScanner(f)
		s.Split(bufio.ScanLines)
		var c int
		for s.Scan() {
			c++
		}

		// Rewind file.
		if _, err := f.Seek(0, 0); err != nil {
			log.Println(err)
		}

		// Compose block text.
		txt := "¢ " + strconv.Itoa(c)

		// Redraw block.
		if block.diff(txt) {
			block.redraw(bar)
		}

		// Listen for next write event.
		ev := <-w.Events
		if ev.Op&fsnotify.Write != fsnotify.Write {
			continue
		}
	}
}

func (bar *Bar) window() *Block {
	// Initialize blocks.
	const (
		bw  = 340
		pad = 8
	)
	block := &Block{
		Width:     bw,
		TextAlign: AlignLeft,
		TextPad:   pad,
	}

	// Notify that the next block can be initialized.
	//bar.ready <- true

	// TODO: This doesn't check for window title changes.
	xevent.PropertyNotifyFun(func(_ *xgbutil.XUtil, ev xevent.
		PropertyNotifyEvent) {
		// Only listen to `_NET_ACTIVE_WINDOW` events.
		atom, err := xprop.Atm(X, "_NET_ACTIVE_WINDOW")
		if err != nil {
			log.Println(err)
		}
		if ev.Atom != atom {
			return
		}

		// Get active window.
		id, err := ewmh.ActiveWindowGet(X)
		if err != nil {
			log.Println(err)
		}
		if id == 0 {
			return
		}

		// Compose block text.
		txt, err := ewmh.WmNameGet(X, id)
		if err != nil || len(txt) == 0 {
			txt, err = icccm.WmNameGet(X, id)
			if err != nil || len(txt) == 0 {
				txt = "?"
			}
		}

		// Redraw block.
		if block.diff(txt) {
			block.redraw(bar)
		}
	}).Connect(X, X.RootWin())
	return block
}

func (bar *Bar) workspace() []*Block {
	ws, err := ewmh.DesktopNamesGet(X)
	if err != nil {
		log.Fatal(err)
	}
	bs := make([]*Block, 0)

	var pwsp, nwsp int

	for i, d := range ws {
		pw := (font.MeasureString(face, d).Ceil() + 4)
		pw = 16 * (1 + pw/16)
		if pw < 32 {
			pw = 32
		}
		b := &Block{
			Width:     pw,
			Text:      d,
			TextAlign: AlignCenter,
			BGColor:   "#5394C9",
		}
		n := i
		b.OnClick(func() error {
			return ewmh.CurrentDesktopReq(X, n)
		})
		b.AddAction("button4", func() error {
			return ewmh.CurrentDesktopReq(X, pwsp)
		})
		b.AddAction("button5", func() error {
			return ewmh.CurrentDesktopReq(X, nwsp)
		})
		bs = append(bs, b)
	}

	// Notify that the next block can be initialized.
	//bar.ready <- true

	var owsp uint
	xevent.PropertyNotifyFun(func(_ *xgbutil.XUtil, ev xevent.PropertyNotifyEvent) {
		// Only listen to `_NET_ACTIVE_WINDOW` events.
		atom, err := xprop.Atm(X, "_NET_CURRENT_DESKTOP")
		if err != nil {
			log.Println(err)
		}
		if ev.Atom != atom {
			return
		}

		// Get the current active desktop.
		wsp, err := ewmh.CurrentDesktopGet(X)
		if err != nil {
			log.Println(err)
		}

		for i, b := range bs {
			if i == int(wsp) {
				b.BGColor = "#72A7D3"
				pwsp = i - 1
				if pwsp < 0 {
					pwsp = len(bs) - 1
				}
				nwsp = i + 1
				if nwsp >= len(bs) {
					nwsp = 0
				}
			} else {
				b.BGColor = "#5394C9"
			}
		}
		if owsp != wsp {
			bs[owsp].redraw(bar)
			bs[wsp].redraw(bar)
			owsp = wsp
		}
	}).Connect(X, X.RootWin())
	return bs
}

func (bar *Bar) battery() *Block {
	// Initialize block.
	block := &Block{
		ID:        "battery",
		Text:      "?",
		Width:     72,
		TextAlign: AlignCenter,
	}

	// Notify that the next block can be initialized.
	//bar.ready <- true

	go func() {

		for {
			b, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/capacity")
			if err != nil {
				log.Printf("error reading battery capacity: %v", err)
			}
			b = bytes.TrimSpace(b)

			txt := string(b) + "%"
			{
				b, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/status")
				if err != nil {
					log.Printf("error reading battery status: %v", err)
				}
				b = bytes.TrimSpace(b)

				if string(b) != "Discharging" {
					txt = fmt.Sprintf("%s %s", txt, string(b))
				}
			}
			txt = fmt.Sprintf("\uf578 %s", txt)

			// Redraw block.
			if block.diff(txt) {
				block.redraw(bar)
			}

			// Update every 45 seconds.
			time.Sleep(60 * time.Second)
		}

	}()
	return block
}

func (bar *Bar) wifi() *Block {
	// Initialize block.
	block := &Block{
		Width:     200,
		TextAlign: AlignCenter,
	}

	// Notify that the next block can be initialized.
	//bar.ready <- true

	go func() {
		for {
			iface := "wlp2s0"
			if i := os.Getenv("WIFI_INTERFACE"); i != "" {
				iface = i
			}
			cmd := exec.Command("iwgetid", "-r", iface)
			b, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("err = %+v\n", err)
			}
			txt := strings.TrimSpace(string(b))
			if txt == "" {
				txt = "\ufaa9 not connected"
			} else {
				txt = fmt.Sprintf("\ufaa8 %s", txt)
			}

			if block.diff(txt) {
				block.redraw(bar)
			}

			time.Sleep(15 * time.Second)
		}
	}()
	return block
}
