package main

import (
	"flag"
	"github.com/Terry-Mao/gopush2/cfg"
	"log"
	"os"
	"runtime"
	"runtime/debug"
)

var (
	Log  *log.Logger
	Conf *cfg.Config
)

func init() {
	Log = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
}

func main() {
	defer recoverFunc()

	var err error
	// parse cmd-line arguments
	flag.Parse()
	// init config
	Conf, err = cfg.New(cfg.ConfFile)
	if err != nil {
		Log.Printf("cfg.New(\"%s\") failed (%s)", cfg.ConfFile, err.Error())
		return
	}

	// init log
	if Conf.Log != "" {
		f, err := os.OpenFile(Conf.Log, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			Log.Printf("os.OpenFile(\"%s\", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644) failed (%s)", Conf.Log, err.Error())
			return
		}

		defer f.Close()
		Log = log.New(f, "", log.LstdFlags|log.Lshortfile)
	}

	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProcs)
	// create channel
	channel = NewChannelList()
	Log.Printf("gopush2 service start.")
	// start stats
	StartStats()
	if Conf.Addr == Conf.AdminAddr {
		Log.Printf("Admin addr must not same with addr for security reason")
		os.Exit(-1)
	}

	// start admin http
	go func() {
		if err := StartAdminHttp(); err != nil {
			Log.Printf("StartAdminHttp() failed (%s)", err.Error())
			os.Exit(-1)
		}
	}()

	if Conf.Protocol == WebsocketProtocol {
		// Start http push service
		if err = StartHttp(); err != nil {
			Log.Printf("StartHttp() failed (%s)", err.Error())
		}
	} else if Conf.Protocol == TCPProtocol {
		// Start http push service
		if err = StartTCP(); err != nil {
			Log.Printf("StartTCP() failed (%s)", err.Error())
		}
	} else {
		Log.Printf("unknown configuration protocol: %d", Conf.Protocol)
	}

	// exit
	Log.Printf("gopush2 service stop.")
}

func recoverFunc() {
	if err := recover(); err != nil {
		Log.Printf("Error : %v, Debug : \n%s", err, string(debug.Stack()))
	}
}
