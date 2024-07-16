package pkg

import (
	"log"
	"net/http"
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
		out = append(out, link{n, "repo?name=" + n})
	}
	return
}

func (c *Config) UiHandler(w http.ResponseWriter, r *http.Request) {
	p := page{Nav: c.generateNav()}
	switch r.URL.Path {
	case "/":
		p.Header = "easy way to deploy your code."
		if err := tmpl.Execute(w, "index", p); err != nil {
			errno := http.StatusInternalServerError
			http.Error(w, http.StatusText(errno), errno)
			log.Print(err)
		}
	case "/repo":

		if repo, ok := c.Repos[r.FormValue("name")]; ok {
			p.Header = repo.Name + "@" + repo.Branch
			p.Content = repo
			if err := tmpl.Execute(w, "repo", p); err != nil {
				errno := http.StatusInternalServerError
				http.Error(w, http.StatusText(errno), errno)
				log.Print(err)
			}
		} else {
			http.NotFound(w, r)
			return
		}
	default:
		http.NotFound(w, r)
	}
}
