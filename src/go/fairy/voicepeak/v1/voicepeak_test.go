package voicepeak

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zzl/go-win32api/win32"
)

func TestMain(m *testing.M) {
	if hr := win32.CoInitializeEx(nil, win32.COINIT_MULTITHREADED); win32.FAILED(hr) {
		panic(win32.HRESULT_ToString(hr))
	}
	defer win32.CoUninitialize()
	os.Exit(m.Run())
}

func namer(name, text string) (string, error) {
	const maxLen = 10
	shortText := make([]rune, 0, maxLen)
	for _, c := range text {
		if len(shortText) == maxLen {
			shortText[maxLen-1] = '…'
			break
		}
		switch c {
		case
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
			0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
			0x20, 0x22, 0x2a, 0x2f, 0x3a, 0x3c, 0x3e, 0x3f, 0x7c, 0x7f:
			continue
		}
		shortText = append(shortText, c)
	}
	return filepath.Join(
		`C:\Users\anonymous\Source\MSYS2\forcepser\bin\tmp`,
		fmt.Sprintf("%d_%s_%s.wav", time.Now().Unix(), name, string(shortText)),
	), nil
}

func TestExecute(t *testing.T) {
	if err := (&voicepeak{}).Execute(0, namer); err != nil {
		t.Fatal(err)
	}
}
