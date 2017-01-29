[![Build Status](https://travis-ci.org/QRCLabs/go-ssh-git.svg?branch=master)](https://travis-ci.org/QRCLabs/go-ssh-git)
[![Coverage Status](https://coveralls.io/repos/github/QRCLabs/go-ssh-git/badge.svg?branch=master)](https://coveralls.io/github/QRCLabs/go-ssh-git?branch=master)

# Handle git requests sent via a SSH connection


Initially based on [gogs's ssh](https://github.com/gogits/gogs/blob/master/modules/ssh/ssh.go) package.

## Build and run default server

```
$ go build
$ ./go-ssh-server
```

## Run tests

```
$ go test ./sshgit
```
