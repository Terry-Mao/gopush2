package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	RetInternalErr    = 65535
	RetChannelExpired = 1
	RetOK             = 0
)

var (
	channel *Channel
)

func Publish(w http.ResponseWriter, r *http.Request) {
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
		if err = pubRetWrite(w, "read http body error", RetInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	// fetch subscriber from the channel
	sub := channel.GetSubscriber(key)
	if sub == nil {
		if err = pubRetWrite(w, "can't get a subscriber", RetInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	sub.AddMessage(string(body), expire)
	if err = pubRetWrite(w, "ok", RetOK); err != nil {
		Log.Printf("pubRetWrite() failed (%s)", err.Error())
	}

	return
}

func Subscribe(ws *websocket.Conn) {
	defer recoverFunc()

	params := ws.Request().URL.Query()
	subKey := params.Get("key")
	//TODO auth
	// get lastest message id

	midStr := params.Get("msg_id")
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		Log.Printf("argument error (%s)", err.Error())
		return
	}

	Log.Printf("client (%s) subscribe to key %s with msg_id = %s", ws.Request().RemoteAddr, subKey, midStr)
	// fetch subscriber from the channel
	sub := channel.GetSubscriber(subKey)
	if sub == nil {
		Log.Printf("can't get a subscriber from channel, key : %s", subKey)
		return
	}

	// add a conn to the subscriber
	if err = sub.AddConn(ws); err != nil {
		Log.Printf("sub.AddConn failed (%s)", err.Error())
		return
	}

	// remove exists conn
	defer sub.RemoveConn(ws)
	// send stored message
	if err = sub.SendStoredMessage(ws, mid); err != nil {
		Log.Printf("sub.SendStoredMessage() failed (%s)", err.Error())
		return
	}

	// blocking untill someone pub the key
	reply := ""
	if err = websocket.Message.Receive(ws, &reply); err != nil {
		Log.Printf("websocket.Message.Receive() failed (%s)", err.Error())
	}

	return
}

func pubRetWrite(w http.ResponseWriter, msg string, ret int) error {
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
