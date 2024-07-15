package pkg

import (
	"os"

	"github.com/BurntSushi/toml"
)

type RepoConfig struct {
	SshPrivKey string   `toml:"keyfile"`
	Name       string   `toml:"name"`
	Path       string   `toml:"path"`
	Cmd        []string `toml:"cmd"`
	Branch     string   `toml:"branch"`
	Auth       string   `toml:"auth"`
}

type Config struct {
	Repositories []RepoConfig `toml:"repo"`
	TLS          bool         `toml:"TLS"`
	CertFile     string       `toml:"certfile"`
	KeyFile      string       `toml:"keyfile"`
	Address      string       `toml:"address"`
	PidFile      string       `toml:"pidfile"`
	LogFile      string       `toml:"logfile"`
	Repos        map[string]RepoConfig
}

func (c *Config) Reload(configfile string) error {
	var tmp Config
	file, err := os.ReadFile(configfile)
	if err != nil {
		return err
	}
	_, err = toml.Decode(string(file), &tmp)
	if err != nil {
		return err
	}
	tmp.Repos = make(map[string]RepoConfig)
	for _, v := range tmp.Repositories {
		tmp.Repos[v.Name+" "+v.Branch] = v
	}
	*c = tmp
	return nil
}
