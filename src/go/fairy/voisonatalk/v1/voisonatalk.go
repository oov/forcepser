package voisonatalk

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/oov/forcepser/fairy"
	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

var (
	exeName = "VoiSona Talk.exe"

	mainWindowName      = "VoiSona Talk Editor"
	mainWindowFramework = "JUCE"

	mainWindowTableHeaderEnableCaptions     = []string{"有効", "Enable"}
	mainWindowTableHeaderSentenceCaptions   = []string{"文", "Sentence"}
	mainWindowTableHeaderExportNameCaptions = []string{"出力名", "Export Name"}

	exportDialogTitles         = []string{"Export WAV Files", "WAVファイルをエクスポート"}
	exportDialogCheckBoxLabels = []string{"Also output text files", "テキストファイルも出力"}
	exportDialogButtonCaptions = []string{"Export...", "エクスポート..."}

	folderSelectDialogClass     = "#32770"
	folderSelectDialogFramework = "Win32"
	folderSelectDialogEditID    = 1152
	folderSelectDialogButtonID  = 1

	windowCreationTimeout       = 5 * time.Second
	windowCreationCheckInterval = 40 * time.Millisecond
)

type voisonatalk struct{}

func New() fairy.Fairy {
	return &voisonatalk{}
}

func (vp *voisonatalk) IsTarget(hwnd win32.HWND, exePath string) bool {
	return filepath.Base(exePath) == exeName
}

func (vp *voisonatalk) TestedProgram() string {
	return "VoiSona Talk Editor 1.2.10.2"
}

func (vp *voisonatalk) Execute(hwnd win32.HWND, namer func(name, text string) (string, error)) error {
	var pid uint32
	win32.GetWindowThreadProcessId(hwnd, &pid)

	uia, err := internal.New()
	if err != nil {
		return fmt.Errorf("failed to create IUIAutomation: %w", err)
	}

	// find main window
	mainWindow, err := newMainWindow(uia, hwnd)
	if err != nil {
		return fmt.Errorf("main window not found: %w", err)
	}
	defer mainWindow.Release()

	// disable during automation
	err = mainWindow.window.SetEnable(false)
	if err != nil {
		return fmt.Errorf("failed to disable window")
	}
	defer mainWindow.window.SetEnable(true)

	err = mainWindow.updateCheckBoxes(uia)
	if err != nil {
		return fmt.Errorf("failed to update checkboxes: %w", err)
	}

	trackName, err := mainWindow.activeTrackName.GetTextViaValuePattern()
	if err != nil {
		return fmt.Errorf("failed to get active track name: %w", err)
	}

	sentence, err := mainWindow.activeSentence.GetTextViaTextPattern()
	if err != nil {
		return fmt.Errorf("failed to get active sentence: %w", err)
	}

	dummyPath, err := namer(trackName, sentence)
	if err != nil {
		return fmt.Errorf("failed to build dummy filename: %w", err)
	}
	name := filepath.Base(dummyPath)
	err = mainWindow.activeExportName.SetTextViaWMChar(hwnd, name[:len(name)-len(filepath.Ext(name))])
	if err != nil {
		return fmt.Errorf("failed to update export name: %w", err)
	}

	err = mainWindow.activeSentence.SetTextViaWMChar(hwnd, "")
	if err != nil {
		return fmt.Errorf("failed to update export name: %w", err)
	}
	err = mainWindow.activeSentence.SetTextViaWMChar(hwnd, sentence)
	if err != nil {
		return fmt.Errorf("failed to update export name: %w", err)
	}

	// forcibly releases specific keys (CONTROL, SHIFT, MENU, RMENU) to avoid conflicts when sending WM_KEYDOWN.
	_, err = internal.SendInput([]internal.Input{
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_CONTROL, Flags: internal.KEYEVENTF_KEYUP}},
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_SHIFT, Flags: internal.KEYEVENTF_KEYUP}},
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_MENU, Flags: internal.KEYEVENTF_KEYUP}},
		{InputType: internal.INPUT_KEYBOARD, KI: internal.KeyboardInput{Vk: win32.VK_RMENU, Flags: internal.KEYEVENTF_KEYUP}},
	})
	if err != nil {
		return fmt.Errorf("failed to send input: %v", err)
	}

	// Push F4
	win32.SendMessage(hwnd, win32.WM_KEYDOWN, win32.WPARAM(win32.VK_F4), 1)
	win32.SendMessage(hwnd, win32.WM_KEYUP, win32.WPARAM(win32.VK_F4), 0xc0000001)

	// find block export dialog
	var exportDialog *exportDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for export dialog creation timed out: %w", err)
		}
		exportDialog, err = findExportDialog(uia, pid, hwnd)
		if err != nil {
			continue
		}
		break
	}
	defer exportDialog.Release()

	// enable text output
	chk, err := exportDialog.checkBox.GetCurrentPropertyStringValue(win32.UIA_ValueValuePropertyId)
	if err != nil {
		return fmt.Errorf("failed to get checkbox state: %w", err)
	}
	if chk != "On" {
		err = exportDialog.checkBox.Invoke()
		if err != nil {
			return fmt.Errorf("failed to set checkbox state: %w", err)
		}
	}

	// click export button
	err = exportDialog.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	// find folder select dialog
	var folderSelectDialog *folderSelectDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for folder select dialog creation timed out: %w", err)
		}
		folderSelectDialog, err = findFolderSelectDialog(uia, exportDialog.window)
		if err != nil {
			continue
		}
		break
	}
	defer folderSelectDialog.Release()

	// input export folder
	err = folderSelectDialog.edit.SetTextViaValuePattern(filepath.Dir(dummyPath))
	if err != nil {
		return fmt.Errorf("failed to input export folder: %w", err)
	}

	mainWindow.window.SetEnable(true)

	// click button
	err = folderSelectDialog.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click select button: %w", err)
	}

	win32.SetForegroundWindow(hwnd)
	return nil
}
