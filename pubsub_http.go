package main

import (
	"code.google.com/p/go.net/websocket"
	"net"
	"net/http"
	"strconv"
	"time"
)

type KeepAliveListener struct {
	net.Listener
}

func (l *KeepAliveListener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		Log.Printf("Listener.Accept() failed (%s)", err.Error())
		return
	}

	// set keepalive
	if tc, ok := c.(*net.TCPConn); !ok {
		panic("Assection type failed c.(net.TCPConn)")
	} else {
		err = tc.SetKeepAlive(true)
		if err != nil {
			Log.Printf("tc.SetKeepAlive(true) failed (%s)", err.Error())
			return
		}
	}

	return
}

func StartHttp() error {
	// set sub handler
	http.Handle("/sub", websocket.Handler(SubscribeHandle))
	if Conf.Debug == 1 {
		http.HandleFunc("/client", Client)
	}

	if Conf.TCPKeepAlive == 1 {
		server := &http.Server{}
		l, err := net.Listen("tcp", Conf.Addr)
		if err != nil {
			Log.Printf("net.Listen(\"tcp\", \"%s\") failed (%s)", Conf.Addr, err.Error())
			return err
		}

		return server.Serve(&KeepAliveListener{Listener: l})
	} else {
		if err := http.ListenAndServe(Conf.Addr, nil); err != nil {
			Log.Printf("http.ListenAdServe(\"%s\") failed (%s)", Conf.Addr, err.Error())
			return err
		}
	}

	// nerve here
	return nil
}

// Subscriber Handle is the websocket handle for sub request
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
		i, err := strconv.Atoi(heartbeatStr)
		if err != nil {
			Log.Printf("heartbeat argument error (%s)", err.Error())
			return
		}

		heartbeat = i
	}

	heartbeat *= 2
	if heartbeat <= 0 {
		Log.Printf("heartbeat argument error, less than 0")
		return
	}

	// get auth token
	token := params.Get("token")
	Log.Printf("client %s subscribe to key = %s, mid = %d, token = %s, heartbeat = %d", ws.Request().RemoteAddr, key, mid, token, heartbeat)
	// fetch subscriber from the channel
	c, err := channel.Get(key)
	if err != nil {
		if Conf.Auth == 0 {
			c, err = channel.New(key)
			if err != nil {
				Log.Printf("device %s: can't create channle", key)
				return
			}
		} else {
			Log.Printf("device %s: can't get a channel (%s)", key, err.Error())
			return
		}
	}

	// auth
	if Conf.Auth == 1 {
		if err = c.AuthToken(token, key); err != nil {
			Log.Printf("device %s: auth token failed \"%s\" (%s)", key, token, err.Error())
			return
		}
	}

	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err = ws.Write(heartbeatBytes); err != nil {
		Log.Printf("device %s: write first heartbeat to client failed (%s)", key, err.Error())
		return
	}

	// send stored message, and use the last message id if sent any
	if err = c.SendMsg(ws, mid, key); err != nil {
		Log.Printf("device %s: send offline message failed (%s)", key, err.Error())
		return
	}

	// add a conn to the channel
	if err = c.AddConn(ws, mid, key); err != nil {
		Log.Printf("device %s: add conn failed (%s)", key, err.Error())
		return
	}

	// blocking wait client heartbeat
	reply := ""
	for {
		ws.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat)))
		if err = websocket.Message.Receive(ws, &reply); err != nil {
			Log.Printf("websocket.Message.Receive() failed (%s)", err.Error())
			break
		}

		if reply == heartbeatMsg {
			if _, err = ws.Write(heartbeatBytes); err != nil {
				Log.Printf("device %s: write heartbeat to client failed (%s)", key, err.Error())
				break
			}

			Log.Printf("device %s: receive heartbeat", key)
		} else {
			Log.Printf("device %s: unknown heartbeat protocol", key)
			break
		}
	}

	// remove exists conn
	if err := c.RemoveConn(ws, mid, key); err != nil {
		Log.Printf("device %s: remove conn failed (%s)", key, err.Error())
	}

	return
}
