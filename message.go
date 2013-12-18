package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	// Message expired
	MsgExpiredErr = errors.New("Message already expired")
	MsgBufErr     = errors.New("Message writeu buffer nil")
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
		LogError(LogLevelErr, "json.Unmarshal() failed (%s), message json: \"%s\"", err.Error(), str)
		return nil, err
	}

	return m, nil
}

func (m *Message) Bytes(b *bytes.Buffer) ([]byte, error) {
	res := map[string]interface{}{
		"msg": m.Msg,
		"mid": m.MsgID,
	}

	byteJson, err := json.Marshal(res)
	if err != nil {
		LogError(LogLevelErr, "message write error, json.Marshal() failed (%s)", err.Error())
		return nil, err
	}

	if Conf.Protocol == TCPProtocol {
		if b == nil {
			return nil, MsgBufErr
		}

		// $size\r\ndata\r\n
		if _, err = b.WriteString(fmt.Sprintf("$%d\r\n", len(byteJson))); err != nil {
			LogError(LogLevelErr, "message write error, b.WriteString() failed (%s)", err.Error())
			return nil, err
		}

		if _, err = b.Write(byteJson); err != nil {
			LogError(LogLevelErr, "message write error, b.WriteString() failed (%s)", err.Error())
			return nil, err
		}

		if _, err = b.WriteString("\r\n"); err != nil {
			LogError(LogLevelErr, "message write error, b.WriteString() failed (%s)", err.Error())
			return nil, err
		}

		return b.Bytes(), nil
	} else {
		return byteJson, nil
	}
}
