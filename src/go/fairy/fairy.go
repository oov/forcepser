package fairy

import (
	"fmt"

	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

type Fairy interface {
	IsTarget(exePath string) bool
	TestedProgram() string
	Execute(hwnd win32.HWND, namer func(name, text string) (string, error)) error
}

type Fairies []Fairy

var ErrTargetNotFound = fmt.Errorf("target not found")

func (fs Fairies) Execute(namer func(name, text string) (string, error)) error {
	hwnd := win32.GetForegroundWindow()
	if hwnd == 0 || hwnd == win32.INVALID_HANDLE_VALUE {
		return fmt.Errorf("failed to get foreground window")
	}
	var pid uint32
	win32.GetWindowThreadProcessId(hwnd, &pid)
	exePath, err := internal.GetModulePathFromPID(pid)
	if err != nil {
		return fmt.Errorf("failed to get module path: %w", err)
	}
	for _, f := range fs {
		if f.IsTarget(exePath) {
			err = f.Execute(hwnd, namer)
			if err != nil {
				return fmt.Errorf("fairy failed to work: %w", err)
			}
			return nil
		}
	}
	return ErrTargetNotFound
}
