package main

import (
	"encoding/json"
	"errors"
	"github.com/Terry-Mao/gopush2/hash"
	"net"
	"sync"
	"time"
)

const (
	Second           = int64(time.Second)
	InnerChannelType = 1
	RedisChannelType = 2
)

var (
	// Exceed the max subscriber per key
	MaxConnErr = errors.New("Exceed the max subscriber connection per key")
	// Assection type failed
	AssertTypeErr = errors.New("Subscriber assert type failed")
	// Auth token failed
	AuthTokenErr = errors.New("Auth token failed")
	// Token exists
	TokenExistErr = errors.New("Token already exist")
	// Message expired
	MsgExpiredErr = errors.New("Message already expired")
	// Channle not exists
	ChannelNotExistErr = errors.New("Channle not exist")
	// Channel expired
	ChannelExpiredErr = errors.New("Channel expired")
	// Channle type unknown
	ChannelTypeErr = errors.New("Channle type unknown")
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

// The subscriber interface
type Channel interface {
	// PushMsg push a message to the subscriber.
	PushMsg(m *Message, key string) error
	// SendMsg send messages which id greate than the request id to the subscriber.
	// net.Conn write failed will return errors.
	SendMsg(conn net.Conn, mid int64, key string) error
	// AddConn add a connection for the subscriber.
	// Exceed the max number of subscribers per key will return errors.
	AddConn(conn net.Conn, mid int64, key string) error
	// RemoveConn remove a connection for the  subscriber.
	RemoveConn(conn net.Conn, mid int64, key string) error
	// Add a token for one subscriber
	// The request token not equal the subscriber token will return errors.
	AddToken(token string, key string) error
	// Auth auth the access token.
	// The request token not match the subscriber token will return errors.
	AuthToken(token string, key string) error
	// SetDeadline set the channel deadline unixnano
	SetDeadline(d int64)
	// Timeout
	Timeout() bool
	// Expire expire the channle and clean data.
	Close() error
}

type channelBucket struct {
	data  map[string]Channel
	mutex *sync.Mutex
}

type ChannelList struct {
	channels []*channelBucket
}

var (
	channel *ChannelList
)

func NewChannelList() *ChannelList {
	l := &ChannelList{}
	l.channels = []*channelBucket{}
	// split hashmap to many bucket
	for i := 0; i < Conf.ChannelBucket; i++ {
		c := &channelBucket{
			data:  map[string]Channel{},
			mutex: &sync.Mutex{},
		}

		l.channels = append(l.channels, c)
	}

	if Conf.ChannelType == 2 {
		if err := InitRedisChannel(); err != nil {
			Log.Printf("init redis channle failed (%s)", err.Error())
			panic(err)
		}
	}

	return l
}

// get a bucket from channel
func (l *ChannelList) bucket(key string) *channelBucket {
	idx := hash.MurMurHash2(key) & uint(Conf.ChannelBucket-1)
	return l.channels[idx]
}

func (l *ChannelList) New(key string) (Channel, error) {
	var (
		c   Channel
		ok  bool
		now = time.Now().UnixNano()
	)

	// get a channel bucket
	b := l.bucket(key)
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if c, ok = b.data[key]; ok {
		// refresh the expire time
		c.SetDeadline(now + Conf.ChannelExpireSec*Second)
		channelStats.IncrRefreshed()
		return c, nil
	} else {
		if Conf.ChannelType == InnerChannelType {
			c = NewInnerChannel()
		} else if Conf.ChannelType == RedisChannelType {
			c = NewRedisChannel()
		} else {
			Log.Printf("unknown channel type : %d", Conf.ChannelType)
			return nil, ChannelTypeErr
		}

		b.data[key] = c
		channelStats.IncrCreated()
	}

	return c, nil
}

// get a subscriber from channel in pub/sub action
func (l *ChannelList) Get(key string) (Channel, error) {
	var (
		c  Channel
		ok bool
	)

	// get a channel bucket
	b := l.bucket(key)
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if c, ok = b.data[key]; !ok {
		return nil, ChannelNotExistErr
	} else {
		// check expired
		if c.Timeout() {
			channelStats.IncrExpired()
			Log.Printf("device %s: channle expired", key)
			delete(b.data, key)
			if err := c.Close(); err != nil {
				return nil, err
			}

			return nil, ChannelExpiredErr
		}
	}

	return c, nil
}
