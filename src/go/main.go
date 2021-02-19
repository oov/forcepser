package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/color"
	"github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
)

var verbose bool

type file struct {
	Filepath string
	ModDate  time.Time
	TryCount int
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

func processFiles(L *lua.LState, files []file, sort string, recentChanged map[string]int, recentSent map[string]time.Time) (needRetry bool, err error) {
	defer func() {
		for k, ct := range recentChanged {
			if ct == 9 {
				log.Println(warn.Renderln("  たくさん失敗したのでこのファイルは諦めます:", k))
				delete(recentChanged, k)
				continue
			}
			recentChanged[k] = ct + 1
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
		file.RawSetString("trycount", lua.LNumber(f.TryCount))
		file.RawSetString("moddate", lua.LNumber(float64(f.ModDate.Unix())+(float64(f.ModDate.Nanosecond())/1e9)))
		t.Append(file)
	}
	pt := L.NewTable()
	pt.RawSetString("projectfile", lua.LString(proj.ProjectFile))
	pt.RawSetString("gcmzapiver", lua.LNumber(proj.GCMZAPIVer))
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
		k := rv.RawGetInt(i).String()
		recentSent[k] = now
		delete(recentChanged, k)
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
			if freshness > 0 {
				st, err := os.Stat(event.Name)
				if err != nil {
					if verbose {
						log.Println(suppress.Renderln("  更新日時の取得に失敗したので何もしません"))
					}
					continue
				}
				if math.Abs(time.Now().Sub(st.ModTime()).Seconds()) > freshness {
					if verbose {
						log.Println(suppress.Renderln("  更新日時が", freshness, "秒以上前なので何もしません"))
					}
					continue
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

func process(watcher *fsnotify.Watcher, settingWatcher *fsnotify.Watcher, settingFile string, recentChanged map[string]int, recentSent map[string]time.Time, loop int) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("exe ファイルのパスが取得できません: %w", err)
	}
	tempDir := filepath.Join(filepath.Dir(exePath), "tmp")

	projectPath := getProjectPath()
	var projectDir string
	if projectPath != "" {
		projectDir = filepath.Dir(projectPath)
	}

	setting, err := newSetting(settingFile, tempDir, projectDir)
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}
	printDetails(setting, tempDir)

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("re", gluare.Loader)
	err = L.DoString(`re = require("re")`)
	if err != nil {
		return fmt.Errorf("Lua スクリプト環境の初期化中にエラーが発生しました: %w", err)
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

	if err = os.Mkdir(tempDir, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("tmp フォルダの作成に失敗しました: %w", err)
	}

	log.Println(caption.Sprintf("監視を開始します:"))
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
	notify := make(chan map[string]struct{}, 32)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchProjectPath(ctx, notify, projectPath)
	go watch(ctx, watcher, settingWatcher, notify, settingFile, setting.Freshness, setting.SortDelay)
	for files := range notify {
		if files == nil {
			log.Println()
			log.Println("設定ファイルを再読み込みします")
			log.Println()
			return nil
		}

		now := time.Now()
		for k := range recentSent {
			if now.Sub(recentSent[k]) > 5*time.Second {
				delete(recentSent, k)
			}
		}
		for k := range files {
			if _, ok := recentChanged[k]; !ok {
				recentChanged[k] = 0
			}
		}
		var files []file
		for k, tryCount := range recentChanged {
			if verbose {
				log.Println(suppress.Renderln("送信ファイル候補検証:", k))
			}
			if _, found := recentSent[k]; found {
				if verbose {
					log.Println(suppress.Renderln("  つい最近送ったファイルなので、重複送信回避のために無視します"))
				}
				delete(recentChanged, k)
				continue
			}
			s1, e1 := os.Stat(k)
			s2, e2 := os.Stat(k[:len(k)-4] + ".txt")
			if e1 != nil || e2 != nil {
				if verbose {
					log.Println(suppress.Renderln("  *.wav と *.txt が揃ってないので無視します"))
				}
				delete(recentChanged, k)
				continue
			}
			s1Mod := s1.ModTime()
			s2Mod := s2.ModTime()
			if setting.Delta > 0 {
				if math.Abs(s1Mod.Sub(s2Mod).Seconds()) > setting.Delta {
					if verbose {
						log.Println(suppress.Renderln("  *.wav と *.txt の更新日時の差が", setting.Delta, "秒以上なので無視します"))
					}
					delete(recentChanged, k)
					continue
				}
			}
			if verbose {
				log.Println(suppress.Renderln("このファイルはルール検索対象です"))
			}
			files = append(files, file{k, s1Mod, tryCount})
		}
		if len(files) == 0 {
			continue
		}
		needRetry, err := processFiles(L, files, setting.Sort, recentChanged, recentSent)
		if err != nil {
			log.Println("ファイルの処理中にエラーが発生しました:", err)
		}
		if needRetry {
			go func() {
				time.Sleep(500 * time.Millisecond)
				notify <- map[string]struct{}{}
			}()
		}
	}
	return nil
}

func main() {
	var mono bool
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&mono, "m", false, "disable color")
	flag.Parse()

	if mono {
		warn = dummyColorizer{}
		caption = dummyColorizer{}
		suppress = dummyColorizer{}
	}
	log.Println(caption.Renderln("かんしくん"), version)
	if verbose {
		log.Println(warn.Renderln("冗長ログモードが有効"))
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

	log.Println(suppress.Renderln("  設定ファイル:"), settingFile)
	log.Println()

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

	recentChanged := map[string]int{}
	recentSent := map[string]time.Time{}
	for i := 0; ; i++ {
		err = process(watcher, settingWatcher, settingFile, recentChanged, recentSent, i)
		if err != nil {
			log.Println(err)
			log.Println("3秒後にリトライします")
			time.Sleep(3 * time.Second)
		}
	}
}
