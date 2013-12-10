package main

import (
	"errors"
	"runtime/debug"
)

const (
	// internal failed
	retInternalErr = 65535
	// param error
	retParamErr = 65534
	// ok
	retOK = 0
	// create channel failed
	retCreateChannel = 1
	// add channel failed
	retAddChannle = 2
	// get channel failed
	retGetChannel = 3
	// add token failed
	retAddToken = 4
	// message push failed
	retPushMsg = 5
)

const (
	WebsocketProtocol = 0
	TCPProtocol       = 1
)

var (
	// Exceed the max subscriber per key
	MaxConnErr = errors.New("Exceed the max subscriber connection per key")
	// Assection type failed
	AssertTypeErr = errors.New("Subscriber assert type failed")
	// Auth token failed
	AuthTokenErr = errors.New("Auth token failed")
	// Token exists
	TokenExistErr = errors.New("Token already exist")
)

var (
	heartbeatMsg   = ""
	heartbeatBytes = []byte(heartbeatMsg)
)

func recoverFunc() {
	if err := recover(); err != nil {
		Log.Printf("Error : %v, Debug : \n%s", err, string(debug.Stack()))
	}
}
