package pkg

import (
	"log"
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
	Repos        map[string]RepoConfig
}

func (c *Config) Reload(configfile string) {
	file, err := os.ReadFile(configfile)
	if err != nil {
		log.Fatal(err)
	}
	_, err = toml.Decode(string(file), &c)
	if err != nil {
		log.Fatal(err)
	}
	c.Repos = make(map[string]RepoConfig)
	for _, v := range c.Repositories {
		c.Repos[v.Name+" "+v.Branch] = v
	}
}
