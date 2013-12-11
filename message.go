package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

var (
	// Message expired
	MsgExpiredErr = errors.New("Message already expired")
)

// The Message struct
type Message struct {
	// Message
	Msg string `json:"msg"`
	// Message expired unixnano
	Expire int64 `json:"expire"`
	// Message id
	MsgID int64 `json:"mid"`
}

// Expired check mesage expired or not
func (m *Message) Expired() bool {
	return time.Now().UnixNano() > m.Expire
}

func NewJsonStrMessage(str string) (*Message, error) {
	m := &Message{}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		Log.Printf("json.Unmarshal(\"%s\", &message) failed (%s)", str, err.Error())
		return nil, err
	}

	return m, nil
}

// Write json encoding the message and write to the conn.
func (m *Message) Write(conn net.Conn, key string) error {
	res := map[string]interface{}{}
	res["msg"] = m.Msg
	res["mid"] = m.MsgID

	byteJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(byteJson)
	Log.Printf("device key: sub send to client: %s", respJson)
	buf := byteJson
	// TCP Protocol use redis reply, reference: http://redis.io/topics/protocol
	if Conf.Protocol == TCPProtocol {
		dl := len(byteJson)
		nl := len(strconv.Itoa(dl))
		// $size\r\ndata\r\n
		buf = make([]byte, 1+nl+2+dl+2)
		copy(buf, []byte(fmt.Sprintf("$%d\r\n", dl)))
		copy(buf[1+nl+2:], byteJson)
		copy(buf[1+nl+2+dl:], []byte("\r\n"))
	}

	if _, err := conn.Write(buf); err != nil {
		Log.Printf("conn.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
}
