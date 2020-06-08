package main

import (
	"github.com/BurntSushi/xgb/xproto"
	"log"
)

func initAtoms() {
	atomNetActiveWindow = internAtom("_NET_ACTIVE_WINDOW")
	atomNetWMName = internAtom("_NET_WM_NAME")
	atomWindow = internAtom("WINDOW")
	atomWMClass = internAtom("WM_CLASS")
	atomWMDeleteWindow = internAtom("WM_DELETE_WINDOW")
	atomWMProtocols = internAtom("WM_PROTOCOLS")
	atomWMTakeFocus = internAtom("WM_TAKE_FOCUS")
	atomWMTransientFor = internAtom("WM_TRANSIENT_FOR")
}

func internAtom(name string) xproto.Atom {
	r, err := xproto.InternAtom(xc, false, uint16(len(name)), name).Reply()
	if err != nil {
		log.Fatal(err)
	}
	if r == nil {
		return 0
	}
	return r.Atom
}
