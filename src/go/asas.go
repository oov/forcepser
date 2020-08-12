package main

import (
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

type asas struct {
	Exe    string
	Folder string `default:"%TEMPDIR%"`
	Filter string `default:"*.wav"`
	Format string
	Flags  int `default:"-1"`
}

func (a *asas) getFMOName() (string, error) {
	h := fnv.New32a()
	if _, err := h.Write([]byte(a.Exe)); err != nil {
		return "", err
	}
	return "forcepser" + strconv.FormatUint(uint64(h.Sum32()), 16), nil
}

func (a *asas) IsRunning() (bool, error) {
	fmoName, err := a.getFMOName()
	if err != nil {
		return false, err
	}
	mutexName, err := windows.UTF16PtrFromString("ASAS-" + fmoName + "-Mutex")
	if err != nil {
		return false, err
	}

	mutex, err := windows.OpenMutex(windows.MUTEX_ALL_ACCESS, false, mutexName)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND {
			return false, nil
		}
		return false, err
	}
	windows.CloseHandle(mutex)
	return true, nil
}

func (a *asas) ConfirmAndRun() (bool, error) {
	r, err := a.IsRunning()
	if err != nil {
		return false, err
	}
	if r {
		return false, nil
	}
	msg, err := windows.UTF16PtrFromString("実行中の " + filepath.Base(a.Exe) + " が見つかりませんでした。\n起動しますか？")
	if err != nil {
		return false, err
	}
	title, err := windows.UTF16PtrFromString("かんしくん " + version)
	if err != nil {
		return false, err
	}
	hwnd := getConsoleWindow()
	setForegroundWindow(hwnd)
	resp, err := windows.MessageBox(hwnd, msg, title, windows.MB_ICONQUESTION|windows.MB_YESNO)
	if err != nil {
		return false, err
	}
	const IDNO = 7
	if resp == IDNO {
		return false, nil
	}
	return a.Run()
}

func (a *asas) Run() (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, errors.Wrap(err, "exe ファイルのパスが取得できません")
	}
	cmd := exec.Command(filepath.Join(filepath.Dir(exePath), "asas", "asas.exe"), a.Exe)
	fmoName, err := a.getFMOName()
	if err != nil {
		return false, err
	}
	cmd.Env = append(os.Environ(),
		"ASAS="+fmoName,
		"ASAS_FILTER="+a.Filter,
		"ASAS_FOLDER="+a.Folder,
		"ASAS_FORMAT="+a.Format,
		"ASAS_FLAGS="+strconv.Itoa(a.Flags),
	)
	cmd.Dir = filepath.Dir(a.Exe)
	if err = cmd.Start(); err != nil {
		return false, errors.Wrap(err, "asas.exe の実行に失敗しました")
	}
	go cmd.Wait()
	return true, nil
}
