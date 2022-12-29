package hotkey

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/win32"
)

type Key struct {
	mods win32.HOT_KEY_MODIFIERS
	key  win32.VIRTUAL_KEY
}

func (k *Key) String() string {
	var s []string
	if ms := modifierToStr(k.mods); ms != "" {
		s = append(s, ms)
	}
	s = append(s, keyToStr(k.key))
	return strings.Join(s, " + ")
}

func ParseKey(s string) (*Key, error) {
	var mods win32.HOT_KEY_MODIFIERS
	var key win32.VIRTUAL_KEY
	for _, p := range strings.Split(s, "+") {
		p = strings.ToLower(strings.TrimSpace(p))
		if m := strToModifier(p); m != 0 {
			mods |= m
			continue
		}
		if k := strToKey(p); k != 0 {
			if key != 0 {
				return nil, fmt.Errorf("only one key can be specified")
			}
			key = k
			continue
		}
		return nil, fmt.Errorf("%v is unexpected key name", p)
	}
	if mods == 0 {
		return nil, fmt.Errorf("at least one modifier key(shift, cyrl, alt, win) is required")
	}
	if key == 0 {
		return nil, fmt.Errorf("no key specified")
	}
	return &Key{mods, key}, nil
}

type Hotkey struct {
	Notify chan func()
	notify chan error
	thread uint32
	key    *Key
}

var worker = syscall.NewCallback(func(userdata unsafe.Pointer) uintptr {
	hk := (*Hotkey)(userdata)
	if b, err := win32.RegisterHotKey(0, 1, hk.key.mods|win32.MOD_NOREPEAT, uint32(hk.key.key)); b == win32.FALSE {
		hk.notify <- err
		return 0
	}
	hk.notify <- nil
	defer func() {
		hk.notify <- nil
	}()
	defer win32.UnregisterHotKey(0, 1)
	var msg win32.MSG
	var busy bool
	complete := func(tid uint32) func() {
		return func() { win32.PostThreadMessage(tid, win32.WM_USER, 1, 0) }
	}(hk.thread)
	for {
		if b, _ := win32.GetMessage(&msg, 0, 0, 0); b == win32.FALSE {
			return 0
		}
		switch msg.Message {
		case win32.WM_HOTKEY:
			if !busy {
				busy = true
				hk.Notify <- complete
			}
		case win32.WM_USER:
			switch msg.WParam {
			case 0:
				return 0
			case 1:
				busy = false
			}
		}
	}
})

func New(hotkey string) (*Hotkey, error) {
	key, err := ParseKey(hotkey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}
	hk := &Hotkey{
		Notify: make(chan func()),
		notify: make(chan error),
		key:    key,
	}

	// I don't want to write a func that always loops with PeekMessage.
	// So create a thread with CreateThread and use GetMessage.

	// Create in suspend to assign thread ID first
	h, err := win32.CreateThread(nil, 0, worker, unsafe.Pointer(hk), win32.THREAD_CREATE_SUSPENDED, &hk.thread)
	if h == 0 {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}
	defer win32.CloseHandle(h)
	//
	if r, err := win32.ResumeThread(h); r == 0xffffffff {
		return nil, fmt.Errorf("failed to resume thread: %w", err)
	}
	err = <-hk.notify
	if err != nil {
		return nil, fmt.Errorf("failed to initialize global hot key: %w", err)
	}
	return hk, nil
}

func (hk *Hotkey) Close() error {
	win32.PostThreadMessage(hk.thread, win32.WM_USER, 0, 0)
	return <-hk.notify
}
