package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != config.WhAuth {
		errno := http.StatusUnauthorized
		http.Error(w, http.StatusText(errno), errno)
		return
	}
	var p struct {
		Repo struct {
			Name string `json:"name"`
			URL  string `json:"ssh_url"`
		} `json:"repository"`
	}
	if err := json.Unmarshal([]byte(r.FormValue("payload")), &p); err != nil {
		log.Print(err)
		return
	}
	go deploy(p.Repo.Name, p.Repo.URL)
}

type CIConfig struct {
	Repos      []RepoConfig `toml:"repo"`
	WhAuth     string       `toml:"auth"`
	SshPrivKey string       `toml:"keyfile"`
	Address    string       `toml:"address"`
}

type RepoConfig struct {
	Name string   `toml:"name"`
	Path string   `toml:"path"`
	Cmd  []string `toml:"cmd"`
}

func deploy(repo string, URL string) {
	if conf, ok := repos[repo]; ok {
		r, err := git.PlainOpen(conf.Path)
		if err != nil && err != git.ErrRepositoryNotExists {
			log.Print(err)
			return
		} else if err == git.ErrRepositoryNotExists {
			log.Print(URL)
			_, err = git.PlainClone(conf.Path, false, &git.CloneOptions{
				URL:      URL,
				Auth:     sshAuth,
				Progress: os.Stdout,
			})
			if err != nil {
				log.Print(err)
				return
			}
		} else {
			remote, err := r.Remote("origin")
			if err != nil {
				log.Print(err)
				return
			}
			if len(remote.Config().URLs) > 0 && remote.Config().URLs[0] != URL {
				log.Print("Wrong repo")
				return
			}
			w, err := r.Worktree()
			if err != nil {
				log.Print(err)
				return
			}
			err = w.Pull(&git.PullOptions{
				Auth:     sshAuth,
				Progress: os.Stdout,
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				log.Print(err)
				return
			}
		}
		if len(conf.Cmd) <= 0 {
			log.Print("No command supplied")
			return
		}
		cmd := exec.Command(conf.Cmd[0], conf.Cmd[1:]...)
		cmd.Dir = conf.Path
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Print(err)
		}
	}
}

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(flag.CommandLine.Output(), `
Config file format:

address = "<bind address>"
auth = "<auth token>"
keyfile = "<ssh private key file>"

[[repo]]
	name = "<gitea repo name>"
	path = "<path to repo>"
	cmd = "<build command>"
...

`)
}

var ConfigFile string
var config CIConfig
var sshAuth *ssh.PublicKeys
var repos map[string]RepoConfig

func reload() {
	file, err := os.ReadFile(ConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	_, err = toml.Decode(string(file), &config)
	if err != nil {
		log.Fatal(err)
	}
	repos = make(map[string]RepoConfig)
	for _, v := range config.Repos {
		repos[v.Name] = v
	}
	key, err := os.ReadFile(config.SshPrivKey)
	if err != nil {
		log.Fatal(err)
	}
	sshAuth, err = ssh.NewPublicKeys("git", key, "")
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.Usage = Usage
	flag.StringVar(&ConfigFile, "c", "config.toml", "Config file")
	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			<-c
			log.Print("Received SIGHUP, reloading config")
			reload()
		}
	}()
	reload()
}

func main() {
	http.HandleFunc("/uci", mainHandler)
	log.Fatal(http.ListenAndServe(config.Address, nil))
}
