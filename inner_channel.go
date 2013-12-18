package main

import (
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

	return c
}

// SendMsg implements the Channel SendMsg method.
func (c *InnerChannel) SendMsg(conn net.Conn, mid int64, key string) error {
	// WARN: inner store must lock
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// find the next node
	for n := c.message.Greate(mid); n != nil; n = n.Next() {
		m, ok := n.Member.(*Message)
		if !ok {
			// never happen
			panic(AssertTypeErr)
		}

		// check message expired
		if m.Expired() {
			// WARN:though the node deleted, can access the next node
			c.message.Delete(n.Score)
			LogError(LogLevelWarn, "delete the expired message:%d for device:%s", n.Score, key)
		} else {
			b, err := m.Bytes(nil)
			if err != nil {
				LogError(LogLevelErr, "message.Bytes(nil) failed (%s)", err.Error())
				return err
			}

			if _, err := conn.Write(b); err != nil {
				return err
			}
		}
	}

	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *InnerChannel) PushMsg(m *Message, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check message expired
	if m.Expired() {
		LogError(LogLevelWarn, "message:%d has already expired for device:%s", m.MsgID, key)
		return MsgExpiredErr
	}

	// check exceed the max message length
	if c.message.Length+1 > c.MaxMessage {
		// remove the first node cause that's the smallest node
		n := c.message.Head.Next()
		if n == nil {
			// never happen
			LogError(LogLevelWarn, "the subscriber touch a impossiable place")
			panic("Skiplist head nil")
		}

		c.message.Delete(n.Score)
		LogError(LogLevelErr, "message:%d exceed the max message (%d) setting, trim the subscriber for device:%s", n.Score, c.MaxMessage, key)
	}

	err := c.message.Insert(m.MsgID, m)
	if err != nil {
		return err
	}

	b, err := m.Bytes(nil)
	if err != nil {
		LogError(LogLevelErr, "message.Bytes(nil) failed (%s)", err.Error())
		return err
	}

	// send message to all the clients
	for conn, _ := range c.conn {
		if _, err = conn.Write(b); err != nil {
			LogError(LogLevelErr, "message write error, conn.Write() failed (%s)", err.Error())
			continue
		}

		LogError(LogLevelInfo, "push message \"%s\":%d for device:%s", m.Msg, m.MsgID, key)
	}

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *InnerChannel) AddConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check exceed the maxsubscribers
	if len(c.conn)+1 > Conf.MaxSubscriberPerKey {
		return MaxConnErr
	}

	LogError(LogLevelInfo, "add conn for device:%s", key)
	c.conn[conn] = true

	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *InnerChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	LogError(LogLevelInfo, "remove conn for device:%s", key)
	delete(c.conn, conn)

	return nil
}

// AddToken implements the Channel AddToken method.
func (c *InnerChannel) AddToken(token string, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.token[token]; ok {
		return TokenExistErr
	}

	// token only used once
	c.token[token] = true

	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *InnerChannel) AuthToken(token string, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.token[token]; !ok {
		return AuthTokenErr
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for conn, _ := range c.conn {
		if err := conn.Close(); err != nil {
			// ignore close error
			LogError(LogLevelErr, "conn.Close() failed (%s)", err.Error())
		}
	}

	return nil
}
