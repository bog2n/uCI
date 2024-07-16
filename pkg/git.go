package pkg

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

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
