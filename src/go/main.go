package main

import (
	"flag"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	lua "github.com/yuin/gopher-lua"
)

func main() {
	log.Println("かんしくん", version)

	flag.Parse()

	settingFile := flag.Arg(0)
	if settingFile == "" {
		settingFile = "setting.txt"
	}
	setting, err := newSetting(settingFile)
	if err != nil {
		log.Fatalln(settingFile+" の読み込みに失敗しました:", err)
	}

	L := lua.NewState()
	defer L.Close()

	L.SetGlobal("debug_print", L.NewFunction(luaDebugPrint))
	L.SetGlobal("readproject", L.NewFunction(luaReadProject))
	L.SetGlobal("sendfile", L.NewFunction(luaSendFile))
	L.SetGlobal("findrule", L.NewFunction(luaFindRule(setting)))
	L.SetGlobal("getaudioinfo", L.NewFunction(luaGetAudioInfo))
	L.SetGlobal("tosjis", L.NewFunction(luaToSJIS))
	L.SetGlobal("fromsjis", L.NewFunction(luaFromSJIS))
	L.SetGlobal("toexostring", L.NewFunction(luaToEXOString))

	if err := L.DoFile("_entrypoint.lua"); err != nil {
		log.Fatalln("_entrypoint.lua の実行中にエラーが発生しました:", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("監視処理の準備中にエラーが発生しました:", err)
	}
	defer watcher.Close()

	err = watcher.Add(setting.Dir)
	if err != nil {
		log.Fatalln("監視フォルダーの登録中にエラーが発生しました:", err)
	}

	log.Println("監視を開始しました:", setting.Dir)

	recentChanged := map[string]struct{}{}
	recentSent := map[string]time.Time{}
	timer := time.NewTimer(100 * time.Millisecond)
	timer.Stop()
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			ext := strings.ToLower(filepath.Ext(event.Name))
			if ext == ".wav" || ext == ".txt" {
				recentChanged[event.Name[:len(event.Name)-len(ext)]+".wav"] = struct{}{}
				timer.Reset(100 * time.Millisecond)
			}
		case err := <-watcher.Errors:
			log.Println("エラーが発生しました:", err)
		case <-timer.C:
			n := time.Now()
			for k := range recentSent {
				if n.Sub(recentSent[k]) > 3*time.Second {
					delete(recentSent, k)
				}
			}
			t := L.NewTable()
			for k := range recentChanged {
				s1, e1 := os.Stat(k)
				s2, e2 := os.Stat(k[:len(k)-4] + ".txt")
				if e1 != nil || e2 != nil {
					continue
				}
				if math.Abs(s1.ModTime().Sub(s2.ModTime()).Seconds()) > setting.Delta {
					continue
				}
				if _, found := recentSent[k]; found {
					continue
				}
				t.Append(lua.LString(k))
				recentSent[k] = n
			}
			if t.Len() == 0 {
				continue
			}
			recentChanged = map[string]struct{}{}
			if err := L.CallByParam(lua.P{
				Fn:      L.GetGlobal("changed"),
				NRet:    0,
				Protect: true,
			}, t); err != nil {
				log.Fatal(err)
			}
		}
	}
}
