package main

import (
	"errors"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xinerama"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/manfromth3m0on/windowmanageragain/keysym"
	"log"
	"os/exec"
	"sync"
	"time"
)

var xc *xgb.Conn
var xroot xproto.ScreenInfo
var quitSignal error = errors.New("quit")
var keymap [256][]xproto.Keysym
var attachedScreens []xinerama.ScreenInfo
var activeWindow *xproto.Window

var (
	atomNetActiveWindow xproto.Atom
	atomNetWMName       xproto.Atom
	atomWindow          xproto.Atom
	atomWMClass         xproto.Atom
	atomWMDeleteWindow  xproto.Atom
	atomWMProtocols     xproto.Atom
	atomWMTakeFocus     xproto.Atom
	atomWMTransientFor  xproto.Atom
)

func takeOwnership() error {
	return xproto.ChangeWindowAttributesChecked(
		xc,
		xroot.Root,
		xproto.CwEventMask,
		[]uint32{
			xproto.EventMaskButtonPress |
				xproto.EventMaskButtonRelease |
				xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease |
				xproto.EventMaskStructureNotify |
				xproto.EventMaskSubstructureRedirect,
		}).Check()
}

func handleKeyPressEvent(key xproto.KeyPressEvent) error {
	switch keymap[key.Detail][0] {
	case keysym.XK_c:
		if (key.State&xproto.ModMaskShift != 0) && (key.State&xproto.ModMask4 != 0) {
			log.Println("Quitting")
			return quitSignal
		}
		return nil
	case keysym.XK_Return:
		if key.State & xproto.ModMask4 != 0 {
			cmd := exec.Command("alacritty")
			err := cmd.Start()
			go func() {
				cmd.Wait()
			}()
			return err
		}
		return nil
	case keysym.XK_q:
		switch key.State {
		case xproto.ModMask1:
			//WM_DELETE_WINDOW
			log.Printf("destroying active window: %v", activeWindow)
			prop, err := xproto.GetProperty(xc, false, *activeWindow, atomWMProtocols,
				xproto.GetPropertyTypeAny, 0, 64).Reply()
			if err != nil {
				return err
			}
			if prop == nil {
				return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
			}
			for v := prop.Value; len(v) >= 4; v = v[4:] {
				switch xproto.Atom( uint32(v[0]) | uint32(v[1]) <<8 | uint32(v[2]) <<16 | uint32(v[3]) <<24) {
				case atomWMDeleteWindow:
					t := time.Now().Unix()
					return xproto.SendEventChecked(
						xc,
						false,
						*activeWindow,
						xproto.EventMaskNoEvent,
						string(xproto.ClientMessageEvent{
							Format: 32,
							Window: *activeWindow,
							Type: atomWMProtocols,
							Data: xproto.ClientMessageDataUnionData32New([]uint32{
								uint32(atomWMDeleteWindow),
								uint32(t),
								0,
								0,
								0,
							}),
						}.Bytes())).Check()
				}
			}
		case xproto.ModMaskShift | xproto.ModMask4:
			// Destroy active window
			log.Printf("forcefully destroying active window: %v", activeWindow)
			return xproto.DestroyWindowChecked(xc, *activeWindow).Check()
		}
		return nil
	default:
		return nil
	}
}

func main() {
	xcon, err := xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	xc = xcon
	defer xc.Close()

	setup := xproto.Setup(xc)
	if setup == nil || len(setup.Roots) < 1 {
		log.Fatal("couldn't parse x conn info")
	}

	if err := xinerama.Init(xc); err != nil {
		log.Fatal(err)
	}

	if r, err := xinerama.QueryScreens(xc).Reply(); err != nil {
		log.Fatal(err)
	} else {
		attachedScreens = r.ScreenInfo
	}

	if r, err := xinerama.QueryScreens(xc).Reply(); err != nil {
		log.Fatal(err)
	} else {
		log.Println(r.ScreenInfo)
		if len(r.ScreenInfo) == 0 {
			log.Println("Manual xineramama setup")
			attachedScreens = []xinerama.ScreenInfo{
				xinerama.ScreenInfo{
					Width: setup.Roots[0].WidthInPixels,
					Height: setup.Roots[0].HeightInPixels,
				},
			}
		} else {
			log.Println("automatic setup")
			attachedScreens = r.ScreenInfo
		}
	}

	xroot = setup.Roots[0]
	initAtoms()

	const (
		loKey = 8
		hiKey = 255
	)
	m := xproto.GetKeyboardMapping(xc, loKey, hiKey-loKey+1)
	reply, err := m.Reply()
	if err != nil {
		log.Fatal(err)
	}
	if reply == nil {
		log.Fatal("couldn't load keymap")
	}

	for i := 0; i <hiKey-loKey+1; i++ {
		keymap[loKey + i] = reply.Keysyms[i*int(reply.KeysymsPerKeycode):(i+1)*int(reply.KeysymsPerKeycode)]
	}

	if err := takeOwnership(); err != nil {
		if _, ok := err.(xproto.AccessError); ok {
			log.Fatal("couldn't take ownership")
		}
		log.Fatal(err)
	}

	tree, err := xproto.QueryTree(xc, xroot.Root).Reply()
	log.Println("tree:")
	log.Println(tree)
	if err != nil {
		log.Fatal(err)
	}

	grabkeys()

	if tree != nil {
		workspaces = make(map[string]*workspace)
		defaultw := &workspace{mu: &sync.Mutex{}}
		if len(attachedScreens) > 0 {
			defaultw.screen = &attachedScreens[0]
		}
		for _, c := range tree.Children {
			if err := defaultw.Add(c); err != nil {
				log.Println(err)
			}
		}

		workspaces["default"] = defaultw

		if err := defaultw.TileWindows(); err != nil {
			log.Println(err)
		}
	}

	/*desktopXWin, err := xproto.NewWindowId(xc)
	if err != nil {
		log.Fatal(err)
	}

	if err := xproto.CreateWindowChecked(
		xc,
		xroot.RootDepth,
		desktopXWin,
		xroot.Root,
		0,
		0,
		xroot.WidthInPixels,
		xroot.HeightInPixels,
		0,
		xproto.WindowClassInputOutput,
		xroot.RootVisual,
		xproto.CwOverrideRedirect | xproto.CwEventMask,
		[]uint32{
			1,
			xproto.EventMaskExposure,
		}).Check(); err != nil {
		log.Fatal(err)
	}
*/
	eventloop:
		for {
			xev, err := xc.WaitForEvent()
			if err != nil {
				log.Printf("Error in eventloop: %v", err)
				continue
			}
			log.Printf("X Event: %v", xev)
			switch e := xev.(type) {
			case xproto.KeyPressEvent:
				if err := handleKeyPressEvent(e); err != nil {
					break eventloop
				}
			case xproto.DestroyNotifyEvent:
				log.Println("destroy event registered")
				for _, w := range workspaces {
					go func(w *workspace) {
						if err := w.removeWindow(e.Window); err == nil {
							if err := w.TileWindows(); err != nil {
								log.Fatal(err)
							}
						}
					}(w)
				}
				if activeWindow != nil && e.Window == *activeWindow {
					activeWindow = nil
				}
			case xproto.ConfigureRequestEvent:
				reply, err := xproto.ListProperties(xc, e.Window).Reply()
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("List Properties reply: %v", reply.Atoms)
				ev := xproto.ConfigureNotifyEvent{
				Event: e.Window,
				Window: e.Window,
				AboveSibling: 0,
				X: e.X,
				Y: e.Y,
				Width: e.Width,
				Height: e.Height,
				BorderWidth: 0,
				OverrideRedirect: false,
				}
				xproto.SendEventChecked(xc, false, e.Window, xproto.EventMaskStructureNotify, string(ev.Bytes()))
			case xproto.MapRequestEvent:
				if winatrib, err := xproto.GetWindowAttributes(xc, e.Window).Reply(); err != nil || !winatrib.OverrideRedirect {
					w := workspaces["default"]
					xproto.MapWindowChecked(xc, e.Window)
					w.Add(e.Window)
					w.TileWindows()
				}
			case xproto.EnterNotifyEvent:
				activeWindow = &e.Event
				prop, err := xproto.GetProperty(xc, false, e.Event, atomWMProtocols,
					xproto.GetPropertyTypeAny, 0, 64).Reply()
				focused := false
				if err == nil {
				TakeFocusPropLoop:
					for v := prop.Value; len(v) >= 4; v = v[4:] {
						switch xproto.Atom( uint32(v[0]) | uint32(v[1]) <<8 | uint32(v[2]) <<16 | uint32(v[3]) << 24 ) {
						case atomWMTakeFocus:
							xproto.SendEventChecked(
								xc,
								false,
								e.Event,
								xproto.EventMaskNoEvent,
								string(xproto.ClientMessageEvent{
									Format: 32,
									Window: *activeWindow,
									Type:   atomWMProtocols,
									Data: xproto.ClientMessageDataUnionData32New([]uint32{
										uint32(atomWMTakeFocus),
										uint32(e.Time),
										0,
										0,
										0,
									}),
								}.Bytes())).Check()q
							focused = true
							break TakeFocusPropLoop
						}
					}
				}
				if !focused {
					if _, err := xproto.SetInputFocusChecked(xc, 0, e.Event, e.Time).Reply(); err != nil {
						log.Println(err)
					}
				}
			default:
				log.Println("Event not in event loop case")
			}
		}
}
