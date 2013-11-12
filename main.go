package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	"github.com/Terry-Mao/gopush2/cfg"
	"log"
	"net/http"
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

	// start stats
	StartStats()
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

	// set sub handler
	http.Handle("/sub", websocket.Handler(Subscribe))
	if Conf.Debug == 1 {
		http.HandleFunc("/client", Client)
	}

	// create channel
	channel = NewChannel()
	Log.Printf("gopush2 service start.")

	// admin
	if Conf.AdminAddr != Conf.Addr || Conf.AdminPort != Conf.Port {
		go func() {
			adminServeMux := http.NewServeMux()
			// publish
			adminServeMux.HandleFunc("/pub", Publish)
			// stat
			adminServeMux.HandleFunc("/stat", Stat)
			err := http.ListenAndServe(fmt.Sprintf("%s:%d", Conf.AdminAddr, Conf.AdminPort), adminServeMux)
			if err != nil {
				panic(err)
			}
		}()
	} else {
		http.HandleFunc("/pub", Publish)
		http.HandleFunc("/stat", Stat)
	}

	// start listen and pending here
	if err = Listen(Conf.Addr, Conf.Port); err != nil {
		Log.Printf("Listen() failed (%s)", err.Error())
		return
	}

	Log.Printf("gopush2 service stop.")
}
