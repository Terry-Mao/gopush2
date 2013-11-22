package main

import (
	"net"
	"sync"
	"time"
)

type RedisChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]int64
	// Channel expired unixnano
	expire int64
	// Max message stored number
	MaxMessage int
}

// New a redis message stored channel
func NewRedisChannel(key string) *RedisChannel {
	c := &RedisChannel{}
	c.mutex = &sync.Mutex{}
	c.conn = map[net.Conn]int64{}
	c.expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second
	c.MaxMessage = Conf.MaxStoredMessage

	return c
}

// PushMsg implements the Channel PushMsg method.
func (c *RedisChannel) PushMsg(msg string, expire int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// send message to each conn when message id > conn last message id

	return nil
}

// SendMsg implements the Channel SendMsg method.
func (c *RedisChannel) SendMsg(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// get offline message from redis which greate mid
	// delete the expired message
	// update the last message id for conn

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *RedisChannel) AddConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	subscriberStats.IncrConn()
	// check exceed the maxsubscribers
	if len(c.conn)+1 > Conf.MaxSubscriberPerKey {
		c.mutex.Unlock()
		return ErrMaxConn
	}

	Log.Printf("add conn for device %s", key)
	// save the last push message id
	c.conn[conn] = mid
	c.mutex.Unlock()

	// store the online state in redis hashes (HINCRBY)

	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *RedisChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	subscriberStats.DecrConn()
	Log.Printf("remove conn for device %s", key)
	delete(c.conn, conn)
	c.mutex.Unlock()

	// remove the online state in redis hashes (HINCRBY)
	return nil
}

// AddToken implements the Channel AddToken method.
func (c *RedisChannel) AddToken(token string, key string) error {
	// store the token in redis sets (SADD)
	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *RedisChannel) AuthToken(token string) error {
	// remove the token from redis sets (SREM)
	return nil
}

// SetDeadline implements the Channel SetDeadline method.
func (c *RedisChannel) SetDeadline(d int64) {
	c.expire = d
}

// Timeout implements the Channel Timeout method.
func (c *RedisChannel) Timeout() bool {
	return time.Now().UnixNano() > c.expire
}

// Close implements the Channel Close method.
func (c *RedisChannel) Close() error {
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
