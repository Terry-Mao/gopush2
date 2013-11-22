package cfg

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"
)

var (
	ConfFile string
)

func init() {
	flag.StringVar(&ConfFile, "c", "./gopush2.conf", " set gopush2 config file path")
}

type RedisConfig struct {
	Network string `json:"network"`
	Addr    string `json:"addr"`
	Timeout int    `json:"timeout"`
	Pool    int    `json:"pool"`
}

type Config struct {
	Node                string                  `json:"node"`
	Addr                string                  `json:"addr"`
	AdminAddr           string                  `json:"admin_addr"`
	Log                 string                  `json:"log"`
	MessageExpireSec    int64                   `json:"message_expire_sec"`
	ChannelExpireSec    int64                   `json:"channel_expire_sec"`
	MaxStoredMessage    int                     `json:"max_stored_message"`
	MaxProcs            int                     `json:"max_procs"`
	MaxSubscriberPerKey int                     `json:"max_subscriber_per_key"`
	TCPKeepAlive        int                     `json:"tcp_keepalive"`
	ChannelBucket       int                     `json:"channel_bucket"`
	ChannelType         int                     `json:"channel_type"`
	HeartbeatSec        int                     `json:"heartbeat_sec"`
	Auth                int                     `json:"auth"`
	Redis               map[string]*RedisConfig `json:"redis"`
	Debug               int                     `json:"debug"`
}

// get a config
func New(file string) (*Config, error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("ioutil.ReadFile(\"%s\") failed (%s)", file, err.Error())
		return nil, err
	}

	cf := &Config{
		Node:      "gopush2-1",
		Addr:      "localhost",
		AdminAddr: "localhost",
		//Pprof:               1,
		MessageExpireSec:    10800,  // 3 hour
		ChannelExpireSec:    604800, // 24 * 7 hour
		Log:                 "./gopush.log",
		MaxStoredMessage:    20,
		MaxSubscriberPerKey: 0, // no limit
		MaxProcs:            runtime.NumCPU(),
		TCPKeepAlive:        1,
		ChannelBucket:       16,
		ChannelType:         0,
		HeartbeatSec:        30,
		Auth:                1,
		Redis:               nil,
		Debug:               0,
	}

	if err = json.Unmarshal(c, cf); err != nil {
		fmt.Printf("json.Unmarshal(\"%s\", cf) failed (%s)", string(c), err.Error())
		return nil, err
	}

	return cf, nil
}
