package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/Terry-Mao/gopush2/skiplist"
	"sync"
	"time"
)

const (
	Second = int64(time.Second)
)

type Subscriber struct {
	mutex      *sync.Mutex
	message    *skiplist.SkipList
	conn       map[*websocket.Conn]bool
	Token      string
	Expire     int64
	MaxMessage int
	Key        string
}

func NewSubscriber(key string) *Subscriber {
	s := &Subscriber{}
	s.mutex = &sync.Mutex{}
	s.message = skiplist.New()
	s.conn = map[*websocket.Conn]bool{}
	s.Expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second
	s.MaxMessage = Conf.MaxStoredMessage
	s.Key = key

	return s
}

// get greate than mid's messages
func (s *Subscriber) Message(mid int64) ([]string, []int64) {
	now := time.Now().UnixNano()
	msgs := []string{}
	scores := []int64{}
	expired := []int64{}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	n := s.message.Greate(mid)
	if n == nil {
		return nil, nil
	}

	// check expired
	if n.Expire >= now {
		msgs = append(msgs, n.Member)
		scores = append(scores, n.Score)
	} else {
		expired = append(expired, n.Score)
	}

	for n = n.Next(); n != nil; n = n.Next() {
		// the next node may expired, recheck
		if n.Expire >= now {
			msgs = append(msgs, n.Member)
			scores = append(scores, n.Score)
		} else {
			expired = append(expired, n.Score)
		}
	}

	// delete the expired message
	for _, score := range expired {
		s.message.Delete(score)
		Log.Printf("delete the expired message %d for device %s", score, s.Key)
	}

	return msgs, scores
}

func (s *Subscriber) AddConn(ws *websocket.Conn) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.conn)+1 > Conf.MaxSubscriberPerKey {
		return ErrMaxConn
	}

	Log.Printf("add websocket.Conn to %s", s.Key)
	s.conn[ws] = true

	return nil
}

func (s *Subscriber) RemoveConn(ws *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	Log.Printf("remove websocket.Conn to %s", s.Key)
	delete(s.conn, ws)
}

func (s *Subscriber) CloseAllConn() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for ws, _ := range s.conn {
		Log.Printf("close websocket.Conn to %s", s.Key)
		if err := ws.Close(); err != nil {
			Log.Printf("ws.Close() failed (%s)", err.Error())
		}

		delete(s.conn, ws)
	}
}

func (s *Subscriber) AddMessage(msg string, expire int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().UnixNano()
	if now >= expire {
		Log.Printf("message %s has already expired now(%d) >= expire(%d)", msg, now, expire)
		return
	}

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
	}

	return
}
