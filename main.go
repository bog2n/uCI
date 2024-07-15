package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"uci/pkg"
)

var (
	ConfigFile string
	Signal     string
	config     pkg.Config
)

func init() {
	flag.Usage = pkg.Usage
	flag.StringVar(&ConfigFile, "c", "config.toml", "Config file")
	flag.StringVar(&Signal, "s", "", "signal to send to process: reload, stop")
	flag.Parse()

	if os.Getenv("DEV") != "" {
		log.Default().SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for {
			switch <-c {
			case syscall.SIGHUP:
				log.Print("Received SIGHUP, reloading config")
				err := config.Reload(ConfigFile)
				if err != nil {
					log.Print("Error reloading config: ", err)
				}
			case syscall.SIGTERM:
				// TODO probably should add context stuff here
				os.Remove(config.PidFile)
				os.Exit(0)
			}
		}
	}()

	err := config.Reload(ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	if Signal != "" {
		pidstring, err := os.ReadFile(config.PidFile)
		if err != nil {
			log.Fatal(err)
		}
		pid, err := strconv.Atoi(string(pidstring))
		switch Signal {
		case "reload":
			proc, err := os.FindProcess(pid)
			if err != nil {
				log.Fatal(err)
			}
			proc.Signal(syscall.SIGHUP)
		case "stop":
			proc, err := os.FindProcess(pid)
			if err != nil {
				log.Fatal(err)
			}
			proc.Signal(syscall.SIGTERM)
		}
		os.Exit(0)
	}

	pid, err := os.Create(config.PidFile)
	if err != nil {
		log.Fatal(err)
	}
	defer pid.Close()
	pid.WriteString(strconv.Itoa(os.Getpid()))
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
