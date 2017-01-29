package main

import (
	"github.com/gogits/gogs/modules/log"
	"github.com/qrclabs/go-ssh-server/sshgit"
)

func main() {
	log.NewLogger(0, "console", `{"level": 0}`)
	config := sshgit.ServerConfig{
		Host: "localhost",
		Port: 1337,
		PrivatekeyPath: "key.rsa",
		KeygenConfig: sshgit.SSHKeygenConfig{"rsa", ""},
	}
	sshgit.Listen(config)
}
