package main

import (
	"net"
	"time"
)

func main() {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		panic(err)
	}

	if _, err := conn.Write([]byte("*4\r\n$3\r\nsub\r\n$9\r\nTerry-Mao\r\n$1\r\n0\r\n$2\r\n30\r\n")); err != nil {
		panic(err)
	}

	go func() {
		for {
			time.Sleep(30 * time.Second)
			if _, err := conn.Write([]byte("")); err != nil {
				panic(err)
			}
		}
	}()

	buf := make([]byte, 1024)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(30))); err != nil {
			panic(err)
		}

		if _, err := conn.Read(buf); err != nil {
			panic(err)
		}

		bufStr := string(buf)
		if bufStr != "" {
			println(bufStr)
		} else if bufStr == "" {
			// receive heartbeat continue
			continue
		} else {
			panic("unknow protocol")
		}
	}
}
