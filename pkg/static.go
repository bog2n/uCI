//go:build !dev

package pkg

import (
	"embed"
	"net/http"
)

//go:embed static
var static embed.FS

var StaticFS http.FileSystem

func init() {
	StaticFS = http.FS(static)
}
