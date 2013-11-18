package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"sync/atomic"
	"time"
)

type ChannelStats struct {
	Created   int64 // channel created number
	Expired   int64 // channel expired number
	Refreshed int64 // channel refreshed number
}

type SubscriberStats struct {
	Created        int64 // subscriber created number
	AddedMessage   int64 // subscribers total added message number
	DeletedMessage int64 // subscribers total deleted message number
	ExpiredMessage int64 // subscribers total expired message number
	SentMessage    int64 // subscribers total sent message number
	TotalConn      int64 // subscribers total connected number
	CurConn        int64 // subscriber current connected number
}

var (
	// server
	startTime int64 // process start unixnano
	// channel stats
	channelStats = &ChannelStats{}
	// subscriber stats
	subscriberStats = &SubscriberStats{}
)

// increment channel created number
func (c *ChannelStats) IncrCreated() {
	atomic.AddInt64(&c.Created, 1)
}

// increment channel expired number
func (c *ChannelStats) IncrExpired() {
	atomic.AddInt64(&c.Expired, 1)

}

// increment channel refreshed number
func (c *ChannelStats) IncrRefreshed() {
	atomic.AddInt64(&c.Refreshed, 1)
}

// increment subscriber created number
func (s *SubscriberStats) IncrCreated() {
	atomic.AddInt64(&s.Created, 1)
}

// increment subscriber added message
func (s *SubscriberStats) IncrAddedMessage() {
	atomic.AddInt64(&s.AddedMessage, 1)
}

// increment subscriber deleted message
func (s *SubscriberStats) IncrDeletedMessage() {
	atomic.AddInt64(&s.DeletedMessage, 1)
}

// increment subscriber sent message
func (s *SubscriberStats) IncrSentMessage() {
	atomic.AddInt64(&s.SentMessage, 1)
}

// increment subscriber expired message
func (s *SubscriberStats) IncrExpiredMessage() {
	atomic.AddInt64(&s.ExpiredMessage, 1)
}

// increment subscriber conn
func (s *SubscriberStats) IncrConn() {
	atomic.AddInt64(&s.TotalConn, 1)
	atomic.AddInt64(&s.CurConn, 1)
}

// decrment subscriber conn
func (s *SubscriberStats) DecrConn() {
	atomic.AddInt64(&s.CurConn, -1)
}

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
		Log.Printf("user.Current() failed (%s)", err.Error())
		res["group"] = ""
		res["user"] = ""
	} else {
		res["group"] = usr.Gid
		res["user"] = usr.Uid
	}

	return jsonRes(res)
}

// channel stats
func (s *ChannelStats) Stats() []byte {
	res := map[string]interface{}{}
	res["refreshed"] = s.Refreshed
	res["created"] = s.Created
	res["expired"] = s.Expired

	return jsonRes(res)
}

// subscriber stats
func (s *SubscriberStats) Stats() []byte {
	res := map[string]interface{}{}
	res["created"] = s.Created
	res["added_message"] = s.AddedMessage
	res["deleted_message"] = s.DeletedMessage
	res["expired_message"] = s.ExpiredMessage
	res["sent_message"] = s.SentMessage
	res["total_conn"] = s.TotalConn
	res["cur_conn"] = s.CurConn

	return jsonRes(res)
}

// configuration info
func ConfigInfo() []byte {
	strJson, err := json.Marshal(Conf)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", Conf)
		return []byte{}
	}

	return strJson
}

func jsonRes(res map[string]interface{}) []byte {
	strJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return []byte{}
	}

	return strJson
}

func Stat(w http.ResponseWriter, r *http.Request) {
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
	case "subscriber":
		res = subscriberStats.Stats()
	case "channel":
		res = channelStats.Stats()
	case "confit":
		res = ConfigInfo()
	}

	if _, err := w.Write(res); err != nil {
		Log.Printf("w.Write(\"%s\") failed (%s)", string(res), err.Error())
	}

	return
}
