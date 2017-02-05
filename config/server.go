package config

import (
	"os/exec"

	"github.com/qrclabs/sshooks/errors"
	"github.com/qrclabs/sshooks/log"
	"golang.org/x/crypto/ssh"
)

type SSHKeygenConfig struct {
	// Default to rsa
	Type string
	// Default to no password (empty string)
	Passphrase string
}

type ServerConfig struct {
	// Default to localhost
	Host              string
	Port              uint
	PrivatekeyPath    string
	PublicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (keyId string, err error)
	KeygenConfig      SSHKeygenConfig
	CommandsCallbacks map[string]func(keyId string, cmd string, args string) (*exec.Cmd, error)
	// Logger based on the interface defined in sshooks/log
	Log log.Log
}

func (sc *ServerConfig) Validate() error {
	if sc.PublicKeyCallback == nil {
		return errors.ErrNoPubKeyCallback
	}
	if sc.CommandsCallbacks == nil {
		return errors.ErrNoCmdsCallbacks
	}
	if sc.PrivatekeyPath == "" {
		return errors.ErrEmptyPrivKeyPath
	}
	if sc.KeygenConfig.Type == "" {
		sc.KeygenConfig.Passphrase = "rsa"
	}
	if sc.Host == "" {
		sc.Host = "localhost"
	}
	return nil
}
