## Terry-Mao/gopush2
`Terry-Mao/gopush2` is an push server written by golang. (websocket)

---------------------------------------
  * [Features](#features)
  * [Requirements](#requirements)
  * [Installation](#installation)
  * [Usage](#usage)
  * [Configuration](#configuration)
  * [Protocol](#protocol)
  * [Development](#development)
  * [Documention](#documentation)
  * [TODO](#todo)

---------------------------------------


## Features
 * Lightweight
 * High performance
 * Native Go implemention
 * Lazy channel expiration
 * Lazy single message expiration
 * Store the offline message
 * Multiple subscribers
 * Restrict the subscribers per key
 * Heartbeat support
 * Token authentication
 * Stats (Memory, Channel, Subscriber, Golang, Server)

## Requirements
 * Go 1.1 or higher
 * Golang websocket

```sh
# websocket
$ go get code.google.com/p/go.net/websocket 
```

## Installation
Just pull `Terry-Mao/gopush2` from github using `go get`:

```sh
$ go get -u github.com/Terry-Mao/gopush2
```

## Usage
```sh
# start the gopush2 server
$ nohup ./gopush2 -c ./gopush2.conf 2>&1 >> ./panic.log &

# create a channel and init a token
$ curl http://localhost:8080/ch?key=Terry-Mao\&token=test
# open http://localhost:8080/client in browser (modify the gopush2.conf, set debug to 1)
# you can use curl
$ curl -d "test" http://localhost:8080/pub?key=Terry-Mao\&expire=30
# then your browser will alert the "message"
# open http://localhost:8080/stat?type=memory in browser to get memstats(type: memory, server, channel, subscriber, golang, config)
```
a simple javascript examples
```javascript
    <script type="text/javascript" src="http://img3.douban.com/js/packed_jquery.min6301986802.js" async="true"></script>
    <script type="text/javascript">
        var sock = null;
        var wsuri = "ws://%s:%d/sub?key=Terry-Mao&mid=0&token=test";

        window.onload = function() {
            try
            {
                sock = new WebSocket(wsuri);
            }catch (e) {
                alert(e.Message);
            }

            sock.onopen = function() {
                alert("connected to " + wsuri);
            }

            sock.onerror = function(e) {
                alert(" error from connect " + e.Message);
            }

            sock.onclose = function(e) {
                alert("connection closed (" + e.code + ")");
            }

            sock.onmessage = function(e) {
                if(e.data != "") {
                    alert("message received: " + e.data);
                }
            }

        };

        setInterval("sock.send('')", 3000);
</script>
```

## Configuration
```python
{
  "addr": "127.0.0.1", # the subscriber listen address
  "port": 8080, # the subscriber listen port
  "admin_addr": "127.0.0.1", # for the publisher and stat listen address
  "admin_port": 8080, # for the publisher and stat listen port
  "log": "/tmp/gopush.log", # the gopush2 log file path
  "message_expire_sec": 7200, # the default message expire second
  "channel_expire_sec": 60, # the default channel expire second
  "max_stored_message": 20, # the max stored message for a subscriber
  "max_procs": 4, # the service used cpu cores
  "max_subscriber_per_key": 1, # the max subscriber per key
  "tcp_keepalive": 1, # use SO_KEEPALIVE, for lost tcp connection fast detection (1: open, 0: close)
  "channel_bucket": 16, # the channel inner hashmap number, default 16
  "channel_type": 1, # the channel type (1: in-process store message, 2: redis store message)
  "heartbeat_sec": 30, # the server receive heartbeat time second (client send heartbeat to server, then reply to client)
  "auth": 1, # the channel need auth (1: yes, 0: no)
  "debug": 1 # use test client, http://xx:xx/client (1: open, 0: close)
}
```

## Protocol
```python
 # 1
 # Create a channle for the subscriber http://localhost:port/ch?key=xxx&token=xxx (when config file set auth = 1)
 # You can implement your own auth logical in your appserver, then create a channle and add a token for the subscriber then reply a token to the client

 # 2
 # Sub the specified key "ws://localhost:port/sub?key=xxx&mid=$mid&token=xxx" use websocket 
 # the $mid is stored is client, every time publish a message will return the mid, if client's $mid is nil then use 0
 # token is received by your own appserver (when config file set auth = 1 else token is not needed)

 # 3
 # The subscriber then block, until a message published to the sub key or receive a server heartbeat
 
 # 4
 # Post to http://localhost:port/pub?key=xxx&expire=30, the message write to http body (the url query field "expire" means message expired after 30 second)

 # 5
 # If any error, gopush2 close the socket, client need to retry connect

 # 6
 # Client send heartbeat and receive heartbeat
```

```python
# Subscriber received response json
{
    "msg" : "hello, world", # published message content
    "mid" : 1 # published message associated id
}

# Publisher received response json
{
    "ret" : 0, # return code (0: succeed, 65535: internal error)
    "msg" : "ok" # return message ("ok" returned if succeed, or detail error message)
}
```

## Development
`Terry-Mao/gopush2` is not feature-complete yet. Your help is very appreciated.
If you want to contribute, you can work on an [open issue](https://github.com/Terry-Mao/gopush2/issues?state=open) or review a [pull request](https://github.com/Terry-Mao/gopush2/pulls).

## Documentation
Read the `Terry-Mao/gopush2` documentation from a terminal
```sh
$ go doc github.com/Terry-Mao/gopush2
```
Alternatively, you can [gopush2](http://go.pkgdoc.org/github.com/Terry-Mao/gopush2).

## TODO
  * Use redis to store the message
  * Add tcp support
  * Add more test
  * Add test case and examples

