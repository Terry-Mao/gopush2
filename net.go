package main

import (
	"net"
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
