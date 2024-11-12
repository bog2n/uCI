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
	logfile    *os.File
	cfg        pkg.Config
)

func init() {
	flag.Usage = pkg.Usage
	flag.StringVar(&ConfigFile, "c", "config.toml", "Config file")
	flag.StringVar(&Signal, "s", "", "signal to send to process: reload, stop")
	flag.Parse()

	err := cfg.Reload(ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for {
			switch <-c {
			case syscall.SIGHUP:
				log.Print("Received SIGHUP, reloading config")
				err := cfg.Reload(ConfigFile)
				if err != nil {
					log.Print("Error reloading config: ", err)
				}
			case syscall.SIGTERM:
				// TODO probably should add context stuff here
				log.Print("Received SIGTERM, exiting...")
				logfile.Close()
				os.Remove(cfg.PidFile)
				os.Exit(0)
			}
		}
	}()

	if Signal != "" {
		pidstring, err := os.ReadFile(cfg.PidFile)
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

	pid, err := os.Create(cfg.PidFile)
	if err != nil {
		log.Fatal(err)
	}
	defer pid.Close()
	pid.WriteString(strconv.Itoa(os.Getpid()) + "\n")
}

func main() {
	http.HandleFunc("/", cfg.BasicAuth(cfg.MainHandler))
	http.HandleFunc("/repo/{name}", cfg.BasicAuth(cfg.RepoHandler))
	http.HandleFunc("/logs/{id}", cfg.BasicAuth(cfg.LogsHandler))

	http.HandleFunc("/uci", cfg.CIHandler)
	http.Handle("/static/", http.FileServer(pkg.StaticFS))
	s := ""
	if cfg.TLS {
		s = " with TLS enabled"
	}
	log.Printf("Listening on %s%s", cfg.Address, s)
	if cfg.TLS {
		log.Fatal(http.ListenAndServeTLS(cfg.Address, cfg.CertFile, cfg.KeyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(cfg.Address, nil))
	}
}
