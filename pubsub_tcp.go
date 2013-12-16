package main

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	fitstPacketTimedoutSec = 5
)

var (
	CmdFmtErr = errors.New("cmd format error")
)

func StartTCP() error {
	addr, err := net.ResolveTCPAddr("tcp", Conf.Addr)
	if err != nil {
		LogError(LogLevelErr, "net.ResolveTCPAddr(\"tcp\"), %s) failed (%s)", Conf.Addr, err.Error())
		return err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		LogError(LogLevelErr, "net.ListenTCP(\"tcp4\", \"%s\") failed (%s)", Conf.Addr, err.Error())
		return err
	}

	// free the listener resource
	defer func() {
		if err := l.Close(); err != nil {
			LogError(LogLevelErr, "listener.Close() failed (%s)", err.Error())
		}
	}()

	// loop for accept conn
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			LogError(LogLevelErr, "listener.AcceptTCP() failed (%s)", err.Error())
			continue
		}

		if err = conn.SetKeepAlive(Conf.TCPKeepAlive == 1); err != nil {
			LogError(LogLevelErr, "conn.SetKeepAlive() failed (%s)", err.Error())
			conn.Close()
			continue
		}

		if err = conn.SetReadBuffer(Conf.ReadBufByte); err != nil {
			LogError(LogLevelErr, "conn.SetReadBuffer(%d) failed (%s)", Conf.ReadBufByte, err.Error())
			conn.Close()
			continue
		}

		if err = conn.SetWriteBuffer(Conf.WriteBufByte); err != nil {
			LogError(LogLevelErr, "conn.SetWriteBuffer(%d) failed (%s)", Conf.WriteBufByte, err.Error())
			conn.Close()
			continue
		}

		// first packet must sent by client in 5 seconds
		if err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(fitstPacketTimedoutSec))); err != nil {
			LogError(LogLevelErr, "conn.SetReadDeadLine() failed (%s)", err.Error())
			break
		}

		go handleTCPConn(conn)
	}

	// nerve here
	return nil
}

func handleTCPConn(conn net.Conn) {
	LogError(LogLevelInfo, "handleTcpConn routine start")
	// parse protocol reference: http://redis.io/topics/protocol (use redis protocol)
	rd := bufio.NewReaderSize(conn, Conf.ReadBufByte)
	if args, err := parseCmd(rd); err == nil {
		switch args[0] {
		case "sub":
			SubscribeTCPHandle(conn, args[1:])
			break
		default:
			LogError(LogLevelWarn, "tcp proto:unknown cmd \"%s\"", args[0])
			break
		}
	} else {
		LogError(LogLevelErr, "parseCmd() failed (%s)", err.Error())
	}

	// close the connection
	if err := conn.Close(); err != nil {
		LogError(LogLevelErr, "conn.Close() failed (%s)", err.Error())
	}

	LogError(LogLevelInfo, "handleTcpConn routine stop")
	return
}

// SubscribeTCPHandle handle the subscribers's connection
func SubscribeTCPHandle(conn net.Conn, args []string) {
	argLen := len(args)
	if argLen < 2 {
		LogError(LogLevelWarn, "subscriber missing argument")
		return
	}

	// key, mid, heartbeat, token
	key := args[0]
	midStr := args[1]
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		LogError(LogLevelErr, "mid:\"%s\" argument error (%s)", midStr, err.Error())
		return
	}

	heartbeat := Conf.HeartbeatSec
	heartbeatStr := ""
	if argLen > 2 {
		heartbeatStr = args[2]
		i, err := strconv.Atoi(heartbeatStr)
		if err != nil {
			LogError(LogLevelErr, "heartbeat:\"%s\" argument error (%s)", heartbeatStr, err.Error())
			return
		}

		heartbeat = i
	}

	heartbeat *= 2
	if heartbeat <= 0 {
		LogError(LogLevelWarn, "device:%s heartbeat argument error, less than 0", key)
		return
	}

	token := ""
	if argLen > 3 {
		token = args[3]
	}

	LogError(LogLevelInfo, "client:%s subscribe to key = %s, mid = %d, token = %s, heartbeat = %d", conn.RemoteAddr().String(), key, mid, token, heartbeat)
	// fetch subscriber from the channel
	c, err := channel.Get(key)
	if err != nil {
		if Conf.Auth == 0 {
			c, err = channel.New(key)
			if err != nil {
				LogError(LogLevelErr, "device:%s can't create channle (%s)", key, err.Error())
				return
			}
		} else {
			LogError(LogLevelWarn, "device:%s can't get a channel (%s)", key, err.Error())
			return
		}
	}

	// auth
	if Conf.Auth == 1 {
		if err = c.AuthToken(token, key); err != nil {
			LogError(LogLevelErr, "device:%s auth token failed \"%s\" (%s)", key, token, err.Error())
			return
		}
	}

	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err := conn.Write(heartbeatBytes); err != nil {
		LogError(LogLevelErr, "device:%s write first heartbeat to client failed (%s)", key, err.Error())
		return
	}

	// send stored message, and use the last message id if sent any
	if err = c.SendMsg(conn, mid, key); err != nil {
		LogError(LogLevelErr, "device:%s send offline message failed (%s)", key, err.Error())
		return
	}

	// add a conn to the channel
	if err = c.AddConn(conn, mid, key); err != nil {
		LogError(LogLevelErr, "device:%s add conn failed (%s)", key, err.Error())
		return
	}

	// blocking wait client heartbeat
	reply := make([]byte, heartbeatByteLen)
	for {
		if err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat))); err != nil {
			LogError(LogLevelErr, "conn.SetReadDeadLine() failed (%s)", err.Error())
			break
		}

		if _, err = conn.Read(reply); err != nil {
			if err != io.EOF {
				LogError(LogLevelErr, "device:%s conn.Read() failed, read heartbeat timedout (%s)", key, err.Error())
			} else {
				// client connection close
				LogError(LogLevelInfo, "device:%s client connection close", key)
			}

			break
		}

		if string(reply) == heartbeatMsg {
			if _, err = conn.Write(heartbeatBytes); err != nil {
				LogError(LogLevelErr, "device:%s conn.Write() failed, write heartbeat to client (%s)", key, err.Error())
				break
			}

			LogError(LogLevelInfo, "device:%s receive heartbeat", key)
		} else {
			LogError(LogLevelWarn, "device:%s unknown heartbeat protocol", key)
			break
		}
	}

	// remove exists conn
	if err := c.RemoveConn(conn, mid, key); err != nil {
		LogError(LogLevelErr, "device:%s remove conn failed (%s)", key, err.Error())
	}

	return
}

func parseCmd(rd *bufio.Reader) ([]string, error) {
	// get argument number
	argNum, err := parseCmdSize(rd, '*')
	if err != nil {
		LogError(LogLevelErr, "tcp proto cmd:argument number error (%s)", err.Error())
		return nil, err
	}

	if argNum < 1 {
		LogError(LogLevelWarn, "tcp proto cmd:cmd argument number length error")
		return nil, CmdFmtErr
	}

	args := make([]string, 0, argNum)
	for i := 0; i < argNum; i++ {
		// get argument length
		cmdLen, err := parseCmdSize(rd, '$')
		if err != nil {
			LogError(LogLevelErr, "parseCmdSize(rd, '$') failed (%s)", err.Error())
			return nil, err
		}

		// get argument data
		d, err := parseCmdData(rd, cmdLen)
		if err != nil {
			LogError(LogLevelErr, "parseCmdData failed() (%s)", err.Error())
			return nil, err
		}

		// append args
		args = append(args, string(d))
	}

	return args, nil
}

// parseCmdSize get the sub request protocol cmd size
func parseCmdSize(rd *bufio.Reader, prefix uint8) (int, error) {
	// get command size
	cs, err := rd.ReadBytes('\n')
	if err != nil {
		LogError(LogLevelErr, "tcp proto cmd:rd.ReadBytes('\\n') failed (%s)", err.Error())
		return 0, err
	}

	csl := len(cs)
	if csl <= 3 || cs[0] != prefix || cs[csl-2] != '\r' {
		LogError(LogLevelWarn, "tcp proto cmd:\"%v\"(%d) number format error, length error or prefix error or no \\r", cs, csl)
		return 0, CmdFmtErr
	}

	// skip the \r\n
	cmdSize, err := strconv.Atoi(string(cs[1 : csl-2]))
	if err != nil {
		LogError(LogLevelErr, "tcp proto cmd:\"%v\" number parse int failed (%s)", cs, err.Error())
		return 0, CmdFmtErr
	}

	return cmdSize, nil
}

// parseCmdData get the sub request protocol cmd data not included \r\n
func parseCmdData(rd *bufio.Reader, cmdLen int) ([]byte, error) {
	d, err := rd.ReadBytes('\n')
	if err != nil {
		LogError(LogLevelErr, "tcp proto cmd:rd.ReadBytes('\\n') failed (%s)", err.Error())
		return nil, err
	}

	dl := len(d)
	// check last \r\n
	if dl != cmdLen+2 || d[dl-2] != '\r' {
		LogError(LogLevelWarn, "tcp proto cmd:\"%v\"(%d) number format error, length error or no \\r", d, dl)
		return nil, CmdFmtErr
	}

	// skip last \r\n
	return d[0 : dl-2], nil
}
