package main

import (
	"encoding/binary"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

type asas struct {
	Exe    string
	Folder string
	Filter string
	Format string
	Flags  int
}

func (a *asas) getASASName() (string, error) {
	h := fnv.New32a()
	if _, err := h.Write([]byte(a.Exe)); err != nil {
		return "", err
	}
	return "forcepser" + strconv.FormatUint(uint64(h.Sum32()), 16), nil
}

func writeStr(p []byte, s string) error {
	u16, err := windows.UTF16FromString(s)
	if err != nil {
		return err
	}
	if len(u16) > windows.MAX_PATH {
		return errors.New("string is too long")
	}
	for idx, ch := range u16 {
		binary.LittleEndian.PutUint16(p[idx*2:], ch)
	}
	return nil
}

func (a *asas) UpdateRunning() (bool, error) {
	asasName, err := a.getASASName()
	if err != nil {
		return false, err
	}
	mutexName, err := windows.UTF16PtrFromString("ASAS-" + asasName + "-Mutex")
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
	defer windows.CloseHandle(mutex)

	fileMappingName, err := windows.UTF16PtrFromString("ASAS-" + asasName)
	if err != nil {
		return false, err
	}
	fmo, err := openFileMapping(windows.FILE_MAP_WRITE, 0, fileMappingName)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(fmo)

	if _, err = windows.WaitForSingleObject(mutex, windows.INFINITE); err != nil {
		return false, err
	}
	defer windows.ReleaseMutex(mutex)

	p, err := windows.MapViewOfFile(fmo, windows.FILE_MAP_WRITE, 0, 0, 0)
	if err != nil {
		return false, err
	}
	defer windows.UnmapViewOfFile(p)

	var m []byte
	mh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	mh.Data = p
	mh.Len = 8 + windows.MAX_PATH*2*3
	mh.Cap = mh.Len
	if binary.LittleEndian.Uint32(m[0:]) != 0 {
		return false, errors.New("unknown api version")
	}

	binary.LittleEndian.PutUint32(m[4:], uint32(a.Flags))
	if err = writeStr(m[8:], a.Filter); err != nil {
		return false, err
	}
	if err = writeStr(m[8+windows.MAX_PATH*2:], a.Folder); err != nil {
		return false, err
	}
	if err = writeStr(m[8+windows.MAX_PATH*2*2:], a.Format); err != nil {
		return false, err
	}
	if err = windows.FlushViewOfFile(p, 0); err != nil {
		return false, err
	}
	return true, nil
}

func (a *asas) Exists() bool {
	_, err := os.Stat(a.Exe)
	return err == nil
}

func (a *asas) ConfirmAndRun(updateOnly bool) (bool, error) {
	r, err := a.UpdateRunning()
	if err != nil {
		return false, err
	}
	if r || updateOnly {
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
	asasName, err := a.getASASName()
	if err != nil {
		return false, err
	}
	cmd.Env = append(os.Environ(),
		"ASAS="+asasName,
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
