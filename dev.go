//go:build dev

package main

import (
	"log"
)

func init() {
	log.Default().SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
}
