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

func (c *Config) UiHandler(w http.ResponseWriter, r *http.Request) {
	p := page{
		Header: "Hello there",
		Nav: []link{
			link{"test1", "/"},
			link{"test2", "/"},
			link{"test3", "/"},
		},
		Content: "hello!",
	}
	switch r.URL.Path {
	case "/":
		if err := tmpl.Execute(w, "index", p); err != nil {
			errno := http.StatusInternalServerError
			http.Error(w, http.StatusText(errno), errno)
			log.Print(err)
		}
	default:
		http.NotFound(w, r)
	}
}
