package main

/*
import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type RedisSubscriber struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]bool
	// Auth token
	Token string
	// Subscriber expired unixnano
	Expire int64
	// Max message stored number
	MaxMessage int
	// Subscriber key
	Key string
}

// New a redis message stored subscriber
func NewRedisSubscriber(key string) *RedisSubscriber {
	s := &RedisSubscriber{}
	s.mutex = &sync.Mutex{}
	s.conn = map[net.Conn]bool{}
	s.Expire = time.Now().UnixNano() + Conf.ChannelExpireSec*Second
	s.MaxMessage = Conf.MaxStoredMessage
	s.Key = key

	subscriberStats.IncrCreated()
	return s
}

// AddConn implements the Subscriber AddConn method.
func (s *RedisSubscriber) AddConn(conn net.Conn) error {
	return addConn(s.mutex, s.conn, conn, s.Key)
}

// RemoveConn implements the Subscriber RemoveConn method.
func (s *RedisSubscriber) RemoveConn(conn net.Conn) {
	removeConn(s.mutex, s.conn, conn, s.Key)
}

// Auth implements the Subscriber Auth method.
func (s *RedisSubscriber) Auth(token string) error {
	if token != s.Token {
		return ErrAuthToken
	}

	return nil
}

*/
