package main

import (
	"sync"
	"time"
)

const (
	Second = int64(time.Second)
)

type Channel struct {
	did   map[string]*Subscriber
	mutex *sync.Mutex
}

func NewChannel() *Channel {
	c := &Channel{}
	c.did = map[string]*Subscriber{}
	c.mutex = &sync.Mutex{}

	return c
}

// get a subscriber from channel in pub/sub action
func (c *Channel) GetSubscriber(key string) *Subscriber {
	var (
		s   *Subscriber
		ok  bool
		now = time.Now().UnixNano()
	)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if s, ok = c.did[key]; !ok {
		// not exists subscriber for the key
		s = NewSubscriber()
		c.did[key] = s
		s.Key = key
	} else {
		// check expired
		if now >= s.Expire {
			// let gc free the old subscriber use a new one
			// remove old sub conn
			Log.Printf("device %s drop the expired channel, refresh a new one now(%d) > expire(%d)", key, now, s.Expire)
			s.CloseAllConn()
			s = NewSubscriber()
			s.Key = key
			c.did[key] = s
		} else {
			// refresh the expire time
			s.Expire = now + Conf.ChannelExpireSec*Second
		}
	}

	return s
}
