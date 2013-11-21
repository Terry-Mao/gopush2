package main

import (
	"encoding/json"
	"github.com/Terry-Mao/gopush2/skiplist"
	"net"
	"sync"
	"time"
)

type InnerChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]bool
	// Stored message
	message *skiplist.SkipList
	// Auth token
	token map[string]bool
	// Subscriber expired unixnano
	expire int64
	// Max message stored number
	MaxMessage int
}

// New a inner message stored channel
func NewInnerChannel() *InnerChannel {
	c := &InnerChannel{}
	c.mutex = &sync.Mutex{}
	c.message = skiplist.New()
	c.conn = map[net.Conn]bool{}
	c.token = map[string]bool{}
	c.MaxMessage = Conf.MaxStoredMessage
	c.expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second
	subscriberStats.IncrCreated()

	return c
}

// SendMsg implements the Channel SendMsg method.
func (c *InnerChannel) SendMsg(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	now := time.Now().UnixNano()
	// find the next node
	for n := c.message.Greate(mid); n != nil; n = n.Next() {
		m, ok := n.Member.(*Message)
		if !ok {
			// never happen
			panic(ErrAssertType)
		}

		// check message expired
		if m.Expire >= now {
			if err := subRetWrite(conn, m.Msg, n.Score, key); err != nil {
				subscriberStats.IncrFailedMessage()
				Log.Printf("subRetWrite() failed (%s)", err.Error())
				return err
			}

			subscriberStats.IncrSentMessage()
		} else {
			// WARN:though the node deleted, can access the next node
			c.message.Delete(n.Score)
			subscriberStats.IncrExpiredMessage()
			Log.Printf("delete the expired message %d for device %s", n.Score, key)
		}
	}

	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *InnerChannel) PushMsg(msg string, expire int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	subscriberStats.IncrAddedMessage()
	now := time.Now().UnixNano()
	// check message expired
	if now >= expire {
		subscriberStats.IncrExpiredMessage()
		Log.Printf("device %s: message %d has already expired now(%d) >= expire(%d)", key, expire, now, expire)
		return ErrMsgExpired
	}

	// check exceed the max message length
	if c.message.Length+1 > c.MaxMessage {
		// remove the first node cause that's the smallest node
		n := c.message.Head.Next()
		if n == nil {
			// never happen
			Log.Printf("the subscriber touch a impossiable place")
			panic("Skiplist head nil")
		}

		c.message.Delete(n.Score)
		Log.Printf("device %s: message %d exceed the max message (%d) setting, trim the subscriber", key, n.Score, c.MaxMessage)
		subscriberStats.IncrDeletedMessage()
	}

	for {
		// if has exists node, sleep and retry
		err := c.message.Insert(now, &Message{Msg: msg, Expire: expire})
		if err != nil {
			now++
		}

		break
	}

	// send message to all the clients
	for conn, _ := range c.conn {
		if err := subRetWrite(conn, msg, now, key); err != nil {
			// remove exists conn
			subscriberStats.IncrFailedMessage()
			Log.Printf("subRetWrite() failed (%s)", err.Error())
			continue
		}

		subscriberStats.IncrSentMessage()
		Log.Printf("push message \"%s\":%d to device %s", msg, now, key)
	}

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *InnerChannel) AddConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	subscriberStats.IncrConn()
	// check exceed the maxsubscribers
	if len(c.conn)+1 > Conf.MaxSubscriberPerKey {
		return ErrMaxConn
	}

	Log.Printf("add conn for device %s", key)
	c.conn[conn] = true

	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *InnerChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	subscriberStats.DecrConn()
	Log.Printf("remove conn for device %s", key)
	delete(c.conn, conn)

	return nil
}

// AddToken implements the Channel AddToken method.
func (c *InnerChannel) AddToken(token string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.token[token]; ok {
		return ErrTokenExist
	}

	// token only used once
	c.token[token] = true

	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *InnerChannel) AuthToken(token string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.token[token]; !ok {
		return ErrAuthToken
	}

	// token only used once
	delete(c.token, token)

	return nil
}

// SetDeadline implements the Channel SetDeadline method.
func (c *InnerChannel) SetDeadline(d int64) {
	c.expire = d
}

// Timeout implements the Channel Timeout method.
func (c *InnerChannel) Timeout() bool {
	return time.Now().UnixNano() > c.expire
}

// Close implements the Channel Close method.
func (c *InnerChannel) Close() error {
	var retErr error

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for conn, _ := range c.conn {
		if err := conn.Close(); err != nil {
			retErr = err
			Log.Printf("conn.Close() failed (%s)", err.Error())
		}
	}

	return retErr
}

// subRetWrite json encoding the message and write to the conn.
func subRetWrite(conn net.Conn, msg string, msgID int64, key string) error {
	res := map[string]interface{}{}
	res["msg"] = msg
	res["msg_id"] = msgID

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
