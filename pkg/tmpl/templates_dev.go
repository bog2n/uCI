//go:build dev

package tmpl

import (
	"html/template"
	"io"
	"os"
)

var tmpl *template.Template

func Execute(w io.Writer, name string, data any) error {
	var err error
	if tmpl, err = template.ParseFS(os.DirFS("pkg/tmpl"), "*.html"); err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, name, data)
}
