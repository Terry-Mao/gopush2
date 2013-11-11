package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"github.com/Terry-Mao/gopush2/skiplist"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
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

var (
	chMutex = &sync.Mutex{}
	channel = map[string]*Subscriber{}
)

func AddChannel(key string) *Subscriber {
	var (
		s  *Subscriber
		ok bool
	)

	chMutex.Lock()
	defer chMutex.Unlock()

	if s, ok = channel[key]; !ok {
		s = NewSubscriber()
		channel[key] = s
        s.Key = key
	}

	return s
}

func NewSubscriber() *Subscriber {
	sub := &Subscriber{}
	sub.mutex = &sync.Mutex{}
	sub.message = skiplist.New()
	sub.conn = map[*websocket.Conn]bool{}
	sub.Expire = time.Now().UnixNano() + int64(Conf.MessageExpireSec)*1000
	sub.MaxMessage = Conf.MaxStoredMessage

	return sub
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

func (s *Subscriber) AddConn(ws *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

    Log.Printf("add websocket.Conn to %s", s.Key)
	s.conn[ws] = true
}

func (s *Subscriber) RemoveConn(ws *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

    Log.Printf("remove websocket.Conn to %s", s.Key)
	delete(s.conn, ws)
}

func (s *Subscriber) AddMessage(msg string, expire int64) {
	now := time.Now().UnixNano()
    if now >= expire {
        Log.Printf("message %s has already expired now(%d) >= expire(%d)", msg, now, expire)
        return
    }

	s.mutex.Lock()
	defer s.mutex.Unlock()

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
			time.Sleep(1 * time.Nanosecond)
			now = time.Now().UnixNano()
		}

		break
	}

	// send message to all the clients
	for ws, _ := range s.conn {
		if err := retWrite(ws, msg, now); err != nil {
			// remove exists conn
			// delete(s.conn, ws)
			Log.Printf("retWrite() failed (%s)", err.Error())
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
	expireStr := params.Get("expire")
	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		http.Error(w, "query parameter expire error", 405)
        return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body error", 500)
        return
	}

	// get subscriber
	sub := AddChannel(key)
	sub.AddMessage(string(body), expire*1000)

	return
}

func Subscribe(ws *websocket.Conn) {
	defer recoverFunc()

	params := ws.Request().URL.Query()
	subKey := params.Get("key")
	//TODO auth
	// get lastest message id
	midStr := ""
	if err := websocket.Message.Receive(ws, &midStr); err != nil {
		Log.Printf("websocket.Message.Receive() failed (%s)", err.Error())
        return
	}

	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		Log.Printf("argument error (%s)", err.Error())
		return
	}

	Log.Printf("client (%s) subscribe to key %s with mid = %s", ws.Request().RemoteAddr, subKey, midStr)
	// get subscriber
	sub := AddChannel(subKey)
	// add a conn to the subscriber
	sub.AddConn(ws)
	// remove exists conn
	defer sub.RemoveConn(ws)
	// send stored message
	msgs, scores := sub.Message(mid)
	if msgs != nil && scores != nil {
		for i := 0; i < len(msgs); i++ {
			msg := msgs[i]
			score := scores[i]
			if err := retWrite(ws, msg, score); err != nil {
				// remove exists conn
				// delete(s.conn, ws)
				Log.Printf("retWrite() failed (%s)", err.Error())
				return
			}
		}
	}

	// blocking untill someone pub the key
	reply := ""
	if err := websocket.Message.Receive(ws, &reply); err != nil {
		Log.Printf("websocket.Message.Receive() failed (%s)", err.Error())
	}

	return
}

func retWrite(ws *websocket.Conn, msg string, msgID int64) error {
	res := map[string]interface{}{}
	res["msg"] = msg
	res["msg_id"] = msgID

	strJson, err := json.Marshal(res)
	if err != nil {
		Log.Printf("json.Marshal(\"%v\") failed", res)
		return err
	}

	respJson := string(strJson)
	Log.Printf("send to client: %s", respJson)
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
