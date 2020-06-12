package main

import (
	"github.com/BurntSushi/xgb/xproto"
	"github.com/manfromth3m0on/windowmanageragain/keysym"
	"log"
)

func grabkeys() {
	grabs := []struct{
		sym xproto.Keysym
		mods uint16
		codes []xproto.Keycode
	}{{
		sym: keysym.XK_c,
		mods: xproto.ModMask4 | xproto.ModMaskShift,
	},
	{
		sym: keysym.XK_Return,
		mods: xproto.ModMask4,
	},
	{
		sym: keysym.XK_q,
		mods: xproto.ModMask4,
	},
	{
		sym: keysym.XK_q,
		mods: xproto.ModMask4 | xproto.ModMaskShift,
	}}

	for i, syms := range keymap {
		for _, sym := range syms {
			for c := range grabs {
				if grabs[c].sym == sym {
					grabs[c].codes = append(grabs[c].codes, xproto.Keycode(i))
				}
			}
		}
	}
	for _, grabbed := range grabs {
		for _, code := range grabbed.codes {
			if err := xproto.GrabKeyChecked(
				xc,
				false,
				xroot.Root,
				grabbed.mods,
				code,
				xproto.GrabModeAsync,
				xproto.GrabModeAsync).Check(); err != nil {
				log.Printf("Key grabbing error: %v", err)
			}
		}
	}
}