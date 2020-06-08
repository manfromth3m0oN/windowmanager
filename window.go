package main

import (
	"fmt"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
	"log"
	"sync"
)

type managedWindow xproto.Window
type workspace struct{
	screen *xinerama.ScreenInfo
	windows []managedWindow
	mu *sync.Mutex
}

var workspaces map[string]*workspace

func(w *workspace) removeWindow(win xproto.Window) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	// find the index of win
	// remove win from the w.windows slice
	indexwintodel := -1
	for i, wintodel := range w.windows {
		if managedWindow(win) == wintodel {
			indexwintodel = i
			break
		}
	}
	if indexwintodel != -1 {
		w.windows = append(w.windows[0:indexwintodel], w.windows[indexwintodel+1:]...)
		return nil
	}
	return fmt.Errorf("window not managed by workspace")

}

func (w *workspace) TileWindows() error {
	log.Println("trying to tile")
	if w.screen == nil {
		return fmt.Errorf("workspace not attached to screen")
	}
	n := uint32(len(w.windows))
	var stackHeight uint32
	if n - 1 == 0 {
		stackHeight = uint32(w.screen.Height)
	} else {
		stackHeight = uint32(w.screen.Height) / (n - 1)
	}
	width := uint32(w.screen.Width / 2)
	log.Printf("n: %d, stackHeight: %d, width: %d", n, stackHeight, width)
	var err error
	log.Printf("len of w.windows: %d", len(w.windows))
	if len(w.windows) == 1 {
		if err := xproto.ConfigureWindowChecked(
			xc,
			xproto.Window(w.windows[0]),
			xproto.ConfigWindowX |
				xproto.ConfigWindowY |
				xproto.ConfigWindowHeight |
				xproto.ConfigWindowWidth,
			[]uint32{
				0,
				0,
				uint32(w.screen.Width),
				uint32(w.screen.Height),
			}).Check(); err != nil {
			log.Fatal("Couldn't configure the master window")
		}
	} else {
		for i, win := range w.windows {
			if i == 0 {
				log.Println("configuring master window")
				log.Printf("X: %d, Y: %d, Width: %d, Height: %d", 0, 0, w.screen.Height, width)
				if err := xproto.ConfigureWindowChecked(
					xc,
					xproto.Window(win),
					xproto.ConfigWindowX |
						xproto.ConfigWindowY |
						xproto.ConfigWindowHeight |
						xproto.ConfigWindowWidth,
					[]uint32{
						0,
						0,
						uint32(width),
						uint32(w.screen.Height),
					}).Check(); err != nil {
					log.Fatal("Couldn't configure the master window")
				}
			} else {
				log.Println(i)
				log.Printf("X: %d, Y: %d, Width: %d, Height: %d", width, (uint32(i)-2)*stackHeight, width, stackHeight)
				if err := xproto.ConfigureWindowChecked(
					xc,
					xproto.Window(win),
					xproto.ConfigWindowX |
						xproto.ConfigWindowY |
						xproto.ConfigWindowHeight |
						xproto.ConfigWindowWidth,
					[]uint32{
						width,
						(uint32(i)-1) * stackHeight,
						width,
						stackHeight,
					}).Check(); err != nil {
					log.Fatal("Couldn't tile stack")
				}
			}
		}
	}
	return err
}

func (w *workspace) Add(win xproto.Window) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := xproto.ChangeWindowAttributesChecked(
		xc,
		win,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskStructureNotify | xproto.EventMaskEnterWindow,
		}).Check(); err != nil {
		return err
	}
	w.windows = append(w.windows, managedWindow(win))
	if err := w.TileWindows(); err != nil {
		log.Println(err)
		log.Fatal("error tiling after window add")
	}
	return nil
}