package voicepeak

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/oov/forcepser/fairy"
	"github.com/oov/forcepser/fairy/internal"
	"github.com/zzl/go-win32api/win32"
)

var (
	exeName = "voicepeak.exe"

	mainWindowName           = "Voicepeak"
	mainWindowFramework      = "JUCE"
	mainWindowIconButtonName = "ImageIconButton"

	// It seems that there is also a Chinese version of VOICEPEAK,
	// but since there is no information, so cannot be supported yet.
	exportDialogTitles         = []string{"Export Settings", "出力設定"}
	exportDialogFilenameLabels = []string{"Save as", "ファイル名"}
	exportDialogButtonCaptions = []string{"Export", "出力"}

	blockExportMenuItemCaptions = []string{"Export Block", "ブロックの出力"}

	folderSelectDialogClass     = "#32770"
	folderSelectDialogFramework = "Win32"
	folderSelectDialogEditID    = 1152
	folderSelectDialogButtonID  = 1

	windowCreationTimeout       = 5 * time.Second
	windowCreationCheckInterval = 40 * time.Millisecond
	fileCreationTimeout         = 1 * time.Minute
	fileCreationCheckInterval   = 100 * time.Millisecond
)

type voicepeak struct{}

func New() fairy.Fairy {
	return &voicepeak{}
}

func (vp *voicepeak) IsTarget(exePath string) bool {
	return filepath.Base(exePath) == exeName
}

func (vp *voicepeak) TestedProgram() string {
	return "VOICEPEAK 1.1.0b2"
}

func (vp *voicepeak) Execute(hwnd win32.HWND, namer func(name, text string) (string, error)) error {
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

	// get character name / text
	name, err := mainWindow.combo.GetName()
	if err != nil {
		return fmt.Errorf("failed to get character name: %w", err)
	}
	if name == "" {
		return fmt.Errorf("character name is empty")
	}
	text, err := mainWindow.edit.GetTextViaTextPattern()
	if err != nil {
		return fmt.Errorf("failed to get text: %w", err)
	}
	if text == "" {
		return fmt.Errorf("text is empty")
	}

	// get current caret position
	tr, err := mainWindow.edit.GetFirstSelection()
	if err != nil {
		return fmt.Errorf("failed to get edit text pattern: %w", err)
	}
	if tr != nil {
		// restore focus and caret on exit, if available
		defer func() {
			mainWindow.edit.SetFocus()
			tr.Select()
			tr.Release()
		}()
	}

	// build filename
	wavPath, err := namer(name, text)
	if err != nil {
		return fmt.Errorf("failed to build filename: %w", err)
	}
	_, err = os.Stat(wavPath)
	if err == nil {
		return fmt.Errorf("file %v already exists: %w", wavPath, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to test file %v: %w", wavPath, err)
	}

	// click block menu button
	err = mainWindow.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click block menu button: %w", err)
	}

	// find menu window
	var blockMenuWindow *blockMenu
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for block menu window creation timed out: %w", err)
		}
		blockMenuWindow, err = findBlockMenu(uia, pid, hwnd)
		if err != nil {
			continue
		}
		break
	}
	defer blockMenuWindow.Release()

	// click block export menu item
	err = blockMenuWindow.export.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click block export menu item: %w", err)
	}

	// find block export dialog
	var blockExportDialog *blockExportDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for block export dialog creation timed out: %w", err)
		}
		blockExportDialog, err = findBlockExportDialog(uia, pid, hwnd)
		if err != nil {
			continue
		}
		break
	}
	defer blockExportDialog.Release()

	// set export file name
	behwnd, err := blockExportDialog.window.GetNativeWindowHandle()
	if err != nil {
		return fmt.Errorf("failed to get dialog window handle: %w", err)
	}
	err = blockExportDialog.edit.SetTextViaWMChar(behwnd, filepath.Base(changeExt(wavPath, "")))
	if err != nil {
		return fmt.Errorf("failed to set text: %w", err)
	}

	// click export button
	err = blockExportDialog.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click export button: %w", err)
	}

	// find folder select dialog
	var folderSelectDialog *folderSelectDialog
	for deadLine := time.Now().Add(windowCreationTimeout); ; time.Sleep(windowCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for folder select dialog creation timed out: %w", err)
		}
		folderSelectDialog, err = findFolderSelectDialog(uia, pid, hwnd)
		if err != nil {
			continue
		}
		break
	}
	defer folderSelectDialog.Release()

	// input export folder
	err = folderSelectDialog.edit.SetTextViaValuePattern(filepath.Dir(wavPath))
	if err != nil {
		return fmt.Errorf("failed to input export folder: %w", err)
	}

	mainWindow.window.SetEnable(true)

	// click button
	err = folderSelectDialog.button.Invoke()
	if err != nil {
		return fmt.Errorf("failed to click select button: %w", err)
	}

	// wait file creation
	for deadLine := time.Now().Add(fileCreationTimeout); ; time.Sleep(fileCreationCheckInterval) {
		if time.Now().After(deadLine) {
			return fmt.Errorf("waiting for file creation timed out: %w", os.ErrDeadlineExceeded)
		}
		f, err := os.OpenFile(wavPath, 0666, fs.FileMode(os.O_RDWR|os.O_APPEND))
		if err == nil {
			f.Close()
			break
		}
	}

	// create text file
	f, err := os.Create(changeExt(wavPath, ".txt"))
	if err != nil {
		return fmt.Errorf("failed to create text file: %w", err)
	}
	defer f.Close()

	_, err = f.Write([]byte{0xef, 0xbb, 0xbf})
	if err != nil {
		return fmt.Errorf("failed to write UTF-8 BOM: %w", err)
	}

	_, err = f.WriteString(text)
	if err != nil {
		return fmt.Errorf("failed to write text: %w", err)
	}

	return nil
}
