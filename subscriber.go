package main

import (
	"net"
	// "code.google.com/p/go.net/websocket"
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
	// ErrExpired = errors.New("Channel expired")
)

type Subscriber struct {
	mutex      *sync.Mutex        // mutex
	message    *skiplist.SkipList // message stored struct
	conn       map[net.Conn]bool  // stored conn
	Token      string             // auth token
	Expire     int64              // absolute expired unix nano
	MaxMessage int                // max message a subscriber can stored
	Key        string             // the sub key
}

// new a subscriber
func NewSubscriber(key string) *Subscriber {
	s := &Subscriber{}
	s.mutex = &sync.Mutex{}
	s.message = skiplist.New()
	s.conn = map[net.Conn]bool{}
	s.Expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second
	s.MaxMessage = Conf.MaxStoredMessage
	s.Key = key

	subscriberStats.IncrCreated()
	return s
}

// send mssage after mid
func (s *Subscriber) SendStoredMessage(conn net.Conn, mid int64) error {
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
			if err := subRetWrite(conn, n.Member, n.Score); err != nil {
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
func (s *Subscriber) AddConn(conn net.Conn) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subscriberStats.IncrConn()
	if len(s.conn)+1 > Conf.MaxSubscriberPerKey {
		return ErrMaxConn
	}

	Log.Printf("add Conn to %s", s.Key)
	s.conn[conn] = true

	return nil
}

// remove a connection
func (s *Subscriber) RemoveConn(conn net.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subscriberStats.DecrConn()
	Log.Printf("remove conn to %s", s.Key)
	delete(s.conn, conn)

	return
}

// close and remove all the connections
func (s *Subscriber) CloseAllConn() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// TODO check expired
	for conn, _ := range s.conn {
		Log.Printf("close websocket.Conn to %s", s.Key)
		if err := conn.Close(); err != nil {
			Log.Printf("conn.Close() failed (%s)", err.Error())
		}

		delete(s.conn, conn)
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
	for conn, _ := range s.conn {
		if err := subRetWrite(conn, msg, now); err != nil {
			// remove exists conn
			Log.Printf("subRetWrite() failed (%s)", err.Error())
		}

		Log.Printf("add message %s:%d to device %s", msg, now, s.Key)
		subscriberStats.IncrSentMessage()
	}

	return
}

func subRetWrite(conn net.Conn, msg string, msgID int64) error {
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
	if _, err := conn.Write(strJson); err != nil {
		Log.Printf("conn.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
}
