package pkg

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"uci/pkg/tmpl"
)

type page struct {
	Header  string
	Nav     []link
	Content any
}

type link struct {
	Name string
	Href string
}

func (c *Config) generateNav() (out []link) {
	for _, repo := range c.Repositories {
		n := repo.Name + "@" + repo.Branch
		out = append(out, link{n, n})
	}
	return
}

func (c *Config) MainHandler(w http.ResponseWriter, r *http.Request) {
	p := page{
		Nav:     c.generateNav(),
		Header:  "simple way to deploy your code.",
		Content: getLatestLogs(),
	}
	if err := tmpl.Execute(w, "index", p); err != nil {
		errno := http.StatusInternalServerError
		http.Error(w, http.StatusText(errno), errno)
		log.Print(err)
	}
}

func (c *Config) RepoHandler(w http.ResponseWriter, r *http.Request) {
	p := page{Nav: c.generateNav()}

	if repo, ok := c.Repos[r.PathValue("name")]; ok {
		p.Header = repo.Name + "@" + repo.Branch
		logs := getLogs(p.Header)
		p.Content = struct {
			Repo any
			Logs any
		}{repo, logs}
		if err := tmpl.Execute(w, "repo", p); err != nil {
			errno := http.StatusInternalServerError
			http.Error(w, http.StatusText(errno), errno)
			log.Print(err)
		}
	} else {
		http.NotFound(w, r)
		return
	}
}

func (c *Config) LogsHandler(w http.ResponseWriter, r *http.Request) {
	p := page{Nav: c.generateNav()}
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errno := http.StatusInternalServerError
		http.Error(w, http.StatusText(errno), errno)
		return
	}
	if l := getLog(id); l.Id >= 0 {
		p.Header = fmt.Sprintf("%s %s - log", l.Name, l.Time)
		p.Content = l
		if err := tmpl.Execute(w, "logs", p); err != nil {
			errno := http.StatusInternalServerError
			http.Error(w, http.StatusText(errno), errno)
			log.Print(err)
		}
	} else {
		http.NotFound(w, r)
	}
}
