package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"
)

const (
	// internal failed
	retInternalErr = 65535
	// param error
	retParamErr = 65534
	// ok
	retOK = 0
	// create channel failed
	retCreateChannel = 1
	// add channel failed
	retAddChannle = 2
	// get channel failed
	retGetChannel = 3
	// add token failed
	retAddToken = 4
	// message push failed
	retPushMsg = 5
)

const (
	WebsocketProtocol = 0
	TCPProtocol       = 1
	heartbeatMsg      = "h"
	maxInt64          = int64(math.MaxInt64)
	oneSecond         = int64(time.Second)
)

var (
	// Exceed the max subscriber per key
	MaxConnErr = errors.New("Exceed the max subscriber connection per key")
	// Assection type failed
	AssertTypeErr = errors.New("Subscriber assert type failed")
	// Auth token failed
	AuthTokenErr = errors.New("Auth token failed")
	// Token exists
	TokenExistErr = errors.New("Token already exist")

	// heartbeat bytes
	heartbeatBytes = []byte(heartbeatMsg)
	// heartbeat len
	heartbeatByteLen = len(heartbeatMsg)
)

func StartAdminHttp() error {
	adminServeMux := http.NewServeMux()
	// publish
	adminServeMux.HandleFunc("/pub", PublishHandle)
	// stat
	adminServeMux.HandleFunc("/stat", StatHandle)
	// channel
	if Conf.Auth == 1 {
		adminServeMux.HandleFunc("/ch", ChannelHandle)
	}

	adminServeMux.HandleFunc("/debug/pprof/", pprof.Index)
	adminServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	adminServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	adminServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	err := http.ListenAndServe(Conf.AdminAddr, adminServeMux)
	if err != nil {
		LogError(LogLevelErr, "http.ListenAdServe(\"%s\") failed (%s)", Conf.AdminAddr, err.Error())
		return err
	}

	return nil
}

// http handler for create channel and add token
func ChannelHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	params := r.URL.Query()
	key := params.Get("key")
	token := params.Get("token")
	if key == "" || token == "" {
		if err := retWrite(w, "param error", retParamErr); err != nil {
			LogError(LogLevelErr, "retWrite failed (%s)", err.Error())
		}

		return
	}

	LogError(LogLevelInfo, "device:%s add channel token = %s", key, token)
	c, err := channel.New(key)
	if err != nil {
		LogError(LogLevelWarn, "device:%s can't create channle", key)
		if err = retWrite(w, "create channel failed", retCreateChannel); err != nil {
			LogError(LogLevelErr, "retWrite failed (%s)", err.Error())
		}

		return
	}

	if err = c.AddToken(token, key); err != nil {
		LogError(LogLevelWarn, "device:%s can't add token %s", key, token)
		if err = retWrite(w, "add token failed", retAddToken); err != nil {
			LogError(LogLevelErr, "retWrite failed (%s)", err.Error())
			return
		}
	}

	if err = retWrite(w, "ok", retOK); err != nil {
		LogError(LogLevelErr, "retWrite() failed (%s)", err.Error())
	}

	return
}

// PublishHandle is the web api for the publish message
func PublishHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	params := r.URL.Query()
	// get pub message key
	key := params.Get("key")
	// get the expired sec
	expireStr := params.Get("expire")
	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		// use default setting
		expire = Conf.MessageExpireSec * Second
	}

	expire = time.Now().UnixNano() + expire*Second
	// get message id
	midStr := params.Get("mid")
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		if err = retWrite(w, "param error", retParamErr); err != nil {
			LogError(LogLevelErr, "retWrite failed (%s)", err.Error())
		}

		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if err = retWrite(w, "read http body error", retInternalErr); err != nil {
			LogError(LogLevelErr, "pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	// fetch subscriber from the channel
	c, err := channel.Get(key)
	if err != nil {
		if err = retWrite(w, "can't get a subscriber", retGetChannel); err != nil {
			LogError(LogLevelErr, "pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	if err = c.PushMsg(&Message{Msg: string(body), Expire: expire, MsgID: mid}, key); err != nil {
		LogError(LogLevelWarn, "device:%s push message failed (%s)", key, err.Error())
		if err = retWrite(w, "push msg failed", retPushMsg); err != nil {
			LogError(LogLevelErr, "pubRetWrite() failed (%s)", err.Error())
		}
		return
	}

	if err = retWrite(w, "ok", retOK); err != nil {
		LogError(LogLevelErr, "pubRetWrite() failed (%s)", err.Error())
		return
	}
}

func retWrite(w http.ResponseWriter, msg string, ret int) error {
	res := map[string]interface{}{}
	res["msg"] = msg
	res["ret"] = ret

	strJson, err := json.Marshal(res)
	if err != nil {
		LogError(LogLevelErr, "json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(strJson)
	LogError(LogLevelInfo, "publish message:\"%s\" to client", respJson)
	if _, err := w.Write(strJson); err != nil {
		LogError(LogLevelErr, "w.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
}
