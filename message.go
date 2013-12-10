package main

import (
	"encoding/json"
	"errors"
	"net"
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

	strJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(strJson)
	Log.Printf("device key: sub send to client: %s", respJson)
	if _, err := conn.Write(strJson); err != nil {
		Log.Printf("conn.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
}
