# µCI

µCI *(micro-see-ay)* is a minimalistic deployment tool using webhooks.
It provides single http(s) endpoint that all your git repositories send
webhooks to and runs specified command after receiving webhook on that
endpoint.

It was tested and currently works with gitea/forgejo and github.

## Building

- basic: `go build`
- minimal binary size: `go build -ldflags="-w -s"`
- static binary: `CGO_ENABLED=0 go build -ldflags="-s -w"`

## Configuration

Save your configuration in `config.toml` file.

Example:
```toml
address  = "<bind address>"
TLS      = true/false
keyfile  = "<tls private key>"
certfile = "<tls certificate>"
logdb    = "<logfile>"
pidfile  = "<pidfile>"
username = "<username>"
password = "<password>"

[[repo]]
        name     = "<repositry name>"
        branch   = "<git branch>"
        keyfile  = "<ssh private key file>"
        path     = "<path to repo>"
        cmd      = ["<build>", "<commands>"]
        auth     = "<auth token>"
```

You can hot reload your configuration using `uci -s reload`

Repository name must be in format `<username>/<repository name>`

## Configuring git repository

- Navigate to your git repository settings
- Go to "webhooks"
- Put `<url>/uci` in "URL" field
- Set content type to "application/json"
- Generate random string and put it in "Secret"/"Authorization header"
- Optionally set branch filters, trigger events, etc...

Don't forget to add your public ssh key to "deploy keys" in your
repository settings.

You also need to have your ssh known hosts entry either in
`/etc/ssh/known_hosts` or in `$HOME/.ssh/known_hosts`

