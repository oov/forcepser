package aivoice2

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/oov/forcepser/fairy"
	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

var (
	exeName = "aivoice.exe"

	mainWindowName          = "A.I.VOICE2 Editor "
	mainWindowFramework     = "Win32"
	mainWindowClassName     = "FLUTTER_RUNNER_WIN32_WINDOW"
	mainWindowViewClassName = "FLUTTERVIEW"

	mainWindowExportButtonCaptions = []string{"書き出し"}

	exportDialogExportButtonCaptions = []string{"書き出しを実行"}

	windowCreationTimeout       = 5 * time.Second
	windowCreationCheckInterval = 40 * time.Millisecond
	windowUpdateInterval        = 300 * time.Millisecond
)

type aivoice2 struct{}

func New() fairy.Fairy {
	return &aivoice2{}
}

func (vp *aivoice2) IsTarget(hwnd win32.HWND, exePath string) bool {
	if filepath.Base(exePath) != exeName {
		return false
	}
	return strings.Contains(internal.GetWindowText(hwnd), mainWindowName)
}

func (vp *aivoice2) TestedProgram() string {
	return "A.I.VOICE 2.10.1"
}

func (vp *aivoice2) Execute(hwnd win32.HWND, namer func(name, text string) (string, error)) error {
	var pid uint32
	win32.GetWindowThreadProcessId(hwnd, &pid)

	uia, err := internal.New()
	if err != nil {
		return fmt.Errorf("failed to create IUIAutomation: %w", err)
	}

	// find main window
	var mainWindow *mainWindow
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("timeout occurred while waiting for the main window to be found: %w", err)
		}
		mainWindow, err = newMainWindow(uia, hwnd)
		if err != nil {
			continue
		}
		break
	}

	defer mainWindow.Release()

	// click export button
	err = mainWindow.export.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	// find export dialog
	var exportDialog *exportDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for export dialog creation timed out: %w", err)
		}
		exportDialog, err = findExportDialog(uia, mainWindow.window)
		if err != nil {
			continue
		}
		break
	}
	defer exportDialog.Release()

	// extract save directory
	dummyPath, err := namer("temp", "temp")
	if err != nil {
		return fmt.Errorf("failed to build dummy filename: %w", err)
	}
	dir := filepath.Dir(dummyPath)

	// read edit text
	var str string
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for edit control to be found timed out: %w", err)
		}
		str, err = exportDialog.edit.GetTextViaValuePattern()
		if err != nil {
			continue
		}
		break
	}

	if str != dir {
		exportDialog.edit.SetFocus()

		for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
			if time.Now().After(deadLine) {
				return fmt.Errorf("waiting for clear edit control timed out: %w", err)
			}
			_, err = internal.SendInput([]internal.Input{
				// forcibly releases specific keys (CONTROL, SHIFT, MENU, RMENU) to avoid conflicts when sending WM_KEYDOWN.
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_CONTROL, Flags: internal.KEYEVENTF_KEYUP}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_SHIFT, Flags: internal.KEYEVENTF_KEYUP}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_MENU, Flags: internal.KEYEVENTF_KEYUP}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_RMENU, Flags: internal.KEYEVENTF_KEYUP}},
				// Ctrl + A
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_CONTROL}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_A}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_A, Flags: internal.KEYEVENTF_KEYUP}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_CONTROL, Flags: internal.KEYEVENTF_KEYUP}},
				// Backspace
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_BACK}},
				{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_BACK, Flags: internal.KEYEVENTF_KEYUP}},
			})
			if err != nil {
				return fmt.Errorf("failed to send input: %v", err)
			}
			str, err = exportDialog.edit.GetTextViaValuePattern()
			if err != nil {
				return fmt.Errorf("failed to get edit text (clear): %w", err)
			}
			if str != "" {
				continue
			}
			break
		}

		// Is there a bug in the processing order of Flutter?
		// If you select all with Ctrl+A and then delete all with BS key input,
		// and then immediately enter a string with WM_CHAR, some characters may be missed or crashed.
		// The only way to deal with it is to add a delay.
		// This delay is quite large, but it should be fine because it is used only once when the settings are different.
		time.Sleep(windowUpdateInterval)

		err = exportDialog.edit.SetTextViaWMCharSimplePost(mainWindow.view, dir)
		if err != nil {
			return fmt.Errorf("failed to set text to edit")
		}

		for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
			if time.Now().After(deadLine) {
				return fmt.Errorf("waiting for apply changes timed out: %w", err)
			}
			str, err = exportDialog.edit.GetTextViaValuePattern()
			if err != nil {
				return fmt.Errorf("failed to get edit text (verify): %w", err)
			}
			if str != dir {
				continue
			}
			break
		}
	}

	// click export button
	err = exportDialog.export.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	win32.SetForegroundWindow(hwnd)
	return nil
}
