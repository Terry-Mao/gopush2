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
	retInternalErr = 65535
	retOK          = 0
)

func StartHttp() error {
	// set sub handler
	http.Handle("/sub", websocket.Handler(Subscribe))
	if Conf.Debug == 1 {
		http.HandleFunc("/client", Client)
	}

	// admin
	if Conf.AdminAddr != Conf.Addr || Conf.AdminPort != Conf.Port {
		go func() {
			adminServeMux := http.NewServeMux()
			// publish
			adminServeMux.HandleFunc("/pub", Publish)
			// stat
			adminServeMux.HandleFunc("/stat", Stat)
			err := http.ListenAndServe(fmt.Sprintf("%s:%d", Conf.AdminAddr, Conf.AdminPort), adminServeMux)
			if err != nil {
				panic(err)
			}
		}()
	} else {
		http.HandleFunc("/pub", Publish)
		http.HandleFunc("/stat", Stat)
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
		if err = pubRetWrite(w, "read http body error", retInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	// fetch subscriber from the channel
	sub := channel.Subscriber(key)
	if sub == nil {
		if err = pubRetWrite(w, "can't get a subscriber", retInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	sub.PublishMessage(string(body), expire)
	if err = pubRetWrite(w, "ok", retOK); err != nil {
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
	sub := channel.Subscriber(subKey)
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
