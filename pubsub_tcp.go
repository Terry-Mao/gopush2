package main

import (
	"bufio"
	"errors"
	"net"
	"strconv"
	"time"
)

var (
	CmdFmtErr  = errors.New("cmd format error")
	CmdSizeErr = errors.New("cmd data size error")
)

func StartTCP() error {
	addr, err := net.ResolveTCPAddr("tcp", Conf.Addr)
	if err != nil {
		Log.Printf("net.ResolveTCPAddr(\"tcp\"), %s) failed (%s)", Conf.Addr, err.Error())
		return err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		Log.Printf("net.ListenTCP(\"tcp4\", \"%s\") failed (%s)", Conf.Addr, err.Error())
		return err
	}

	// free the listener resource
	defer func() {
		if err := l.Close(); err != nil {
			Log.Printf("l.Close() failed (%s)", err.Error())
		}
	}()

	// loop for accept conn
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			Log.Printf("l.AcceptTCP() failed (%s)", err.Error())
			continue
		}

		if err = conn.SetKeepAlive(Conf.TCPKeepAlive == 1); err != nil {
			Log.Printf("conn.SetKeepAlive() failed (%s)", err.Error())
			conn.Close()
			continue
		}

		if err = conn.SetReadBuffer(Conf.ReadBufByte); err != nil {
			Log.Printf("conn.SetReadBuffer(%d) failed (%s)", Conf.ReadBufByte, err.Error())
			conn.Close()
			continue
		}

		if err = conn.SetWriteBuffer(Conf.WriteBufByte); err != nil {
			Log.Printf("conn.SetWriteBuffer(%d) failed (%s)", Conf.WriteBufByte, err.Error())
			conn.Close()
			continue
		}

		// TODO read timeout

		go handleTCPConn(conn)
	}

	// nerve here
	return nil
}

func handleTCPConn(conn net.Conn) {
	defer recoverFunc()
	defer func() {
		if err := conn.Close(); err != nil {
			Log.Printf("conn.Close() failed (%s)", err.Error())
		}
	}()

	// parse protocol reference: http://redis.io/topics/protocol (use redis protocol)
	rd := bufio.NewReaderSize(conn, Conf.ReadBufByte)
	// get argument number
	argNum, err := parseCmdSize(rd, '*')
	if err != nil {
		Log.Printf("parse cmd argument number error")
		return
	}

	if argNum < 1 {
		Log.Printf("parse cmd argument number length error")
		return
	}

	args := make([]string, 0, argNum)
	for i := 0; i < argNum; i++ {
		// get argument length
		cmdLen, err := parseCmdSize(rd, '$')
		if err != nil {
			Log.Printf("parse cmd first argument size error")
			return
		}

		// get argument data
		d, err := parseCmdData(rd, cmdLen)
		if err != nil {
			Log.Printf("parse cmd data error (%s)", err.Error())
			return
		}

		// append args
		args = append(args, string(d))
	}

	switch args[0] {
	case "sub":
		SubscribeTCPHandle(conn, args[1:])
		break
	default:
		Log.Printf("tcp protocol unknown cmd: %s", args[0])
		return
	}

	return
}

// SubscribeTCPHandle handle the subscribers's connection
func SubscribeTCPHandle(conn net.Conn, args []string) {
	argLen := len(args)
	if argLen < 2 {
		Log.Printf("subscriber missing argument")
		return
	}

	// key, mid, heartbeat, token
	key := args[0]
	midStr := args[1]
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		Log.Printf("mid argument error (%s)", err.Error())
		return
	}

	heartbeat := Conf.HeartbeatSec
	heartbeatStr := ""
	if argLen > 2 {
		heartbeatStr = args[2]
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

	token := ""
	if argLen > 3 {
		token = args[3]
	}

	Log.Printf("client %s subscribe to key = %s, mid = %d, token = %s, heartbeat = %d", conn.RemoteAddr().String(), key, mid, token, heartbeat)
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
	if _, err := conn.Write(heartbeatBytes); err != nil {
		Log.Printf("device %s: write first heartbeat to client failed (%s)", key, err.Error())
		return
	}

	// send stored message, and use the last message id if sent any
	if err = c.SendMsg(conn, mid, key); err != nil {
		Log.Printf("device %s: send offline message failed (%s)", key, err.Error())
		return
	}

	// add a conn to the channel
	if err = c.AddConn(conn, mid, key); err != nil {
		Log.Printf("device %s: add conn failed (%s)", key, err.Error())
		return
	}

	// remove exists conn
	defer func() {
		if err := c.RemoveConn(conn, mid, key); err != nil {
			Log.Printf("device %s: remove conn failed (%s)", key, err.Error())
		}
	}()

	// blocking wait client heartbeat
	reply := make([]byte, heartbeatByteLen)
	for {
		if err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat))); err != nil {
			Log.Printf("conn.SetReadDeadLine() failed (%s)", err.Error())
			return
		}

		if _, err = conn.Read(reply); err != nil {
			Log.Printf("conn.Read() failed (%s)", err.Error())
			return
		}

		if string(reply) == heartbeatMsg {
			if _, err = conn.Write(heartbeatBytes); err != nil {
				Log.Printf("device %s: write heartbeat to client failed (%s)", key, err.Error())
				return
			}

			Log.Printf("device %s: receive heartbeat", key)
		} else {
			Log.Printf("device %s: unknown heartbeat protocol", key)
			return
		}
	}

	return
}

// parseCmdSize get the sub request protocol cmd size
func parseCmdSize(rd *bufio.Reader, prefix uint8) (int, error) {
	cmd := ""
	for {
		str, err := rd.ReadString('\n')
		if err != nil {
			Log.Printf("rd.ReadBytes('\\n') failed (%s)", err.Error())
			return 0, err
		}

		cmd += str
		if len(cmd) > 2 && cmd[len(cmd)-2] == '\r' {
			break
		}
	}

	cmdLen := len(cmd)
	if cmdLen <= 3 || cmd[0] != prefix {
		Log.Printf("tcp protocol cmd: %s(%d) number format error", cmd, cmdLen)
		return 0, CmdFmtErr
	}

	// skip the \r\n
	cmdSize, err := strconv.Atoi(cmd[1 : cmdLen-2])
	if err != nil {
		Log.Printf("tcp protocol cmd: %s number parse int failed (%s)", cmd, err.Error())
		return 0, CmdFmtErr
	}

	return cmdSize, nil
}

// parseCmdData get the sub request protocol cmd data not included \r\n
func parseCmdData(rd *bufio.Reader, cmdLen int) ([]byte, error) {
	rcmdLen := cmdLen + 2
	buf := make([]byte, rcmdLen)
	r := 0
	for {
		n, err := rd.Read(buf[r:])
		if err != nil {
			return nil, err
		}

		if r = r + n; r < rcmdLen {
			continue
		} else if r == rcmdLen {
			break
		} else {
			Log.Printf("tcp protocol parse data exceed the cmd size")
			return nil, CmdSizeErr
		}
	}

	// check last \r\n
	if buf[cmdLen] != '\r' || buf[cmdLen+1] != '\n' {
		return nil, CmdFmtErr
	}

	// skip last \r\n
	return buf[0:cmdLen], nil
}
