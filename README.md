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

---------------------------------------


## Features
 * Lightweight
 * Native Go implemention
 * Lazy channel expiration
 * Lazy single message expiration
 * Store the offline message
 * Multiple subscribers
 * Restrict the subscribers per key
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

# 1. open http://localhost:8080/client in browser and press the Send button (modify the gopush2.conf, set debug to 1)
# 2. you can use curl
$ curl -d "test" http://localhost:8080/pub?key=Terry-Mao\&expire=30
# then your browser will alert the "message"
# open http://localhost:8080/stat?type=memory in browser to get memstats(type: memory, server, channel, subscriber, golang, config)
```
a simple javascript examples
```javascript
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
  "debug": 1 # use test client, http://xx:xx/client (1: open, 0: close)
}
```

## Protocol
 1. Sub the specified key "ws://localhost:port/sub?key=xxx&msg_id=$msg_id" use websocket (the $msg_id is stored is client, every time publish a message will return the msg_id, if client's $msg_id is nil then use 0)
 2. the subscriber then block, till a message published to the sub key
 3. post to http://localhost:port/pub?key=xxx&expire=30, the message write to http body (the url query field "expire" means message expired after 30 second)
 4. if any error, gopush2 close the socket, client need to retry connect

Response Json:
### msg
* the publish message

### msg_id
* the publish message id

the reponse json examples:
```json
{
    "msg" : "hello, world",
    "msg_id" : 1
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
