package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oov/forcepser/fairy"
	"github.com/oov/forcepser/fairy/voicepeak/v2"
	"github.com/oov/forcepser/fairy/voisonatalk/v1"
	"github.com/oov/forcepser/hotkey"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/color"
	"github.com/oov/audio/wave"
	"github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
	"github.com/zzl/go-win32api/win32"
)

const maxRetry = 10
const maxStay = 20
const resendProtectDuration = 5 * time.Second
const writeNotificationDeadline = 5 * time.Second

var verbose bool
var preventClear bool
var version string

type file struct {
	Filepath string
	Hash     string
	ModDate  time.Time
	TryCount int
}

type fileState struct {
	Retry int
	Stay  int
}

type sentFileState struct {
	At   time.Time
	Hash string
}

type colorizer interface {
	Renderln(a ...interface{}) string
	Sprintf(format string, args ...interface{}) string
}

type dummyColorizer struct{}

func (dummyColorizer) Renderln(a ...interface{}) string {
	if len(a) == 0 {
		return ""
	}
	if len(a) == 1 {
		return fmt.Sprint(a...)
	}
	r := fmt.Sprintln(a...)
	return r[:len(r)-1]
}

func (dummyColorizer) Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

var (
	warn     colorizer = color.Yellow
	caption  colorizer = color.Bold
	suppress colorizer = color.Gray
)

func clearScreen() error {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func verifyAndCalcHash(wavPath string, txtPath string) (string, error) {
	txt, err := os.OpenFile(txtPath, os.O_RDWR, 0666)
	if err != nil {
		return "", fmt.Errorf("テキストファイルが開けませんでした: %w", err)
	}
	defer txt.Close()
	wav, err := os.OpenFile(wavPath, os.O_RDWR, 0666)
	if err != nil {
		return "", fmt.Errorf("waveファイルが開けませんでした: %w", err)
	}
	defer wav.Close()
	r, wfe, err := wave.NewLimitedReader(wav)
	if err != nil {
		return "", fmt.Errorf("waveファイルが読み取れませんでした: %w", err)
	}
	if r.N == 0 || wfe.Format.SamplesPerSec == 0 || wfe.Format.Channels == 0 || wfe.Format.BitsPerSample == 0 {
		return "", fmt.Errorf("waveファイルに記録されている値が不正です: %w", err)
	}
	if _, err := wav.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("waveファイルの読み取りカーソルを移動できませんでした: %w", err)
	}
	h := fnv.New32a()
	if _, err := io.Copy(h, wav); err != nil {
		return "", fmt.Errorf("waveファイルが読み取れませんでした: %w", err)
	}
	h2 := fnv.New32a()
	if _, err := io.Copy(h2, txt); err != nil {
		return "", fmt.Errorf("テキストファイルが読み取れませんでした: %w", err)
	}
	return string(h2.Sum(h.Sum(nil))), nil
}

func processFiles(L *lua.LState, files []file, sort string, recentChanged map[string]fileState, recentSent map[string]sentFileState) (needRetry bool, err error) {
	var errStay error
	defer func() {
		for k, ct := range recentChanged {
			if ct.Retry == maxRetry-1 || ct.Stay == maxStay-1 {
				log.Println(warn.Renderln("  たくさん失敗したのでこのファイルは諦めます:", k))
				delete(recentChanged, k)
				continue
			}
			if errStay != nil {
				ct.Stay += 1
			} else {
				ct.Retry += 1
			}
			recentChanged[k] = ct
			needRetry = true
		}
	}()

	proj, err := readGCMZDropsData()
	if err != nil {
		if verbose {
			log.Println(suppress.Renderln("プロジェクト情報取得失敗:", err))
		}
		err = fmt.Errorf("ごちゃまぜドロップス v0.3 以降がインストールされた AviUtl が検出できませんでした")
		return
	}
	if proj.Width == 0 {
		err = fmt.Errorf("AviUtl で編集中のプロジェクトが見つかりません")
		return
	}
	t := L.NewTable()
	for _, f := range files {
		file := L.NewTable()
		file.RawSetString("path", lua.LString(f.Filepath))
		file.RawSetString("hash", lua.LString(f.Hash))
		file.RawSetString("trycount", lua.LNumber(f.TryCount))
		file.RawSetString("maxretry", lua.LNumber(maxRetry))
		file.RawSetString("moddate", lua.LNumber(float64(f.ModDate.Unix())+(float64(f.ModDate.Nanosecond())/1e9)))
		t.Append(file)
	}
	pt := L.NewTable()
	pt.RawSetString("projectfile", lua.LString(proj.ProjectFile))
	pt.RawSetString("gcmzapiver", lua.LNumber(proj.GCMZAPIVer))
	pt.RawSetString("flags", lua.LNumber(proj.Flags))
	pt.RawSetString("flags_englishpatched", lua.LBool(proj.Flags&1 == 1))
	pt.RawSetString("window", lua.LNumber(proj.Window))
	pt.RawSetString("width", lua.LNumber(proj.Width))
	pt.RawSetString("height", lua.LNumber(proj.Height))
	pt.RawSetString("video_rate", lua.LNumber(proj.VideoRate))
	pt.RawSetString("video_scale", lua.LNumber(proj.VideoScale))
	pt.RawSetString("audio_rate", lua.LNumber(proj.AudioRate))
	pt.RawSetString("audio_ch", lua.LNumber(proj.AudioCh))
	if err = L.CallByParam(lua.P{
		Fn:      L.GetGlobal("changed"),
		NRet:    1,
		Protect: true,
	}, t, lua.LString(sort), pt); err != nil {
		return
	}
	rv := L.ToTable(-1)
	if rv == nil {
		err = fmt.Errorf("処理後の戻り値が異常です")
		return
	}
	// remove processed entries
	now := time.Now()
	n := rv.MaxN()
	for i := 1; i <= n; i++ {
		tbl, ok := rv.RawGetInt(i).(*lua.LTable)
		if !ok {
			continue
		}
		// remove it from the candidate list regardless of success or failure.
		src := tbl.RawGetString("src").String()
		hash := tbl.RawGetString("hash").String()
		delete(recentChanged, src)
		recentSent[src] = sentFileState{
			At:   now,
			Hash: hash,
		}

		destV := tbl.RawGetString("dest")
		if destV.Type() != lua.LTString {
			continue // rule not found
		}
		// if TTS software creates files in the same location as the project,
		// files that have already been processed may be subject to processing again.
		// put dest on recentSent to prevent it.
		dest := destV.String()
		recentSent[dest] = sentFileState{
			At:   now,
			Hash: hash,
		}
		delete(recentChanged, dest)
	}
	L.Pop(1)
	return
}

func getProjectPath() string {
	proj, err := readGCMZDropsData()
	if err != nil {
		return ""
	}
	if proj.Width == 0 {
		return ""
	}
	if proj.GCMZAPIVer < 1 {
		return ""
	}
	return proj.ProjectFile
}

func getNamer(dir string) func(name, text string) (string, error) {
	return func(name, text string) (string, error) {
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
			dir,
			fmt.Sprintf("%d_%s_%s.wav", time.Now().Unix(), name, string(shortText)),
		), nil
	}
}

func watchFairyCall(ctx context.Context, notify chan<- map[string]struct{}, hk *hotkey.Hotkey, namer func(name, text string) (string, error)) {
	for {
		select {
		case complete := <-hk.Notify:
			if err := (fairy.Fairies{voicepeak.New(), voisonatalk.New()}).Execute(namer); err != nil {
				if !errors.Is(err, fairy.ErrTargetNotFound) {
					log.Println(warn.Renderln("  フェアリー: 処理を完遂できませんでした:", err))
				} else {
					log.Println(warn.Renderln("  フェアリー: アクティブなウィンドウが処理対象ではありません。"))
				}
			}
			complete()
		case <-ctx.Done():
			return
		}
	}
}

func watchProjectPath(ctx context.Context, notify chan<- map[string]struct{}, projectPath string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			if projectPath != getProjectPath() {
				if verbose {
					log.Println(suppress.Renderln("  AviUtl のプロジェクトパスの変更を検出しました"))
				}
				notify <- nil
				return
			}
		}
	}
}

func watch(ctx context.Context, watcher *fsnotify.Watcher, settingWatcher *fsnotify.Watcher, notify chan<- map[string]struct{}, settingFile string, freshness float64, sortdelay float64) {
	defer close(notify)
	var finish bool
	changed := map[string]struct{}{}
	timer := time.NewTimer(time.Duration(sortdelay) * time.Second)
	timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watcher.Events:
			if verbose {
				log.Println(suppress.Renderln("イベント検証:", event))
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				if verbose {
					log.Println(suppress.Renderln("  オペレーションが Create / Write ではないので何もしません"))
				}
				continue
			}
			ext := strings.ToLower(filepath.Ext(event.Name))
			if ext != ".wav" && ext != ".txt" {
				if verbose {
					log.Println(suppress.Renderln("  *.wav / *.txt のどちらでもないので何もしません"))
				}
				continue
			}
			st, err := os.Stat(event.Name)
			if err != nil {
				if verbose {
					log.Println(suppress.Renderln("  更新日時の取得に失敗したので何もしません"))
				}
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Even if freshness == 0, verify when notified by Write.
				// Because Write notification is also sent in the case of a file attribute change or modify zone identifier.
				// See: https://github.com/oov/forcepser/issues/10
				if time.Since(st.ModTime()) > writeNotificationDeadline {
					if verbose {
						log.Println(suppress.Renderln("  ファイル変更通知がありましたが更新日時が", writeNotificationDeadline, "以上前なので何もしません"))
					}
					continue
				}
			} else {
				if freshness > 0 {
					if math.Abs(time.Since(st.ModTime()).Seconds()) > freshness {
						if verbose {
							log.Println(suppress.Renderln("  更新日時が", freshness, "秒以上前なので何もしません"))
						}
						continue
					}
				}
			}
			if verbose {
				log.Println(suppress.Renderln("  送信ファイル候補にします"))
			}
			changed[event.Name[:len(event.Name)-len(ext)]+".wav"] = struct{}{}
			timer.Reset(time.Duration(sortdelay) * time.Second)
		case event := <-settingWatcher.Events:
			if event.Name == settingFile {
				if verbose {
					log.Println(suppress.Renderln("  設定ファイルの再読み込みとして処理します"))
				}
				finish = true
				timer.Reset(100 * time.Millisecond)
				continue
			}
		case err := <-watcher.Errors:
			log.Println(warn.Renderln("監視中にエラーが発生しました:", err))
		case err := <-settingWatcher.Errors:
			log.Println(warn.Renderln("監視中にエラーが発生しました:", err))
		case <-timer.C:
			if finish {
				notify <- nil
				continue
			}
			notify <- changed
			changed = map[string]struct{}{}
		}
	}
}

func bool2str(b bool, t string, f string) string {
	if b {
		return t
	}
	return f
}

func printDetails(setting *setting, tempDir string) {
	log.Println(caption.Renderln("AviUtl プロジェクト情報:"))
	proj, err := readGCMZDropsData()
	if err != nil {
		log.Println(warn.Renderln("  ごちゃまぜドロップス v0.3 以降がインストールされた AviUtl が見つかりません"))
	} else {
		if proj.Width == 0 {
			log.Println(warn.Renderln("  AviUtl で編集中のプロジェクトが見つかりません"))
		} else {
			if proj.GCMZAPIVer < 1 {
				log.Println(warn.Renderln("  ごちゃまぜドロップスのバージョンが古いため一部の機能が使用できません"))
			} else {
				log.Println(suppress.Renderln("  ProjectFile:"), proj.ProjectFile)
			}
			log.Println(suppress.Renderln("  Window:     "), int(proj.Window))
			log.Println(suppress.Renderln("  Width:      "), proj.Width)
			log.Println(suppress.Renderln("  Height:     "), proj.Height)
			log.Println(suppress.Renderln("  VideoRate:  "), proj.VideoRate)
			log.Println(suppress.Renderln("  VideoScale: "), proj.VideoScale)
			log.Println(suppress.Renderln("  AudioRate:  "), proj.AudioRate)
			log.Println(suppress.Renderln("  AudioCh:    "), proj.AudioCh)
			if proj.GCMZAPIVer >= 2 {
				log.Println(suppress.Renderln("  Flags:      "), proj.Flags)
			}
			log.Println()
		}
	}

	log.Println(caption.Renderln("環境変数:"))
	log.Println(suppress.Renderln("  %BASEDIR%:   "), setting.BaseDir)
	log.Println(suppress.Renderln("  %TEMPDIR%:   "), tempDir)
	log.Println(suppress.Renderln("  %PROJECTDIR%:"), setting.projectDir)
	log.Println(suppress.Renderln("  %PROFILE%:   "), getSpecialFolderPath(CSIDL_PROFILE))
	log.Println(suppress.Renderln("  %DESKTOP%:   "), getSpecialFolderPath(CSIDL_DESKTOP))
	log.Println(suppress.Renderln("  %MYDOC%:     "), getSpecialFolderPath(CSIDL_PERSONAL))
	log.Println()

	log.Println(suppress.Renderln("  delta:"), setting.Delta)
	log.Println(suppress.Renderln("  freshness:"), setting.Freshness)
	log.Println()

	log.Println(caption.Renderln("フェアリーコール:"))
	if setting.FairyCall != "" {
		log.Println(suppress.Renderln("  呼び出しキー: "), setting.FairyCall)
	} else {
		log.Println(suppress.Renderln("  呼び出しキーの設定が行われていないため使用できません"))
	}
	log.Println()

	for i, a := range setting.Asas {
		log.Println(caption.Sprintf("Asas %d:", i+1))
		log.Println(suppress.Renderln("  対象EXE:"), a.Exe)
		log.Println(suppress.Renderln("  フィルター:"), a.Filter)
		log.Println(suppress.Renderln("  保存先フォルダー:"), a.ExpandedFolder())
		log.Println(suppress.Renderln("  フォーマット:"), a.Format)
		log.Println(suppress.Renderln("  フラグ:"), a.Flags)
		if !a.Exists() {
			log.Println(warn.Renderln("  [警告] 対象EXE が見つからないため設定を無視します"))
		}
	}
	log.Println()
	for i, r := range setting.Rule {
		log.Println(caption.Sprintf("ルール%d:", i+1))
		log.Println(suppress.Renderln("  対象フォルダー:"), r.ExpandedDir())
		log.Println(suppress.Renderln("  対象ファイル名:"), r.File)
		log.Println(suppress.Renderln("  テキストファイルの文字コード:"), r.Encoding)
		if r.textRE != nil {
			log.Println(suppress.Renderln("  テキスト判定用の正規表現:"), r.Text)
		}
		log.Println(suppress.Renderln("  挿入先レイヤー:"), r.Layer)
		log.Println(suppress.Renderln("  modifier:"), bool2str(r.Modifier != "", "あり", "なし"))
		log.Println(suppress.Renderln("  ユーザーデータ:"), r.UserData)
		log.Println(suppress.Renderln("  パディング:"), r.Padding)
		log.Println(suppress.Renderln("  EXOファイル:"), r.ExoFile)
		log.Println(suppress.Renderln("  Luaファイル:"), r.LuaFile)
		log.Println(suppress.Renderln("  Waveファイルの移動:"), r.FileMove.Readable())
		if r.FileMove != "off" {
			log.Println(suppress.Sprintf("    %s先:", r.FileMove.Readable()), r.ExpandedDestDir())
		}
		log.Println(suppress.Renderln("  テキストファイルの削除:"), bool2str(r.DeleteText, "する", "しない"))
		if !r.ExistsDir() {
			log.Println(warn.Renderln("  [警告] 対象フォルダー が見つからないため設定を無視します"))
		}
	}
	log.Println()
}

func loadSetting(path string, tempDir string, projectDir string) (*setting, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return newSetting(f, tempDir, projectDir)
}

func tempSetting(tempDir string, projectDir string) (*setting, error) {
	return newSetting(strings.NewReader(``), tempDir, projectDir)
}

func process(watcher *fsnotify.Watcher, settingWatcher *fsnotify.Watcher, settingFile string, recentChanged map[string]fileState, recentSent map[string]sentFileState, loop int) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("exe ファイルのパスが取得できません: %w", err)
	}
	tempDir := filepath.Join(filepath.Dir(exePath), "tmp")
	if err = os.Mkdir(tempDir, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("tmp フォルダの作成に失敗しました: %w", err)
	}

	projectPath := getProjectPath()
	var projectDir string
	if projectPath != "" {
		projectDir = filepath.Dir(projectPath)
	}

	setting, err := loadSetting(settingFile, tempDir, projectDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
		}
		log.Println(warn.Renderln("設定ファイルが開けませんでした。"))
		log.Println(suppress.Renderln(filepath.Base(settingFile), "を作成すると自動で読み込みます。"))
		log.Println()
		setting, _ = tempSetting(tempDir, projectDir)
	} else {
		printDetails(setting, tempDir)
	}

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("re", gluare.Loader)
	err = L.DoString(`re = require("re")`)
	if err != nil {
		return fmt.Errorf("スクリプト環境の初期化中にエラーが発生しました: %w", err)
	}

	L.SetGlobal("debug_print", L.NewFunction(luaDebugPrint))
	L.SetGlobal("debug_error", L.NewFunction(luaDebugError))
	L.SetGlobal("debug_print_verbose", L.NewFunction(luaDebugPrintVerbose))
	L.SetGlobal("sendfile", L.NewFunction(luaSendFile))
	L.SetGlobal("findrule", L.NewFunction(luaFindRule(setting)))
	L.SetGlobal("getaudioinfo", L.NewFunction(luaGetAudioInfo))
	L.SetGlobal("tosjis", L.NewFunction(luaToSJIS))
	L.SetGlobal("fromsjis", L.NewFunction(luaFromSJIS))
	L.SetGlobal("toexostring", L.NewFunction(luaToEXOString))
	L.SetGlobal("fromexostring", L.NewFunction(luaFromEXOString))
	L.SetGlobal("tofilename", L.NewFunction(luaToFilename))
	L.SetGlobal("replaceenv", L.NewFunction(luaReplaceEnv(setting)))

	if err := L.DoFile("_entrypoint.lua"); err != nil {
		return fmt.Errorf("_entrypoint.lua の実行中にエラーが発生しました: %w", err)
	}

	updateOnly := loop > 0
	for _, a := range setting.Asas {
		if a.Exists() {
			if _, err := a.ConfirmAndRun(updateOnly); err != nil {
				return fmt.Errorf("プログラムの起動に失敗しました: %w", err)
			}
		}
	}

	log.Println(caption.Sprintf("監視を開始します:"))
	var hk *hotkey.Hotkey
	if setting.FairyCall != "" {
		hk, err = hotkey.New(setting.FairyCall)
		if err == nil {
			defer hk.Close()
		}
		if err != nil {
			var msg string
			if errors.Is(err, win32.ERROR_HOTKEY_ALREADY_REGISTERED) {
				msg = "ショートカットキーが既に使用されています"
			} else {
				msg = err.Error()
			}
			log.Println(warn.Renderln("  [警告] フェアリーコール呼び出しキーの登録に失敗しました:", msg))
		}
	}

	watching := 0
	for _, dir := range setting.Dirs() {
		err = watcher.Add(dir)
		if err != nil {
			return fmt.Errorf("フォルダー %s が監視できません: %w", dir, err)
		}
		log.Println("  " + dir)
		watching++
		defer watcher.Remove(dir)
	}
	if watching == 0 {
		log.Println(warn.Renderln("  [警告] 監視対象のフォルダーがひとつもありません"))
	}
	notify := make(chan map[string]struct{}, 10000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchProjectPath(ctx, notify, projectPath)
	if hk != nil {
		go watchFairyCall(ctx, notify, hk, getNamer(tempDir))
	}
	go watch(ctx, watcher, settingWatcher, notify, settingFile, setting.Freshness, setting.SortDelay)
	timer := time.NewTimer(time.Duration(setting.SortDelay) * time.Second)
	timer.Stop()
	timerAt := time.Now()
	for {
		select {
		case updatedFiles, ok := <-notify:
			if !ok {
				return nil
			}
			if updatedFiles == nil {
				if preventClear {
					return nil
				}
				return clearScreen()
			}
			now := time.Now()
			for k, st := range recentSent {
				if now.Sub(st.At) > resendProtectDuration {
					delete(recentSent, k)
				}
			}
			for k := range updatedFiles {
				if _, ok := recentChanged[k]; !ok {
					recentChanged[k] = fileState{}
				}
			}
			d := time.Duration(setting.SortDelay) * time.Second
			at := time.Now().Add(d)
			if at.After(timerAt) {
				timerAt = at
				timer.Reset(d)
			}
		case <-timer.C:
			var files []file
			var needRetry bool
			for wavPath, fState := range recentChanged {
				if verbose {
					log.Println(suppress.Renderln("送信ファイル候補検証:", wavPath))
				}
				if fState.Stay == maxStay {
					log.Println(warn.Renderln("  以下のファイルは長時間準備が整わなかったので一旦諦めます"))
					log.Println("    ", wavPath)
					delete(recentChanged, wavPath)
					continue
				}

				txtPath := changeExt(wavPath, ".txt")
				s1, e1 := os.Stat(wavPath)
				s2, e2 := os.Stat(txtPath)
				if e1 != nil || e2 != nil {
					// Whenever this issue is resolved, an Create/Write event will occur.
					// So we ignore it for now.
					if verbose {
						log.Println(suppress.Renderln("  *.wav と *.txt が揃ってないので無視します"))
					}
					delete(recentChanged, wavPath)
					continue
				}
				s1Mod := s1.ModTime()
				s2Mod := s2.ModTime()
				if setting.Delta > 0 {
					// Whenever this issue is resolved, an Write event will occur.
					// So we ignore it for now.
					if math.Abs(s1Mod.Sub(s2Mod).Seconds()) > setting.Delta {
						if verbose {
							log.Println(suppress.Renderln("  *.wav と *.txt の更新日時の差が", setting.Delta, "秒以上なので無視します"))
						}
						delete(recentChanged, wavPath)
						continue
					}
				}
				hash, err := verifyAndCalcHash(wavPath, txtPath)
				if err != nil {
					if verbose {
						log.Println(suppress.Renderln("  まだファイルの準備が整わないので保留にします"))
						log.Println(suppress.Renderln("    理由:", err))
					}
					fState.Stay++
					recentChanged[wavPath] = fState
					d := 500 * time.Millisecond
					at := time.Now().Add(d)
					if at.After(timerAt) {
						timerAt = at
						timer.Reset(d)
					}
					needRetry = true
				}
				if st, found := recentSent[wavPath]; found && st.Hash == hash {
					if verbose {
						log.Println(suppress.Renderln("  つい最近送ったファイルなので、重複送信回避のために無視します"))
					}
					delete(recentChanged, wavPath)
					continue
				}
				if verbose {
					log.Println(suppress.Renderln("このファイルはルール検索対象です"))
				}
				files = append(files, file{wavPath, hash, s1Mod, fState.Retry})
			}
			if needRetry || len(files) == 0 {
				continue
			}
			needRetry, err = processFiles(L, files, setting.Sort, recentChanged, recentSent)
			if err != nil {
				log.Println("ファイルの処理中にエラーが発生しました:", err)
			}
			if needRetry {
				d := 500 * time.Millisecond
				at := time.Now().Add(d)
				if at.After(timerAt) {
					timerAt = at
					timer.Reset(d)
				}
			}
		}
	}
}

func main() {
	if _, ok := os.LookupEnv("ASAS"); ok {
		// asas emulation mode
		if err := emulateAsas(); err != nil {
			log.Fatalln(err)
		}
		return
	}
	if hr := win32.CoInitializeEx(nil, win32.COINIT_MULTITHREADED); win32.FAILED(hr) {
		log.Fatalln("CoInitializeEx に失敗しました:", win32.HRESULT_ToString(hr))
	}
	defer win32.CoUninitialize()

	var mono bool
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&mono, "m", false, "disable color")
	flag.BoolVar(&preventClear, "prevent-clear", false, "prevent clear screen on reload")
	flag.Parse()

	if mono {
		warn = dummyColorizer{}
		caption = dummyColorizer{}
		suppress = dummyColorizer{}
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalln("exe ファイルのパスが取得できません", err)
	}

	settingFile := flag.Arg(0)
	if settingFile == "" {
		settingFile = filepath.Join(filepath.Dir(exePath), "setting.txt")
	}
	if !filepath.IsAbs(settingFile) {
		p, err := filepath.Abs(settingFile)
		if err != nil {
			log.Fatalln("filepath.Abs に失敗しました:", err)
		}
		settingFile = p
	}

	if err := os.Chdir(filepath.Dir(exePath)); err != nil {
		log.Fatalln("カレントディレクトリの変更に失敗しました:", err)
	}

	settingWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("fsnotify.NewWatcher に失敗しました:", err)
	}
	defer settingWatcher.Close()

	err = settingWatcher.Add(filepath.Dir(settingFile))
	if err != nil {
		log.Fatalln("設定ファイルフォルダーの監視に失敗しました:", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("fsnotify.NewWatcher に失敗しました:", err)
	}
	defer watcher.Close()

	recentChanged := map[string]fileState{}
	recentSent := map[string]sentFileState{}
	for i := 0; ; i++ {
		log.Println(caption.Renderln("かんしくん"), version)
		if verbose {
			log.Println(warn.Renderln("冗長ログモードが有効"))
		}
		log.Println(suppress.Renderln("  設定ファイル:"), settingFile)
		log.Println()
		err = process(watcher, settingWatcher, settingFile, recentChanged, recentSent, i)
		if err != nil {
			log.Println(err)
			log.Println("3秒後にリトライします")
			time.Sleep(3 * time.Second)
		}
	}
}
