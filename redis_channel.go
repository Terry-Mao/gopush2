package main

import (
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush2/hash"
	"github.com/garyburd/redigo/redis"
	"net"
	"sync"
	"time"
)

var (
	ConfigRedisErr = errors.New("redis config not set")
	RedisNoConnErr = errors.New("can't get a redis conn")
	RedisDataErr   = errors.New("redis data fatal error")
	redisPool      = map[string]*redis.Pool{}
	redisHash      *hash.Ketama

	msgRedisPre    = "m_"
	onlineRedisPre = "o_"
	tokenRedisPre  = "t_"

	defaultRedisNode = "node1"
)

type RedisChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]int64
	// Channel expired unixnano
	expire int64
}

// Init redis channel, such as init redis pool, init consistent hash ring
func InitRedisChannel() error {
	if Conf.Redis == nil || len(Conf.Redis) == 0 {
		return ConfigRedisErr
	}

	// redis pool
	for n, c := range Conf.Redis {
		redisPool[n] = &redis.Pool{
			MaxIdle:     c.Idle,
			MaxActive:   c.Active,
			IdleTimeout: time.Duration(c.Timeout) * time.Second,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial(c.Network, c.Addr)
				if err != nil {
					Log.Printf("redis.Dial(\"%s\", \"%s\") failed (%s)", c.Network, c.Addr, err.Error())
				}
				return conn, err
			},
		}
	}

	// consistent hashing
	redisHash = hash.NewKetama(len(redisPool), 255)

	return nil
}

// New a redis message stored channel
func NewRedisChannel() *RedisChannel {
	c := &RedisChannel{}
	c.mutex = &sync.Mutex{}
	c.conn = map[net.Conn]int64{}
	c.expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second

	return c
}

// PushMsg implements the Channel PushMsg method.
func (c *RedisChannel) PushMsg(m *Message, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// send message to each conn when message id > conn last message id
	for conn, mid := range c.conn {
		// ignore message cause it's id less than mid
		if mid >= m.MsgID {
			Log.Printf("device %s: ignore send message : %d, the last message id : %d", key, m.MsgID, mid)
			continue
		}

		if err := m.Write(conn, key); err != nil {
			continue
		}

		// if succeed, update the last message id, conn.Write may failed but err == nil(client shutdown or sth else), but the message won't loss till next connect to sub
		c.conn[conn] = m.MsgID
		Log.Printf("push message \"%s\":%d to device %s", m.Msg, m.MsgID, key)
	}

	return nil
}

// SendMsg implements the Channel SendMsg method.
func (c *RedisChannel) SendMsg(conn net.Conn, mid int64, key string) error {
	// get offline message from redis which greate mid (ZRANGEBYSCORE)
	// delete the expired message
	// update the last message id for conn
	nmid := mid
	rc := getRedisConn(key)
	if rc == nil {
		return RedisNoConnErr
	}

	defer rc.Close()
	midStr := fmt.Sprintf("(%d", mid)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check exceed the maxsubscribers
	if len(c.conn)+1 > Conf.MaxSubscriberPerKey {
		return MaxConnErr
	}

	Log.Printf("add conn for device %s", key)
	// save the last push message id
	c.conn[conn] = mid
	reply, err := rc.Do("ZRANGEBYSCORE", msgRedisPre+key, midStr, -1)
	if err != nil {
		delete(c.conn, conn)
		Log.Printf("redis(\"ZRANGEBYSCORE\", \"%s\", \"%s\", 1) failed (%s)", msgRedisPre+key, midStr, err.Error())
		return err
	}

	msgs, err := redis.Strings(reply, nil)
	if err != nil {
		delete(c.conn, conn)
		Log.Printf("redis.Strings() failed (%s)", err.Error())
		return err
	}

	for _, msg := range msgs {
		m, err := NewJsonStrMessage(msg)
		if err != nil {
			// drop the message, can't unmarshal
			_, err := rc.Do("HDEL", msgRedisPre+key, m.MsgID)
			if err != nil {
				Log.Printf("redis(\"HDEL\", \"%s\", %d) failed (%s)", msgRedisPre+key, m.MsgID, err.Error())
			}

			Log.Printf("device %s: can't unmarshal message %s (%s)", key, msg, err.Error())
			continue
		}

		if m.Expired() {
			// drop the message, expired
			_, err := rc.Do("HDEL", msgRedisPre+key, m.MsgID)
			if err != nil {
				Log.Printf("redis(\"HDEL\", \"%s\", %d) failed (%s)", msgRedisPre+key, m.MsgID, err.Error())
			}
			Log.Printf("device %s: message %d expired", key, m.MsgID)
			continue
		}

		if err := m.Write(conn, key); err != nil {
			delete(c.conn, conn)
			return err
		}

		nmid = m.MsgID
		Log.Printf("push message \"%s\":%d to device %s", m.Msg, m.MsgID, key)
	}

	c.conn[conn] = nmid

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *RedisChannel) AddConn(conn net.Conn, mid int64, key string) error {
	// store the online state in redis hashes (HINCRBY)
	rc := getRedisConn(key)
	if rc == nil {
		// drop the connection from map, RemoveConn won't call if err
		c.mutex.Lock()
		delete(c.conn, conn)
		c.mutex.Unlock()
		return RedisNoConnErr
	}

	defer rc.Close()
	_, err := rc.Do("HINCRBY", onlineRedisPre+key, Conf.Node, 1)
	if err != nil {
		// drop the connection from map, RemoveConn won't call if err
		c.mutex.Lock()
		delete(c.conn, conn)
		c.mutex.Unlock()
		Log.Printf("redis(\"HINCRBY\", \"%s\", \"%s\", 1) failed (%s)", onlineRedisPre+key, Conf.Node, err.Error())
		return err
	}

	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *RedisChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	Log.Printf("remove conn for device %s", key)
	delete(c.conn, conn)
	c.mutex.Unlock()

	// remove the online state in redis hashes (HINCRBY)
	rc := getRedisConn(key)
	if rc == nil {
		return RedisNoConnErr
	}

	defer rc.Close()
	_, err := rc.Do("HINCRBY", onlineRedisPre+key, Conf.Node, -1)
	if err != nil {
		Log.Printf("redis(\"HINCRBY\", \"%s\", \"%s\", -1) failed (%s)", onlineRedisPre+key, Conf.Node, err.Error())
		return err
	}

	return nil
}

// AddToken implements the Channel AddToken method.
func (c *RedisChannel) AddToken(token string, key string) error {
	// store the token in redis sets (SADD)
	conn := getRedisConn(key)
	if conn == nil {
		return RedisNoConnErr
	}

	defer conn.Close()
	reply, err := conn.Do("SADD", tokenRedisPre+key, token)
	if err != nil {
		Log.Printf("redis(\"SADD\", \"%s\", \"%s\") failed (%s)", tokenRedisPre+key, token, err.Error())
		return err
	}

	r, err := redis.Int(reply, nil)
	if err != nil {
		Log.Printf("redis.Int() failed (%s)", err.Error())
		return err
	}

	if r == 0 {
		Log.Printf("device %s: token %salready exists", key, token)
		return TokenExistErr
	}

	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *RedisChannel) AuthToken(token string, key string) error {
	// remove the token from redis sets (SREM)
	conn := getRedisConn(key)
	defer conn.Close()
	reply, err := conn.Do("SREM", tokenRedisPre+key, token)
	if err != nil {
		Log.Printf("c.Do(\"SREM\", \"%s\", \"%s\") failed (%s)", tokenRedisPre+key, token, err.Error())
		return err
	}

	r, err := redis.Int(reply, nil)
	if err != nil {
		Log.Printf("redis.Int() failed (%s)", err.Error())
		return err
	}

	if r == 0 {
		Log.Printf("device %s: token %s not exist, auth failed", key, token)
		return AuthTokenErr
	}

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for conn, _ := range c.conn {
		if err := conn.Close(); err != nil {
			// ignore close error
			Log.Printf("conn.Close() failed (%s)", err.Error())
		}
	}

	return nil
}

func getRedisConn(key string) redis.Conn {
	node := defaultRedisNode
	// if multiple redispool use ketama
	if len(redisPool) != 1 {
		node = redisHash.Node(key)
	}

	p, ok := redisPool[node]
	if !ok {
		return nil
	}

	return p.Get()
}
