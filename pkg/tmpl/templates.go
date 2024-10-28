//go:build !dev

package tmpl

import (
	"embed"
	"html/template"
	"io"
	"log"
)

var tmpl *template.Template

//go:embed *.html
var embedFS embed.FS

func init() {
	var err error
	if tmpl, err = template.ParseFS(embedFS, "*.html"); err != nil {
		log.Fatal(err)
	}
}

func Execute(w io.Writer, name string, data any) error {
	return tmpl.ExecuteTemplate(w, name, data)
}
