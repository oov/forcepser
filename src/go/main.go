package main

import (
	"flag"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/yuin/gluare"
	lua "github.com/yuin/gopher-lua"
)

var verbose bool

type file struct {
	Filepath string
	ModDate  time.Time
	TryCount int
}

func processFiles(L *lua.LState, files []file, recentChanged map[string]int, recentSent map[string]time.Time) (needRetry bool, err error) {
	defer func() {
		for k, ct := range recentChanged {
			if ct == 9 {
				log.Println("  たくさん失敗したのでこのファイルは諦めます:", k)
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
			log.Println("[INFO] プロジェクト情報取得失敗:", err)
		}
		err = errors.Errorf("ごちゃまぜドロップス v0.3 以降がインストールされた AviUtl が検出できませんでした")
		return
	}
	if proj.Width == 0 {
		err = errors.Errorf("AviUtl で編集中のプロジェクトが見つかりません")
		return
	}
	if verbose {
		log.Println("[INFO] プロジェクト情報:")
		if proj.GCMZAPIVer >= 1 {
			log.Println("[INFO]   ProjectFile:", proj.ProjectFile)
		}
		log.Println("[INFO]   Window:", int(proj.Window))
		log.Println("[INFO]   Width:", proj.Width)
		log.Println("[INFO]   Height:", proj.Height)
		log.Println("[INFO]   VideoRate:", proj.VideoRate)
		log.Println("[INFO]   VideoScale:", proj.VideoScale)
		log.Println("[INFO]   AudioRate:", proj.AudioRate)
		log.Println("[INFO]   AudioCh:", proj.AudioCh)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModDate.Before(files[j].ModDate)
	})
	t := L.NewTable()
	tc := L.NewTable()
	for _, f := range files {
		t.Append(lua.LString(f.Filepath))
		tc.Append(lua.LNumber(f.TryCount))
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
	}, t, tc, pt); err != nil {
		return
	}
	rv := L.ToTable(-1)
	if rv == nil {
		err = errors.Errorf("処理後の戻り値が異常です")
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

func watch(watcher *fsnotify.Watcher, settingFile string, recentChanged map[string]int, recentSent map[string]time.Time, timer *time.Timer) error {
	exePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "exe ファイルのパスが取得できません")
	}
	tempDir := filepath.Join(filepath.Dir(exePath), "tmp")

	setting, err := newSetting(settingFile, tempDir)
	if err != nil {
		return errors.Wrap(err, "設定の読み込みに失敗しました")
	}

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("re", gluare.Loader)
	err = L.DoString(`re = require("re")`)
	if err != nil {
		return errors.Wrap(err, "Lua スクリプト環境の初期化中にエラーが発生しました")
	}

	L.SetGlobal("debug_print", L.NewFunction(luaDebugPrint))
	L.SetGlobal("debug_print_verbose", L.NewFunction(luaDebugPrintVerbose))
	L.SetGlobal("sendfile", L.NewFunction(luaSendFile))
	L.SetGlobal("findrule", L.NewFunction(luaFindRule(setting)))
	L.SetGlobal("getaudioinfo", L.NewFunction(luaGetAudioInfo))
	L.SetGlobal("tosjis", L.NewFunction(luaToSJIS))
	L.SetGlobal("fromsjis", L.NewFunction(luaFromSJIS))
	L.SetGlobal("toexostring", L.NewFunction(luaToEXOString))
	L.SetGlobal("tofilename", L.NewFunction(luaToFilename))

	if err := L.DoFile("_entrypoint.lua"); err != nil {
		return errors.Wrap(err, "_entrypoint.lua の実行中にエラーが発生しました")
	}

	for i, a := range setting.Asas {
		log.Printf("  Asas %d:", i+1)
		log.Println("    対象EXE:", a.Exe)
		log.Println("    フィルター:", a.Filter)
		log.Println("    保存先フォルダー:", a.Folder)
		log.Println("    フォーマット:", a.Format)
		log.Println("    フラグ:", a.Flags)
		if _, err := a.ConfirmAndRun(); err != nil {
			return errors.Wrap(err, "プログラムの起動に失敗しました")
		}
	}
	for i, r := range setting.Rule {
		log.Printf("  ルール%d:", i+1)
		log.Println("    対象フォルダー:", r.Dir)
		log.Println("    対象ファイル名:", r.File)
		log.Println("    テキストファイルの文字コード:", r.Encoding)
		if r.textRE != nil {
			log.Println("    テキスト判定用の正規表現:", r.Text)
		}
		log.Println("    挿入先レイヤー:", r.Layer)
		if r.Modifier != "" {
			log.Println("    挿入前のテキスト加工: あり")
		} else {
			log.Println("    挿入前のテキスト加工: なし")
		}
		log.Println("    ユーザーデータ:", r.UserData)
		log.Println("    パディング:", r.Padding)
	}

	log.Println("  filemove:", setting.FileMove)
	log.Println("  deletetext:", setting.DeleteText)
	log.Println("  delta:", setting.Delta)
	log.Println("  freshness:", setting.Freshness)
	log.Println("  exofile:", setting.ExoFile)
	log.Println("  luafile:", setting.LuaFile)
	log.Println("  padding:", setting.Padding)

	if err = os.Mkdir(tempDir, 0777); err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "tmp フォルダの作成に失敗しました")
	}

	log.Println("監視を開始します:")
	for _, dir := range setting.Dirs() {
		err = watcher.Add(dir)
		if err != nil {
			if _, err2 := os.Stat(dir); os.IsNotExist(err2) {
				return errors.Errorf("フォルダーが見つかりません: %v", dir)
			}
			return errors.Wrapf(err, "フォルダーが監視できません: %v", dir)
		}
		log.Println("  " + dir)
		defer watcher.Remove(dir)
	}

	var reload bool
	for {
		select {
		case event := <-watcher.Events:
			if verbose {
				log.Println("[INFO]", "イベント検証:", event)
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				if verbose {
					log.Println("[INFO]", "  オペレーションが Create / Write ではないので何もしません")
				}
				continue
			}
			if event.Name == settingFile {
				if verbose {
					log.Println("[INFO]", "  設定ファイルの再読み込みとして処理します")
				}
				reload = true
				timer.Reset(100 * time.Millisecond)
				continue
			}
			ext := strings.ToLower(filepath.Ext(event.Name))
			if ext != ".wav" && ext != ".txt" {
				if verbose {
					log.Println("[INFO]", "  *.wav / *.txt のどちらでもないので何もしません")
				}
				continue
			}
			if setting.Freshness > 0 {
				st, err := os.Stat(event.Name)
				if err != nil {
					if verbose {
						log.Println("[INFO]", "  更新日時の取得に失敗したので何もしません")
					}
					continue
				}
				if math.Abs(time.Now().Sub(st.ModTime()).Seconds()) > setting.Freshness {
					if verbose {
						log.Println("[INFO]", "  更新日時が", setting.Freshness, "秒以上前なので何もしません")
					}
					continue
				}
			}
			if verbose {
				log.Println("[INFO]", "  送信ファイル候補にします")
			}
			recentChanged[event.Name[:len(event.Name)-len(ext)]+".wav"] = 0
			timer.Reset(100 * time.Millisecond)
		case err := <-watcher.Errors:
			log.Println("監視中にエラーが発生しました:", err)
		case <-timer.C:
			if reload {
				log.Println()
				log.Println("設定ファイルを再読み込みします")
				log.Println()
				return nil
			}

			now := time.Now()
			for k := range recentSent {
				if now.Sub(recentSent[k]) > 3*time.Second {
					delete(recentSent, k)
				}
			}
			var files []file
			for k, tryCount := range recentChanged {
				if verbose {
					log.Println("[INFO]", "送信ファイル候補検証:", k)
				}
				if _, found := recentSent[k]; found {
					if verbose {
						log.Println("[INFO]", "  つい最近送ったファイルなので、重複送信回避のために無視します")
					}
					delete(recentChanged, k)
					continue
				}
				s1, e1 := os.Stat(k)
				s2, e2 := os.Stat(k[:len(k)-4] + ".txt")
				if e1 != nil || e2 != nil {
					if verbose {
						log.Println("[INFO]", "  *.wav と *.txt が揃ってないので無視します")
					}
					delete(recentChanged, k)
					continue
				}
				s1Mod := s1.ModTime()
				s2Mod := s2.ModTime()
				if setting.Delta > 0 {
					if math.Abs(s1Mod.Sub(s2Mod).Seconds()) > setting.Delta {
						if verbose {
							log.Println("[INFO]", "  *.wav と *.txt の更新日時の差が", setting.Delta, "秒以上なので無視します")
						}
						delete(recentChanged, k)
						continue
					}
				}
				if verbose {
					log.Println("[INFO]", "  このファイルはルール検索対象です")
				}
				files = append(files, file{k, s1Mod, tryCount})
			}
			if len(files) == 0 {
				continue
			}
			L.SetGlobal("exofile", lua.LString(setting.ExoFile))
			L.SetGlobal("luafile", lua.LString(setting.LuaFile))
			needRetry, err := processFiles(L, files, recentChanged, recentSent)
			if err != nil {
				log.Println("ファイルの処理中にエラーが発生しました:", err)
			}
			if needRetry {
				timer.Reset(500 * time.Millisecond)
			}
		}
	}
}

func main() {
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()
	log.Println("かんしくん", version)
	if verbose {
		log.Println("  [INFO] 冗長ログモードが有効です")
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

	log.Println("設定ファイル:")
	log.Println("  " + settingFile)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("fsnotify.NewWatcher に失敗しました:", err)
	}
	defer watcher.Close()

	err = watcher.Add(settingFile)
	if err != nil {
		log.Fatalln("設定ファイルの監視に失敗しました:", err)
	}

	recentChanged := map[string]int{}
	recentSent := map[string]time.Time{}
	timer := time.NewTimer(100 * time.Millisecond)
	timer.Stop()
	for {
		err = watch(watcher, settingFile, recentChanged, recentSent, timer)
		if err != nil {
			log.Println(err)
			log.Println("3秒後にリトライします")
			time.Sleep(3 * time.Second)
		}
	}
}
