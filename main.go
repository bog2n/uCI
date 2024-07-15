package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"uci/pkg"
)

var ConfigFile string
var config pkg.Config

func init() {
	flag.Usage = pkg.Usage
	flag.StringVar(&ConfigFile, "c", "config.toml", "Config file")
	flag.Parse()

	if os.Getenv("DEV") != "" {
		log.Default().SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			<-c
			log.Print("Received SIGHUP, reloading config")
			err := config.Reload(ConfigFile)
			if err != nil {
				log.Print("Error reloading config: ", err)
			}
		}
	}()
	err := config.Reload(ConfigFile)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/uci", config.MainHandler)
	s := ""
	if config.TLS {
		s = " with TLS enabled"
	}
	log.Printf("Listening on %s%s", config.Address, s)
	if config.TLS {
		log.Fatal(http.ListenAndServeTLS(config.Address, config.CertFile, config.KeyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(config.Address, nil))
	}
}
