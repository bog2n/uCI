package pkg

import (
	"flag"
	"fmt"
	"os"
)

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(flag.CommandLine.Output(), `
Config file format:

address  = "<bind address>"
TLS      = true/false
keyfile  = "<tls private key>"
certfile = "<tls certificate>"
logfile  = "<logfile>"

[[repo]]
	name     = "<gitea repo name>"
	branch   = "<git branch>"
	keyfile  = "<ssh private key file>"
	path     = "<path to repo>"
	cmd      = "<build command>"
	auth     = "<auth token>"
...

You might want to specify SSH_KNOWN_HOSTS environment variable for ssh to work

`)
}