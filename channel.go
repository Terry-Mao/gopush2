package main

import (
	"github.com/Terry-Mao/gopush2/mmhash"
	"sync"
	"time"
)

const (
	bucketSize = 16
)

type channelBucket struct {
	data  map[string]*Subscriber
	mutex *sync.Mutex
}

type Channel struct {
	subscriber []*channelBucket
}

var (
	channel *Channel
)

func NewChannel() *Channel {
	c := &Channel{}
	c.subscriber = []*channelBucket{}
	// split hashmap to 16 bucket
	for i := 0; i < bucketSize; i++ {
		b := &channelBucket{
			data:  map[string]*Subscriber{},
			mutex: &sync.Mutex{},
		}

		c.subscriber = append(c.subscriber, b)
	}

	return c
}

// get a bucket from channel
func (c *Channel) bucket(key string) *channelBucket {
	idx := mmhash.MurMurHash2(key) & (bucketSize - 1)
	return c.subscriber[idx]
}

// get a subscriber from channel in pub/sub action
func (c *Channel) Subscriber(key string) *Subscriber {
	var (
		s   *Subscriber
		ok  bool
		now = time.Now().UnixNano()
	)

	// get a channel bucket
	b := c.bucket(key)
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if s, ok = b.data[key]; !ok {
		// not exists subscriber for the key
		s = NewSubscriber(key)
		b.data[key] = s
		channelStats.IncrCreated()
	} else {
		// check expired
		if now >= s.Expire {
			// let gc free the old subscriber use a new one
			// remove old sub conn
			Log.Printf("device %s drop the expired channel, refresh a new one now(%d) > expire(%d)", key, now, s.Expire)
			s.CloseAllConn()
			s = NewSubscriber(key)
			b.data[key] = s
			channelStats.IncrExpired()
		} else {
			// refresh the expire time
			s.Expire = now + Conf.ChannelExpireSec*Second
			channelStats.IncrRefreshed()
		}
	}

	return s
}
