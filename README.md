## Terry-Mao/gopush2
`Terry-Mao/gopush2` is an push server written by golang. (websocket)

---------------------------------------
  * [Features](#features)
  * [Requirements](#requirements)
  * [Installation](#installation)
  * [Usage](#usage)
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
 * Performance Profile

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
$ ./gopush2 -c ./gopush2.conf

# 1. open http://localhost:8080/client in browser and press the Send button (modify the gopush2.conf, set debug to 1)
# 2. you can use curl
$ curl -d "test" http://localhost:8080/pub?key=Terry-Mao\&expire=30
# then your browser will alert the "message"
```
a simple javascript examples
```javascript
```

## Protocol

response json field:
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
