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

func watch(watcher *fsnotify.Watcher, settingFile string, recentChanged map[string]int, recentSent map[string]time.Time, timer *time.Timer) error {
	setting, err := newSetting(settingFile)
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
	L.SetGlobal("readproject", L.NewFunction(luaReadProject))
	L.SetGlobal("sendfile", L.NewFunction(luaSendFile))
	L.SetGlobal("findrule", L.NewFunction(luaFindRule(setting)))
	L.SetGlobal("getaudioinfo", L.NewFunction(luaGetAudioInfo))
	L.SetGlobal("tosjis", L.NewFunction(luaToSJIS))
	L.SetGlobal("toexostring", L.NewFunction(luaToEXOString))

	if err := L.DoFile("_entrypoint.lua"); err != nil {
		return errors.Wrap(err, "_entrypoint.lua の実行中にエラーが発生しました")
	}

	log.Println("監視を開始します:")
	for _, dir := range setting.Dirs() {
		log.Println("  " + dir)
		err = watcher.Add(dir)
		if err != nil {
			return errors.Wrap(err, "監視フォルダーの登録中にエラーが発生しました")
		}
		defer watcher.Remove(dir)
	}
	log.Println("  delta:", setting.Delta)
	log.Println("  freshness:", setting.Freshness)

	var reload bool
	for {
		select {
		case event := <-watcher.Events:
			if verbose {
				log.Println("[INFO]", "イベント検出", event)
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
					log.Println("[INFO]", k)
				}
				if _, found := recentSent[k]; found {
					if verbose {
						log.Println("[INFO]", "  つい最近送ったファイルなので、重複送信回避のために無視します")
					}
					continue
				}
				s1, e1 := os.Stat(k)
				s2, e2 := os.Stat(k[:len(k)-4] + ".txt")
				if e1 != nil || e2 != nil {
					if verbose {
						log.Println("[INFO]", "  *.wav と *.txt が揃ってないので無視します")
					}
					continue
				}
				s1Mod := s1.ModTime()
				s2Mod := s2.ModTime()
				if setting.Delta > 0 {
					if math.Abs(s1Mod.Sub(s2Mod).Seconds()) > setting.Delta {
						if verbose {
							log.Println("[INFO]", "  *.wav と *.txt の更新日時の差が", setting.Delta, "秒以上なので無視します")
						}
						continue
					}
				}
				files = append(files, file{k, s1Mod, tryCount})
			}
			if len(files) == 0 {
				continue
			}
			sort.Slice(files, func(i, j int) bool { return files[i].ModDate.Before(files[j].ModDate) })
			t := L.NewTable()
			tc := L.NewTable()
			for _, f := range files {
				t.Append(lua.LString(f.Filepath))
				tc.Append(lua.LNumber(f.TryCount))
			}
			if err := L.CallByParam(lua.P{
				Fn:      L.GetGlobal("changed"),
				NRet:    1,
				Protect: true,
			}, t, tc); err != nil {
				log.Println("ファイルの処理中にエラーが発生しました:", err)
			}
			rv := L.ToTable(-1)
			if rv != nil {
				// remove processed entries
				n := rv.MaxN()
				for i := 1; i <= n; i++ {
					k := rv.RawGetInt(i).String()
					recentSent[k] = now
					delete(recentChanged, k)
				}
				// increment retry count
				var found bool
				for k := range recentChanged {
					ct := recentChanged[k]
					if ct == 9 {
						log.Println("たくさん失敗したのでこのファイルは諦めます:", k)
						delete(recentChanged, k)
						continue
					}
					recentChanged[k] = ct + 1
					found = true
				}
				if found {
					timer.Reset(500 * time.Millisecond)
				}
			} else {
				log.Println("処理後の戻り値が異常です")
			}
			L.Pop(1)
		}
	}
}

func main() {
	log.Println("かんしくん", version)

	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()

	settingFile := flag.Arg(0)
	if settingFile == "" {
		settingFile = "setting.txt"
	}

	if !filepath.IsAbs(settingFile) {
		p, err := filepath.Abs(settingFile)
		if err != nil {
			log.Fatalln("filepath.Abs に失敗しました:", err)
		}
		settingFile = p
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
			log.Println("監視処理でエラーが発生しました:", err)
			log.Println("3秒後にリトライします")
			time.Sleep(3 * time.Second)
		}
	}
}
