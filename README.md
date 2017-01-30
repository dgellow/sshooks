[![Build Status](https://travis-ci.org/QRCLabs/sshooks.svg?branch=master)](https://travis-ci.org/QRCLabs/sshooks)
[![Coverage Status](https://coveralls.io/repos/github/QRCLabs/sshooks/badge.svg?branch=master)](https://coveralls.io/github/QRCLabs/sshooks?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/qrclabs/sshooks)](https://goreportcard.com/report/github.com/qrclabs/sshooks)

# sshooks - React to ssh commands

## API

### struct `SSHKeygenConfig`

```go
type SSHKeygenConfig struct {
	// Default to rsa
	Type string
	// Default to no password (empty string)
	Passphrase string
}
```

### struct `ServerConfig`

```go
type ServerConfig struct {
	// Default to localhost
	Host              string
	Port              uint
	PrivatekeyPath    string
	PublicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)
	KeygenConfig      SSHKeygenConfig
	CommandsCallbacks map[string]func(args string) error
}
```

### func `Listen`

```go
func Listen(config *ServerConfig)
```

## Example

In this example we setup a server responding to the command `git-upload-pack` commands (e.g: sent by `git clone ssh://git@localhost:1337/QRCLabs/nanogit.git`). You can read the [example program](https://github.com/QRCLabs/sshooks/blob/master/example/main.go) for more details.

```go
func publicKeyHandler(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
    // Do something with the public key
	return &ssh.Permissions{}, nil
}

func handleUploadPack(args string) error {
    // Do something with the args
	return nil
}

func main() {
	commandsHandlers := map[string]func (string) error {
		"git-upload-pack": handleUploadPack,
	}

	config := &sshooks.ServerConfig{
		Host:              "localhost",
		Port:              1337,
		PrivatekeyPath:    "key.rsa",
		KeygenConfig:      sshooks.SSHKeygenConfig{"rsa", ""},
		PublicKeyCallback: publicKeyHandler,
		CommandsCallbacks: commandsHandlers,
	}

	sshooks.Listen(config)

	// Keep the program running
	for {
	}
}
```
