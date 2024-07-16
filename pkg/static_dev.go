//go:build dev

package pkg

import (
	"net/http"
)

var StaticFS http.FileSystem

func init() {
	StaticFS = http.Dir("pkg")
}
