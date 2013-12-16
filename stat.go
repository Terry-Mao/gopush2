package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"
)

var (
	// server
	startTime int64 // process start unixnano
)

// start stats, called at process start
func StartStats() {
	startTime = time.Now().UnixNano()
}

// memory stats
func MemStats() []byte {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	// general
	res := map[string]interface{}{}
	res["alloc"] = m.Alloc
	res["total_alloc"] = m.TotalAlloc
	res["sys"] = m.Sys
	res["lookups"] = m.Lookups
	res["mallocs"] = m.Mallocs
	res["frees"] = m.Frees
	// heap
	res["heap_alloc"] = m.HeapAlloc
	res["heap_sys"] = m.HeapSys
	res["heap_idle"] = m.HeapIdle
	res["heap_inuse"] = m.HeapInuse
	res["heap_released"] = m.HeapReleased
	res["heap_objects"] = m.HeapObjects
	// low-level fixed-size struct alloctor
	res["stack_inuse"] = m.StackInuse
	res["stack_sys"] = m.StackSys
	res["mspan_inuse"] = m.MSpanInuse
	res["mspan_sys"] = m.MSpanSys
	res["mcache_inuse"] = m.MCacheInuse
	res["mcache_sys"] = m.MCacheSys
	res["buckhash_sys"] = m.BuckHashSys
	// GC
	res["next_gc"] = m.NextGC
	res["last_gc"] = m.LastGC
	res["pause_total_ns"] = m.PauseTotalNs
	res["pause_ns"] = m.PauseNs
	res["num_gc"] = m.NumGC
	res["enable_gc"] = m.EnableGC
	res["debug_gc"] = m.DebugGC
	res["by_size"] = m.BySize

	return jsonRes(res)
}

// golang stats
func GoStats() []byte {
	res := map[string]interface{}{}
	res["compiler"] = runtime.Compiler
	res["arch"] = runtime.GOARCH
	res["os"] = runtime.GOOS
	res["max_procs"] = runtime.GOMAXPROCS(-1)
	res["root"] = runtime.GOROOT()
	res["cgo_call"] = runtime.NumCgoCall()
	res["goroutine_num"] = runtime.NumGoroutine()
	res["version"] = runtime.Version()

	return jsonRes(res)
}

// server stats
func ServerStats() []byte {
	res := map[string]interface{}{}
	res["uptime"] = time.Now().UnixNano() - startTime
	hostname, _ := os.Hostname()
	res["hostname"] = hostname
	wd, _ := os.Getwd()
	res["wd"] = wd
	res["ppid"] = os.Getppid()
	res["pid"] = os.Getpid()
	res["pagesize"] = os.Getpagesize()
	if usr, err := user.Current(); err != nil {
		LogError(LogLevelErr, "user.Current() failed (%s)", err.Error())
		res["group"] = ""
		res["user"] = ""
	} else {
		res["group"] = usr.Gid
		res["user"] = usr.Uid
	}

	return jsonRes(res)
}

// configuration info
func ConfigInfo() []byte {
	strJson, err := json.Marshal(Conf)
	if err != nil {
		LogError(LogLevelErr, "json.Marshal(\"%v\") failed", Conf)
		return []byte{}
	}

	return strJson
}

func jsonRes(res map[string]interface{}) []byte {
	strJson, err := json.Marshal(res)
	if err != nil {
		LogError(LogLevelErr, "json.Marshal(\"%v\") failed", res)
		return []byte{}
	}

	return strJson
}

func StatHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	params := r.URL.Query()
	types := params.Get("type")

	res := []byte{}
	switch types {
	case "memory":
		res = MemStats()
	case "server":
		res = ServerStats()
	case "golang":
		res = GoStats()
	case "confit":
		res = ConfigInfo()
	}

	if _, err := w.Write(res); err != nil {
		LogError(LogLevelErr, "w.Write(\"%s\") failed (%s)", string(res), err.Error())
	}

	return
}
