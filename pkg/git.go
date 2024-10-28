package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func (c *Config) CIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		errno := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(errno), errno)
		return
	}
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
		log.Print("Error decoding json payload: ", err)
		return
	}

	if b, ok := strings.CutPrefix(p.Ref, "refs/heads/"); ok {
		repoKey := p.Repo.Name + "@" + b
		if conf, ok := c.Repos[repoKey]; ok {
			shasum := r.Header.Get("x-hub-signature-256")
			h := hmac.New(sha256.New, []byte(conf.Auth))
			h.Write(payload)
			payloadsum := "sha256=" + hex.EncodeToString(h.Sum(nil))
			if subtle.ConstantTimeCompare([]byte(payloadsum), []byte(shasum)) != 1 {
				errno := http.StatusUnauthorized
				http.Error(w, http.StatusText(errno), errno)
				log.Print("Unauthorized request")
				return
			}
			logger, ready := newDeployLogger(repoKey)
			var s bool
			defer func() { ready <- s }()
			err := deploy(conf, p.Repo.URL, logger)
			if err != nil {
				s = false
				logger.Print("Error deploying: ", err)
				errno := http.StatusInternalServerError
				http.Error(w, http.StatusText(errno), errno)
			} else {
				s = true
				logger.Print("Successfully deployed!")
			}
		} else {
			log.Print("Repository key: ", repoKey, " not found")
			http.NotFound(w, r)
		}
	} else {
		log.Print("Branch not found")
		errno := http.StatusInternalServerError
		http.Error(w, http.StatusText(errno), errno)
	}
}

func deploy(conf RepoConfig, URL string, logger *log.Logger) error {
	logger.Printf("Deploying: %s", URL)
	var user string
	if s := strings.Split(strings.TrimLeft(URL, "ssh://"), "@"); len(s) == 2 {
		user = s[0]
	} else {
		return errors.New("Can't find ssh user in provided URL")
	}
	key, err := os.ReadFile(conf.SshPrivKey)
	if err != nil {
		return err
	}
	sshAuth, err := ssh.NewPublicKeys(user, key, "")
	if err != nil {
		return err
	}

	r, err := git.PlainOpen(conf.Path)
	if err != nil && err != git.ErrRepositoryNotExists {
		return err
	} else if err == git.ErrRepositoryNotExists {
		logger.Printf("Directory not found, cloning to: %s", conf.Path)
		_, err = git.PlainClone(conf.Path, false, &git.CloneOptions{
			URL:           URL,
			Auth:          sshAuth,
			Progress:      logger.Writer(),
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
			logger.Print("Repository URL doesn't match")
			return err
		}
		w, err := r.Worktree()
		if err != nil {
			return err
		}
		err = w.Pull(&git.PullOptions{
			Auth:     sshAuth,
			Progress: logger.Writer(),
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}
	}
	if len(conf.Cmd) <= 0 {
		return errors.New("No command specified")
	}
	logger.Printf("Running command: %s", conf.Cmd)
	cmd := exec.Command(conf.Cmd[0], conf.Cmd[1:]...)
	cmd.Dir = conf.Path
	cmd.Stderr = logger.Writer()
	cmd.Stdout = logger.Writer()
	return cmd.Run()
}
