package sshgit

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

var PackageName = "SSHGit"

func FormatLog(s string) string {
	return fmt.Sprintf("%s: %s", PackageName, s)
}

type PubKeyHandler func (conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

type SSHKeygenConfig struct {
	// Default to rsa
	Type string
	// Default to no password (empty string)
	Passphrase string
}

type ServerConfig struct {
	// Default to localhost
	Host string
	Port uint
	PrivatekeyPath string
	PublicKeyCallback PubKeyHandler
	KeygenConfig SSHKeygenConfig
	Log *log.Logger
}



// Starts an SSH server on given port
func Listen(config ServerConfig) {
	if config.PublicKeyCallback == nil {
		panic("Provide property PublicKeyCallback")
	}
	if config.PrivatekeyPath == "" {
		panic("Provide property PrivatekeyPath")
	}
	if config.KeygenConfig.Type == "" {
		config.KeygenConfig.Passphrase = "rsa"
	}

	sshConfig := &ssh.ServerConfig{PublicKeyCallback: config.PublicKeyCallback}
	keyPath := config.PrivatekeyPath
	if !FileExists(keyPath) {
		os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)

		// Generate a new ssh key pair without password
		// -f <filename>
		// -t <keytype>
		// -N <new_passphrase>
		_, stderr, err :=  ExecCmd("ssh-keygen", "-f", keyPath, "-t", config.KeygenConfig.Type, "-N", config.KeygenConfig.Passphrase)
		if err != nil {
			panic(FormatLog(fmt.Sprintf("Failed to generate private key: %v - %s", err, stderr)))
		}
		config.Log.Trace(FormatLog(fmt.Sprintf("Generated a new private key at: %s", keyPath)))
	}

	// Read private key
	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic(FormatLog("Failed to read private key"))
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic(FormatLog("Failed to parse private key"))
	}
	sshConfig.AddHostKey(private)

	host := config.Host
	if host == "" {
		host = "localhost"
	}
	go serve(sshConfig, host, config.Port, config.Log)
}

// Actual server
func serve(config *ssh.ServerConfig, host string, port int, log *log.Logger) {
	// Listen on given host and port
	listener, err := net.Listen("tcp", host + ":" + IntToStr(port))
	if err != nil {
		log.Fatal(4, FormatLog(fmt.Sprintf("Fail to start SSH server: %v", err)))
	}

	// Infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(3, FormatLog(fmt.Sprintf("Error accepting incoming connection: %v", err)))
			continue
		}

		// Before use, a handshake must be performed on the incoming
		// net.Conn.
		// It must be handled in a separate goroutine, otherwise one
		// user could easily block entire loop. For example, user could
		// be asked to trust server key fingerprint and hangs.
		go func() {
			log.Trace(FormatLog(fmt.Sprintf("Handshaking was terminated: %v", err)))
			sConn, channels, reqs, err := ssh.NewServerConn(conn, config)
			if err != nil {
				if err == io.EOF {
					log.Warn(FormatLog(fmt.Sprintf("Handshaking was terminated: %v", err)))
				} else {
					log.Error(3, FormatLog(fmt.Sprintf("Error on handshaking: %v", err)))
				}
				return
			}

			log.Trace(FormatLog(fmt.Sprintf("Connection from %s (%s)", sConn.RemoteAddr(), sConn.ClientVersion())))
			go ssh.DiscardRequests(reqs)
			// go handleServerConn(sConn.Permissions.Extensions["key-id"], channels)
		}()
	}
}



func handleServerConn(keyID string, chans <-chan ssh.NewChannel) {
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, reqs, err := newChan.Accept()
		if err != nil {
			log.Error(3, "Error accepting channel: %v", err)
			continue
		}

		go func(in <-chan *ssh.Request) {
			defer ch.Close()
			for req := range in {
				payload := cleanCommand(string(req.Payload))
				switch req.Type {
				case "env":
					args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
					if len(args) != 2 {
						log.Warn("SSH: Invalid env arguments: '%#v'", args)
						continue
					}
					args[0] = strings.TrimLeft(args[0], "\x04")
					_, _, err := com.ExecCmdBytes("env", args[0]+"="+args[1])
					if err != nil {
						log.Error(3, "env: %v", err)
						return
					}
				case "exec":
					cmdName := strings.TrimLeft(payload, "'()")
					log.Trace("SSH: Payload: %v", cmdName)

					args := []string{"serv", "key-" + keyID, "--config=" + setting.CustomConf}
					log.Trace("SSH: Arguments: %v", args)
					cmd := exec.Command(setting.AppPath, args...)
					cmd.Env = append(os.Environ(), "SSH_ORIGINAL_COMMAND="+cmdName)

					stdout, err := cmd.StdoutPipe()
					if err != nil {
						log.Error(3, "SSH: StdoutPipe: %v", err)
						return
					}
					stderr, err := cmd.StderrPipe()
					if err != nil {
						log.Error(3, "SSH: StderrPipe: %v", err)
						return
					}
					input, err := cmd.StdinPipe()
					if err != nil {
						log.Error(3, "SSH: StdinPipe: %v", err)
						return
					}

					// FIXME: check timeout
					if err = cmd.Start(); err != nil {
						log.Error(3, "SSH: Start: %v", err)
						return
					}

					req.Reply(true, nil)
					go io.Copy(input, ch)
					io.Copy(ch, stdout)
					io.Copy(ch.Stderr(), stderr)

					if err = cmd.Wait(); err != nil {
						log.Error(3, "SSH: Wait: %v", err)
						return
					}

					ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				default:
				}
			}
		}(reqs)
	}
}
