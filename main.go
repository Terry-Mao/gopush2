package main

import (
	"flag"
	"github.com/Terry-Mao/gopush2/cfg"
	"log"
	"os"
	"runtime"
)

var (
	Log  *log.Logger
	Conf *cfg.Config
)

func init() {
	Log = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
}

func main() {
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
	// Start http push service
	if err = StartHttp(); err != nil {
		Log.Printf("StartHttp failed (%s)", err.Error())
	}

	// exit
	Log.Printf("gopush2 service stop.")
}
