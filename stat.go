package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/user"
	"runtime"
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
	c.Created++
}

// increment channel expired number
func (c *ChannelStats) IncrExpired() {
	c.Expired++

}

// increment channel refreshed number
func (c *ChannelStats) IncrRefreshed() {
	c.Refreshed++
}

// increment subscriber created number
func (s *SubscriberStats) IncrCreated() {
	s.Created++
}

// increment subscriber added message
func (s *SubscriberStats) IncrAddedMessage() {
	s.AddedMessage++
}

// increment subscriber deleted message
func (s *SubscriberStats) IncrDeletedMessage() {
	s.DeletedMessage++
}

// increment subscriber sent message
func (s *SubscriberStats) IncrSentMessage() {
	s.SentMessage++
}

// increment subscriber expired message
func (s *SubscriberStats) IncrExpiredMessage() {
	s.ExpiredMessage++
}

// increment subscriber conn
func (s *SubscriberStats) IncrConn() {
	s.TotalConn++
	s.CurConn++
}

// decrment subscriber conn
func (s *SubscriberStats) DecrConn() {
	s.CurConn--
}

// start stats, called at process start
func StartStats() {
	startTime = time.Now().UnixNano()
}

// memory stats
func GetMemStats() []byte {
	//TODO
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)

	res := map[string]interface{}{}
	res["todo"] = 1

	return jsonRes(res)
}

// golang stats
func GetGoStats() []byte {
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
func GetServerStats() []byte {
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
func GetChannelStats() []byte {
	res := map[string]interface{}{}
	res["cur"] = channel.NumDID()
	res["refreshed"] = channelStats.Refreshed
	res["created"] = channelStats.Created
	res["expired"] = channelStats.Expired

	return jsonRes(res)
}

// subscriber stats
func GetSubscriberStats() []byte {
	res := map[string]interface{}{}
	res["created"] = subscriberStats.Created
	res["added_message"] = subscriberStats.AddedMessage
	res["deleted_message"] = subscriberStats.DeletedMessage
	res["expired_message"] = subscriberStats.ExpiredMessage
	res["sent_message"] = subscriberStats.SentMessage
	res["total_conn"] = subscriberStats.TotalConn
	res["cur_conn"] = subscriberStats.CurConn

	return jsonRes(res)
}

// configuration info
func GetConfigInfo() []byte {
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
		res = GetMemStats()
	case "server":
		res = GetServerStats()
	case "golang":
		res = GetGoStats()
	case "subscriber":
		res = GetSubscriberStats()
	case "channel":
		res = GetChannelStats()
	case "confit":
		res = GetConfigInfo()
	}

	if _, err := w.Write(res); err != nil {
		Log.Printf("w.Write(\"%s\") failed (%s)", string(res), err.Error())
	}

	return
}
