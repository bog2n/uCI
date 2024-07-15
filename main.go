package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"uci/pkg"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

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
		repoKey := p.Repo.Name + "^" + b
		if conf, ok := config.Repos[repoKey]; ok {
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
			err := deploy(conf, p.Repo.URL)
			if err != nil {
				log.Print(err)
				errno := http.StatusInternalServerError
				http.Error(w, http.StatusText(errno), errno)
			}
		} else {
			log.Print("Repository key: ", repoKey, " not found")
			http.NotFound(w, r)
		}
	} else {
		log.Print("branch name not found in payload")
		errno := http.StatusInternalServerError
		http.Error(w, http.StatusText(errno), errno)
	}
}

func deploy(conf pkg.RepoConfig, URL string) error {
	log.Print(URL)
	urlinfo, err := url.Parse(URL)
	if err != nil {
		return err
	}
	user := urlinfo.User.Username()
	key, err := os.ReadFile(conf.SshPrivKey)
	if err != nil {
		return err
	}
	sshAuth, err := ssh.NewPublicKeys(user, key, "")

	r, err := git.PlainOpen(conf.Path)
	if err != nil && err != git.ErrRepositoryNotExists {
		return err
	} else if err == git.ErrRepositoryNotExists {
		_, err = git.PlainClone(conf.Path, false, &git.CloneOptions{
			URL:           URL,
			Auth:          sshAuth,
			Progress:      os.Stdout,
			ReferenceName: plumbing.NewBranchReferenceName(conf.Branch),
		})
		if err != nil {
			return err
		}
	} else {
		remote, err := r.Remote("origin")
		if err != nil {
			return err
		}
		if len(remote.Config().URLs) > 0 && remote.Config().URLs[0] != URL {
			log.Print("Wrong repo")
			return err
		}
		w, err := r.Worktree()
		if err != nil {
			return err
		}
		err = w.Pull(&git.PullOptions{
			Auth:     sshAuth,
			Progress: os.Stdout,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}
	}
	if len(conf.Cmd) <= 0 {
		return err
	}
	cmd := exec.Command(conf.Cmd[0], conf.Cmd[1:]...)
	cmd.Dir = conf.Path
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

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
			config.Reload(ConfigFile)
		}
	}()
	config.Reload(ConfigFile)
}

func main() {
	http.HandleFunc("/uci", mainHandler)
	if config.TLS {
		log.Fatal(http.ListenAndServeTLS(config.Address, config.CertFile, config.KeyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(config.Address, nil))
	}
}
