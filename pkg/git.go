package pkg

import (
	"log"
	"net/url"
	"os"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func deploy(conf RepoConfig, URL string) error {
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
