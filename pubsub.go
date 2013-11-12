package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"github.com/Terry-Mao/gopush2/skiplist"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

const (
	Second            = int64(time.Second)
	RetInternalErr    = 65535
	RetChannelExpired = 1
	RetOK             = 0
)

var (
	chMutex    = &sync.Mutex{}
	channel    = map[string]*Subscriber{}
	ErrMaxConn = errors.New("Exceed the max subscriber connection per key")
	ErrExpired = errors.New("Channel expired")
)

func subChannel(key string) *Subscriber {
	var (
		s   *Subscriber
		ok  bool
		now = time.Now().UnixNano()
	)

	chMutex.Lock()
	defer chMutex.Unlock()

	if s, ok = channel[key]; !ok {
		// not exists subscriber for the key
		s = NewSubscriber()
		channel[key] = s
		s.Key = key
	} else {
		// check expired
		if now >= s.Expire {
			// let gc free the old subscriber
			Log.Printf("device %s drop the expired channel, refresh a new one now(%d) > expire(%d)", key, now, s.Expire)
			s = NewSubscriber()
			s.Key = key
			channel[key] = s
		} else {
			// refresh the expire time
			s.Expire = now + int64(Conf.ChannelExpireSec)*Second
		}
	}

	return s
}

func pubChannel(key string) (*Subscriber, error) {
	var (
		s   *Subscriber
		ok  bool
		now = time.Now().UnixNano()
	)

	chMutex.Lock()
	defer chMutex.Unlock()

	if s, ok = channel[key]; !ok {
		// not exists subscriber for the key
		s = NewSubscriber()
		channel[key] = s
		s.Key = key
	} else {
		// check expired
		if now >= s.Expire {
			// let gc free the old subscriber
			Log.Printf("device %s drop the expired channel, now(%d) > expire(%d)", key, now, s.Expire)
			// drop the key
			delete(channel, key)
			return nil, ErrExpired
		}
	}

	return s, nil
}

type Subscriber struct {
	mutex      *sync.Mutex
	message    *skiplist.SkipList
	conn       map[*websocket.Conn]bool
	Token      string
	Expire     int64
	MaxMessage int
	Key        string
}

func NewSubscriber() *Subscriber {
	s := &Subscriber{}
	s.mutex = &sync.Mutex{}
	s.message = skiplist.New()
	s.conn = map[*websocket.Conn]bool{}
	s.Expire = time.Now().UnixNano() + int64(Conf.ChannelExpireSec)*Second
	s.MaxMessage = Conf.MaxStoredMessage

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
		Log.Printf("key %s exceed the maxmessage setting, trim the subscriber", s.Key)
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

func Publish(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	params := r.URL.Query()
	key := params.Get("key")
	//TODO auth
	// get the expired sec
	expireStr := params.Get("expire")
	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		// use default setting
		expire = int64(Conf.MessageExpireSec) * Second
	}

	expire = time.Now().UnixNano() + expire*Second
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if err = pubRetWrite(w, "read http body error", RetInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	// fetch subscriber from the channel
	sub, err := pubChannel(key)
	if sub != nil {
		sub.AddMessage(string(body), expire)
	} else if err == ErrExpired {
		if err = pubRetWrite(w, "channel expired", RetChannelExpired); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	} else {
		if err = pubRetWrite(w, "unknown error", RetInternalErr); err != nil {
			Log.Printf("pubRetWrite() failed (%s)", err.Error())
		}

		return
	}

	if err = pubRetWrite(w, "ok", RetOK); err != nil {
		Log.Printf("pubRetWrite() failed (%s)", err.Error())
	}

	return
}

func Subscribe(ws *websocket.Conn) {
	defer recoverFunc()

	params := ws.Request().URL.Query()
	subKey := params.Get("key")
	//TODO auth
	// get lastest message id

	midStr := params.Get("msg_id")
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		Log.Printf("argument error (%s)", err.Error())
		return
	}

	Log.Printf("client (%s) subscribe to key %s with msg_id = %s", ws.Request().RemoteAddr, subKey, midStr)
	// fetch subscriber from the channel
	sub := subChannel(subKey)
	// add a conn to the subscriber
	if err := sub.AddConn(ws); err != nil {
		Log.Printf("sub.AddConn failed (%s)", err.Error())
		return
	}

	// remove exists conn
	defer sub.RemoveConn(ws)
	// send stored message
	msgs, scores := sub.Message(mid)
	if msgs != nil && scores != nil {
		for i := 0; i < len(msgs); i++ {
			msg := msgs[i]
			score := scores[i]
			if err = subRetWrite(ws, msg, score); err != nil {
				// remove exists conn
				// delete(s.conn, ws)
				Log.Printf("subRetWrite() failed (%s)", err.Error())
				return
			}
		}
	}

	// blocking untill someone pub the key
	reply := ""
	if err = websocket.Message.Receive(ws, &reply); err != nil {
		Log.Printf("websocket.Message.Receive() failed (%s)", err.Error())
	}

	return
}

func pubRetWrite(w http.ResponseWriter, msg string, ret int) error {
	res := map[string]interface{}{}
	res["msg"] = msg
	res["ret"] = ret

	strJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(strJson)
	Log.Printf("pub send to client: %s", respJson)
	if _, err := w.Write(strJson); err != nil {
		Log.Printf("w.Write(\"%s\") failed (%s)", respJson, err.Error())
		return err
	}

	return nil
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

func recoverFunc() {
	if err := recover(); err != nil {
		Log.Printf("Error : %v, Debug : \n%s", err, string(debug.Stack()))
	}
}
