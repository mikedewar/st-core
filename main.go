package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/nytlabs/st-core/server"
)

var (
	port = flag.String("port", "7071", "streamtools port")
)

func main() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-sigc
		log.Println(s)
		os.Exit(0)
	}()

	flag.Parse()

	// Unpack settings file, or create a new one if necessary
	var settings server.Settings

	dir, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	fname := dir + "/.st.json"
	d, err := ioutil.ReadFile(fname)
	if err != nil {
		if os.IsNotExist(err) {
			// make a fresh settings file
			log.Println("creating new settings file at", fname)
			newSettings := server.NewSettings()
			d, err = json.Marshal(newSettings)
			err = ioutil.WriteFile(fname, d, 0644)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}
	err = json.Unmarshal(d, &settings)
	if err != nil {
		log.Fatal(err)
	}

	s := server.NewServer(settings)
	r := s.NewRouter()

	http.Handle("/", r)

	log.Println("serving on", *port)
	err = http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Panicf(err.Error())
	}
}
