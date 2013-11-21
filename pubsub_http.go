package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	// internal failed
	retInternalErr = 65535
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
)

func StartHttp() error {
	// set sub handler
	http.Handle("/sub", websocket.Handler(SubscribeHandle))
	http.HandleFunc("/ch", ChannelHandle)
	if Conf.Debug == 1 {
		http.HandleFunc("/client", Client)
	}

	// admin
	if Conf.AdminAddr != Conf.Addr || Conf.AdminPort != Conf.Port {
		go func() {
			adminServeMux := http.NewServeMux()
			// publish
			adminServeMux.HandleFunc("/pub", PublishHandle)
			// stat
			adminServeMux.HandleFunc("/stat", StatHandle)
			err := http.ListenAndServe(fmt.Sprintf("%s:%d", Conf.AdminAddr, Conf.AdminPort), adminServeMux)
			if err != nil {
				panic(err)
			}
		}()
	} else {
		http.HandleFunc("/pub", PublishHandle)
		http.HandleFunc("/stat", StatHandle)
	}

	a := fmt.Sprintf("%s:%d", Conf.Addr, Conf.Port)
	if Conf.TCPKeepAlive == 1 {
		server := &http.Server{}
		l, err := net.Listen("tcp", a)
		if err != nil {
			Log.Printf("net.Listen(\"tcp\", \"%s\") failed (%s)", a, err.Error())
			return err
		}

		return server.Serve(&KeepAliveListener{Listener: l})
	} else {
		if err := http.ListenAndServe(a, nil); err != nil {
			Log.Printf("http.ListenAdServe(\"%s\") failed (%s)", a, err.Error())
			return err
		}
	}

	// nerve here
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
	// TODO bussiness logical

	Log.Printf("device %s: add channel, token = %s", key, token)
	c, err := channel.New(key)
	if err != nil {
		Log.Printf("device %s: can't create channle", key)
		if err = retWrite(w, "create channel failed", retCreateChannel); err != nil {
			Log.Printf("retWrite failed (%s)", err.Error())
		}

		return
	}

	if err = c.AddToken(token); err != nil {
		Log.Printf("device %s: can't add token %s", key, token)
		if err = retWrite(w, "add token failed", retAddToken); err != nil {
			Log.Printf("retWrite failed (%s)", err.Error())
			return
		}
	}

	if err = retWrite(w, "ok", retOK); err != nil {
		Log.Printf("retWrite() failed (%s)", err.Error())
	}

	return
}

func PublishHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	params := r.URL.Query()
	key := params.Get("key")
	//TODO auth
	// get the expired sec
	expireStr := params.Get("expire")
	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		// use default setting
		expire = Conf.MessageExpireSec * Second
	}

	expire = time.Now().UnixNano() + expire*Second
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if err = retWrite(w, "read http body error", retInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	// fetch subscriber from the channel
	c, err := channel.Get(key)
	if err != nil {
		if err = retWrite(w, "can't get a subscriber", retGetChannel); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	if err = c.PushMsg(string(body), expire, key); err != nil {
		Log.Printf("device %s: push message failed (%s)", key, err.Error())
		return
	}

	if err = retWrite(w, "ok", retOK); err != nil {
		Log.Printf("pubRetWrite() failed (%s)", err.Error())
		return
	}
}

func SubscribeHandle(ws *websocket.Conn) {
	defer recoverFunc()

	params := ws.Request().URL.Query()
	// get subscriber key
	key := params.Get("key")
	// get lastest message id
	midStr := params.Get("mid")
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		Log.Printf("mid argument error (%s)", err.Error())
		return
	}

	// get heartbeat second
	heartbeat := Conf.HeartbeatSec
	heartbeatStr := params.Get("heartbeat")
	if heartbeatStr != "" {
		i, err := strconv.ParseInt(heartbeatStr, 10, 32)
		if err != nil {
			Log.Printf("heartbeat argument error (%s)", err.Error())
			return
		}

		heartbeat = int(i)
	}

	heartbeat *= 2
	if heartbeat <= 0 {
		Log.Printf("heartbeat argument error, less than 0")
		return
	}

	// get auth token
	token := params.Get("token")
	Log.Printf("client %s subscribe to key = %s, mid = %s, token = %s, heartbeat = %d", ws.Request().RemoteAddr, key, midStr, token, heartbeat)
	// fetch subscriber from the channel
	c, err := channel.Get(key)
	if err != nil {
		Log.Printf("device %s: can't get a channel (%s)", key, err.Error())
		return
	}

	// auth
	if err = c.AuthToken(token); err != nil {
		Log.Printf("device %s: auth token failed \"%s\" (%s)", key, token, err.Error())
		return
	}

	// add a conn to the channel
	if err = c.AddConn(ws, mid, key); err != nil {
		Log.Printf("device %s: add conn failed (%s)", key, err.Error())
		return
	}

	// remove exists conn
	defer func() {
		if err := c.RemoveConn(ws, mid, key); err != nil {
			Log.Printf("device %s: remove conn failed (%s)", key, err.Error())
		}
	}()

	// send stored message
	if err = c.SendMsg(ws, mid, key); err != nil {
		Log.Printf("device %s: send offline message failed (%s)", key, err.Error())
		return
	}

	// blocking wait client heartbeat
	reply := ""
	for {
		ws.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat)))
		if err = websocket.Message.Receive(ws, &reply); err != nil {
			Log.Printf("websocket.Message.Receive() failed (%s)", err.Error())
			return
		}

		if reply == "" {
			if _, err = ws.Write([]byte("")); err != nil {
				Log.Printf("device %s: write heartbeat to client failed (%s)", key, err.Error())
				return
			}

			Log.Printf("device %s: receive heartbeat", key)
		} else {
			Log.Printf("device %s: unknown heartbeat protocol", key)
			return
		}
	}
}

func retWrite(w http.ResponseWriter, msg string, ret int) error {
	res := map[string]interface{}{}
	res["msg"] = msg
	res["ret"] = ret

	strJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(strJson)
	Log.Printf("pub send to client: %s", respJson)
	if _, err := w.Write(strJson); err != nil {
		Log.Printf("w.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
}

func recoverFunc() {
	if err := recover(); err != nil {
		Log.Printf("Error : %v, Debug : \n%s", err, string(debug.Stack()))
	}
}
