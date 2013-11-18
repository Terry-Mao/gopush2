package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"github.com/Terry-Mao/gopush2/skiplist"
	"sync"
	"time"
)

const (
	Second = int64(time.Second)
)

var (
	ErrMaxConn = errors.New("Exceed the max subscriber connection per key")
	ErrExpired = errors.New("Channel expired")
)

type Subscriber struct {
	mutex      *sync.Mutex              // mutex
	message    *skiplist.SkipList       // message stored struct
	conn       map[*websocket.Conn]bool // stored conn
	Token      string                   // auth token
	Expire     int64                    // absolute expired unix nano
	MaxMessage int                      // max message a subscriber can stored
	Key        string                   // the sub key
}

// new a subscriber
func NewSubscriber(key string) *Subscriber {
	s := &Subscriber{}
	s.mutex = &sync.Mutex{}
	s.message = skiplist.New()
	s.conn = map[*websocket.Conn]bool{}
	s.Expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second
	s.MaxMessage = Conf.MaxStoredMessage
	s.Key = key

	subscriberStats.IncrCreated()
	return s
}

// send mssage after mid
func (s *Subscriber) SendStoredMessage(ws *websocket.Conn, mid int64) error {
	now := time.Now().UnixNano()
	expired := []int64{}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer func() {
		// delete the expired message
		for _, score := range expired {
			s.message.Delete(score)
			Log.Printf("delete the expired message %d for device %s", score, s.Key)
		}
	}()

	for n := s.message.Greate(mid); n != nil; n = n.Next() {
		// the next node may expired, recheck
		if n.Expire >= now {
			if err := subRetWrite(ws, n.Member, n.Score); err != nil {
				Log.Printf("subRetWrite() failed (%s)", err.Error())
				return err
			}

			subscriberStats.IncrSentMessage()
		} else {
			expired = append(expired, n.Score)
			subscriberStats.IncrExpiredMessage()
		}
	}

	return nil
}

// add a connection
func (s *Subscriber) AddConn(ws *websocket.Conn) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subscriberStats.IncrConn()
	if len(s.conn)+1 > Conf.MaxSubscriberPerKey {
		return ErrMaxConn
	}

	Log.Printf("add websocket.Conn to %s", s.Key)
	s.conn[ws] = true

	return nil
}

// remove a connection
func (s *Subscriber) RemoveConn(ws *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subscriberStats.DecrConn()
	Log.Printf("remove websocket.Conn to %s", s.Key)
	delete(s.conn, ws)

	return
}

// close and remove all the connections
func (s *Subscriber) CloseAllConn() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// TODO check expired
	for ws, _ := range s.conn {
		Log.Printf("close websocket.Conn to %s", s.Key)
		if err := ws.Close(); err != nil {
			Log.Printf("ws.Close() failed (%s)", err.Error())
		}

		delete(s.conn, ws)
		subscriberStats.DecrConn()
	}

	return
}

// publish message to the subscriber
func (s *Subscriber) PublishMessage(msg string, expire int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subscriberStats.IncrAddedMessage()
	now := time.Now().UnixNano()
	if now >= expire {
		subscriberStats.IncrExpiredMessage()
		Log.Printf("message %s has already expired now(%d) >= expire(%d)", msg, now, expire)
		return
	}

	//TODO check expired
	// check exceed the max message length
	if s.message.Length+1 > s.MaxMessage {
		// remove the first node cause that's the smallest node
		n := s.message.Head.Next()
		if n == nil {
			// never happen
			Log.Printf("the subscriber touch a impossiable place")
		}

		s.message.Delete(n.Score)
		Log.Printf("key %s:%d exceed the max_message setting, trim the subscriber", s.Key, n.Score)
		subscriberStats.IncrDeletedMessage()
	}

	for {
		// if has exists node, sleep and retry
		err := s.message.Insert(now, msg, expire)
		if err != nil {
			now++
		}

		break
	}

	// send message to all the clients
	for ws, _ := range s.conn {
		if err := subRetWrite(ws, msg, now); err != nil {
			// remove exists conn
			// delete(s.conn, ws)
			Log.Printf("subRetWrite() failed (%s)", err.Error())
		}

		Log.Printf("add message %s:%d to device %s", msg, now, s.Key)
		subscriberStats.IncrSentMessage()
	}

	return
}

func subRetWrite(ws *websocket.Conn, msg string, msgID int64) error {
	res := map[string]interface{}{}
	res["msg"] = msg
	res["msg_id"] = msgID

	strJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(strJson)
	Log.Printf("sub send to client: %s", respJson)
	if _, err := ws.Write(strJson); err != nil {
		Log.Printf("ws.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
}
