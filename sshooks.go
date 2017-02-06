package sshooks

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	sshooks "github.com/qrclabs/sshooks/config"
	"github.com/qrclabs/sshooks/session"
	"github.com/qrclabs/sshooks/util"
	"golang.org/x/crypto/ssh"
)

var packageName = "sshooks"

func formatLog(s string) string {
	return fmt.Sprintf("%s: %s", packageName, s)
}

func genPrivateKey(config *sshooks.ServerConfig, keyPath string) error {
	os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)

	// Generate a new ssh key pair without password
	// -f <filename>
	// -t <keytype>
	// -N <new_passphrase>
	_, stderr, err := util.ExecCmd("ssh-keygen", "-f", keyPath, "-t", config.KeygenConfig.Type, "-N", config.KeygenConfig.Passphrase)
	if err != nil {
		return fmt.Errorf("failed to generate private key %s: %v", stderr, err)
	}
	config.Log.Trace(formatLog("Generated a new private key at: %s"), keyPath)
	return nil
}

func readPrivateKey(keyPath string) (ssh.Signer, error) {
	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, err
	}
	return private, nil
}

// Starts an SSH server on given port
func Listen(config *sshooks.ServerConfig) error {
	err := config.Validate()
	if err != nil {
		return nil
	}

	sshConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			keyId, err := config.PublicKeyCallback(conn, key)
			if err != nil {
				config.Log.Error(formatLog("Error while handling public key: %v"), err)
			}
			return &ssh.Permissions{Extensions: map[string]string{"key-id": keyId}}, nil
		},
	}

	keyPath := config.PrivatekeyPath
	if !util.FileExists(keyPath) {
		err := genPrivateKey(config, keyPath)
		if err != nil {
			return err
		}
	}
	private, err := readPrivateKey(keyPath)
	if err != nil {
		return err
	}
	sshConfig.AddHostKey(private)

	go serve(config, sshConfig)
	return nil
}

// Actual server
func serve(config *sshooks.ServerConfig, sshConfig *ssh.ServerConfig) error {
	listener, err := net.Listen("tcp", config.Host+":"+util.UIntToStr(config.Port))
	defer listener.Close()
	if err != nil {
		config.Log.Fatal(formatLog("Failed to start SSH server: %v"), err)
	}

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
			config.Log.Trace(formatLog("[%s] Handshaking"), conn.RemoteAddr())
			session, err := session.NewSession(config, sshConfig, conn)
			if err != nil {
				config.Log.Error(formatLog("%v"), err)
				return
			}
			session.Run()
		}()
	}
}
