package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

func (c *Config) MainHandler(w http.ResponseWriter, r *http.Request) {
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
		repoKey := p.Repo.Name + " " + b
		if conf, ok := c.Repos[repoKey]; ok {
			shasum := r.Header.Get("x-hub-signature-256")
			h := hmac.New(sha256.New, []byte(conf.Auth))
			h.Write(payload)
			payloadsum := "sha256=" + hex.EncodeToString(h.Sum(nil))
			if payloadsum != shasum {
				errno := http.StatusUnauthorized
				http.Error(w, http.StatusText(errno), errno)
				log.Print("Unauthorized request")
				return
			}
			err := deploy(conf, p.Repo.URL)
			if err != nil {
				log.Print("Error deploying: ", err)
				errno := http.StatusInternalServerError
				http.Error(w, http.StatusText(errno), errno)
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
