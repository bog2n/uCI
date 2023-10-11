package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type CIConfig struct {
	Repos    []RepoConfig `toml:"repo"`
	TLS      bool         `toml:"TLS"`
	CertFile string       `toml:"certfile"`
	KeyFile  string       `toml:"keyfile"`
	Address  string       `toml:"address"`
}

type RepoConfig struct {
	SshPrivKey string   `toml:"keyfile"`
	Name       string   `toml:"name"`
	Path       string   `toml:"path"`
	Cmd        []string `toml:"cmd"`
	Branch     string   `toml:"branch"`
	Provider   string   `toml:"prov"`
	Auth       string   `toml:"auth"`
	SshAuth    *ssh.PublicKeys
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
		return
	}

	var p struct {
		Ref  string `json:"ref"`
		Repo struct {
			Name string `json:"full_name"`
			URL  string `json:"ssh_url"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Print(err)
		return
	}

	if b, ok := strings.CutPrefix(p.Ref, "refs/heads/"); ok {
		key := repoKey{
			name:   p.Repo.Name,
			branch: b,
		}
		if conf, ok := repos[key]; ok {
			switch conf.Provider {
			case "gitea":
				log.Print("Gitea provider")
				if r.Header.Get("Authorization") != conf.Auth {
					errno := http.StatusUnauthorized
					http.Error(w, http.StatusText(errno), errno)
					log.Print("Unauthorized")
					return
				}
			case "github":
				log.Print("Github provider")
				shasum := r.Header.Get("x-hub-signature-256")
				h := hmac.New(sha256.New, []byte(conf.Auth))
				h.Write(payload)
				payloadsum := "sha256=" + hex.EncodeToString(h.Sum(nil))
				if payloadsum != shasum {
					errno := http.StatusUnauthorized
					http.Error(w, http.StatusText(errno), errno)
					log.Print("Unauthorized")
					return
				}
			default:
				log.Print("Unknown provider")
				return
			}
			go deploy(conf, p.Repo.URL)
		}
	} else {
		log.Print("branch name not found in payload")
		errno := http.StatusInternalServerError
		http.Error(w, http.StatusText(errno), errno)
	}
}

func deploy(conf RepoConfig, URL string) {
	r, err := git.PlainOpen(conf.Path)
	if err != nil && err != git.ErrRepositoryNotExists {
		log.Print(err)
		return
	} else if err == git.ErrRepositoryNotExists {
		log.Print(URL)
		_, err = git.PlainClone(conf.Path, false, &git.CloneOptions{
			URL:           URL,
			Auth:          conf.SshAuth,
			Progress:      os.Stdout,
			ReferenceName: plumbing.NewBranchReferenceName(conf.Branch),
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
			Auth:     conf.SshAuth,
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

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(flag.CommandLine.Output(), `
Config file format:

address  = "<bind address>"
TLS      = true/false
keyfile  = "<tls private key>"
certfile = "<tls certificate>"

[[repo]]
	name     = "<gitea repo name>"
	branch   = "<git branch>"
	keyfile  = "<ssh private key file>"
	path     = "<path to repo>"
	cmd      = "<build command>"
	prov     = "<git provider>"
	auth     = "<auth token>"
...

Valid providers are: github, gitea

You might want to specify SSH_KNOWN_HOSTS environment variable for ssh to work

`)
}

type repoKey struct {
	name   string
	branch string
}

var ConfigFile string
var config CIConfig
var repos map[repoKey]RepoConfig

func reload() {
	file, err := os.ReadFile(ConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	_, err = toml.Decode(string(file), &config)
	if err != nil {
		log.Fatal(err)
	}
	repos = make(map[repoKey]RepoConfig)
	for _, v := range config.Repos {
		key, err := os.ReadFile(v.SshPrivKey)
		if err != nil {
			log.Fatal(err)
		}
		sshAuth, err := ssh.NewPublicKeys("git", key, "")
		if err != nil {
			log.Fatal(err)
		}
		v.SshAuth = sshAuth
		repos[repoKey{
			name:   v.Name,
			branch: v.Branch,
		}] = v
	}
}

func init() {
	flag.Usage = Usage
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
			reload()
		}
	}()
	reload()
}

func main() {
	http.HandleFunc("/uci", mainHandler)
	if config.TLS {
		log.Fatal(http.ListenAndServeTLS(config.Address, config.CertFile, config.KeyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(config.Address, nil))
	}
}
