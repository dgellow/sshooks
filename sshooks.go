package sshooks

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"github.com/qrclabs/sshooks/log"
)

var packageName = "sshooks"

func formatLog(s string) string {
	return fmt.Sprintf("%s: %s", packageName, s)
}

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
	CommandsCallbacks map[string]func(keyId string, cmd string, args string) error
	// Logger based on the interface defined in sshooks/log
 	Log               log.Log
}

// Starts an SSH server on given port
func Listen(config *ServerConfig) {
	if config.PublicKeyCallback == nil {
		config.Log.Fatal(formatLog("PublicKeyCallback cannot be nil"))
	}
	if config.PrivatekeyPath == "" {
		config.Log.Fatal(formatLog("PrivatekeyPath cannot be empty"))
	}
	if config.KeygenConfig.Type == "" {
		config.KeygenConfig.Passphrase = "rsa"
	}

	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			keyId, err := config.PublicKeyCallback(conn, key)
			if err != nil {
				config.Log.Error("Error while handling public key: %v", err)
			}
			return &ssh.Permissions{Extensions: map[string]string{"key-id": keyId}}, nil
		},
	}
	keyPath := config.PrivatekeyPath
	if !FileExists(keyPath) {
		os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)

		// Generate a new ssh key pair without password
		// -f <filename>
		// -t <keytype>
		// -N <new_passphrase>
		_, stderr, err := ExecCmd("ssh-keygen", "-f", keyPath, "-t", config.KeygenConfig.Type, "-N", config.KeygenConfig.Passphrase)
		if err != nil {
			config.Log.Fatal(formatLog("Failed to generate private key: %v - %s"), err, stderr)
		}
		config.Log.Trace(formatLog("Generated a new private key at: %s"), keyPath)
	}

	// Read private key
	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		config.Log.Fatal(formatLog("Failed to read private key"))
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		config.Log.Fatal(formatLog("Failed to parse private key"))
	}
	sshConfig.AddHostKey(private)

	host := config.Host
	if host == "" {
		host = "localhost"
	}

	go serve(config, sshConfig, host, config.Port)
}

// Actual server
func serve(config *ServerConfig, sshConfig *ssh.ServerConfig, host string, port uint) {
	// Listen on given host and port
	listener, err := net.Listen("tcp", host+":"+UIntToStr(port))
	if err != nil {
		config.Log.Fatal(formatLog("Failed to start SSH server: %v"), err)
	}

	// Infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			config.Log.Error(formatLog("Error accepting incoming connection: %v"), err)
			continue
		}

		// Before use, a handshake must be performed on the incoming
		// net.Conn.
		// It must be handled in a separate goroutine, otherwise one
		// user could easily block entire loop. For example, user could
		// be asked to trust server key fingerprint and hangs.
		go func() {
			config.Log.Warn(formatLog("Handshaking was terminated: %v"), err)
			sConn, channels, reqs, err := ssh.NewServerConn(conn, sshConfig)
			if err != nil {
				if err == io.EOF {
					config.Log.Warn(formatLog(fmt.Sprintf("Handshaking was terminated: %v", err)))
				} else {
					config.Log.Error(formatLog(fmt.Sprintf("Error on handshaking: %v", err)))
				}
				return
			}

			config.Log.Trace(formatLog(fmt.Sprintf("Connection from %s (%s)", sConn.RemoteAddr(), sConn.ClientVersion())))
			go ssh.DiscardRequests(reqs)
			go handleServerConn(config, sConn.Permissions.Extensions["key-id"], channels)
		}()
	}
}

// Remove unwanted characters in the received command
func cleanCommand(cmd string) string {
	i := strings.Index(cmd, "git")
	if i == -1 {
		return cmd
	}
	return cmd[i:]
}

func handleServerConn(config *ServerConfig, keyId string, chans <-chan ssh.NewChannel) {
	fmt.Println("Handle server connection")

	// Loop on channels
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, reqs, err := newChan.Accept()
		if err != nil {
			config.Log.Error(formatLog("Error accepting channel: %v"), err)
			continue
		}

		go func(in <-chan *ssh.Request) {
			defer ch.Close()

			for req := range in {
				fmt.Println("Request")
				fmt.Printf("req.Type: %v\n", req.Type)
				fmt.Printf("req.Payload: %s\n", req.Payload)
				fmt.Println("")

				payload := cleanCommand(string(req.Payload))
				switch req.Type {
				case "env":
					args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
					if len(args) != 2 {
						config.Log.Error(formatLog("Invalid env arguments: %#v"), args)
						continue
					}
					args[0] = strings.TrimLeft(args[0], "\x04")

					_, _, err := ExecCmd("env", args[0]+"="+args[1])
					if err != nil {
						config.Log.Error("Error while executing env command: %v", err)
						return
					}
				case "exec":
					cmd := strings.TrimLeft(payload, "'()")
					config.Log.Trace(formatLog("Cleaned payload: %v"), cmd)
					err := handleCommand(config, keyId, cmd)
					if err != nil {
						config.Log.Error("Error in command handler: cmd: %s, error: %v", cmd, err)
					}

					req.Reply(true, nil)
					ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				default:
				}

				fmt.Println("")
				fmt.Println("")
			}
		}(reqs)
	}
}

func parseCommand(cmd string) (exec string, args string) {
	ss := strings.SplitN(cmd, " ", 2)
	if len(ss) != 2 {
		return "", ""
	}
	return ss[0], strings.Replace(ss[1], "'/", "'", 1)
}

func handleCommand(config *ServerConfig, keyId string, cmd string) error {
	exec, args := parseCommand(cmd)
	cmdHandler, present := config.CommandsCallbacks[exec]
	if !present {
		config.Log.Trace("No handler for command: %s, args: %v", exec, args)
		return nil
	}
	return cmdHandler(keyId, cmd, args)
}
