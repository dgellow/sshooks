package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/qrclabs/sshooks"
	"golang.org/x/crypto/ssh"
)

var (
	logger *Logger
)

func checkPubKey(key ssh.PublicKey) (string, error) {
	keystr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))

	filename := "authorized_keys.txt"
	file, err := os.Open(filename)
	if err != nil {
		logger.Fatal("Cannot open file: %s, error: %v", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		fmt.Printf("line: %s\n", line)
		fmt.Printf("keystr: %s\n", keystr)
		fmt.Println("")

		if line == keystr {
			fmt.Println("found key!")
			return keystr, nil
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Fatal("Error while reading file: %s, error: %v", file, err)
	}

	fmt.Println("found nothing :(")
	return keystr, errors.New("key not found")
}

func publicKeyHandler(conn ssh.ConnMetadata, key ssh.PublicKey) (keyId string, err error) {
	keyId, err = checkPubKey(key)
	if err != nil {
		logger.Error("Cannot find key: %v", err)
		return "", err
	}
	return keyId, nil
}

func handleUploadPack(keyId string, cmd string, args string) (*exec.Cmd, error) {
	logger.Trace("Handle git-upload-pack: cmd: %s, args: %s, keyId: %s", cmd, args, keyId)
	command := exec.Command("git-upload-pack", "repo1.git")
	return command, nil
}

func handleUploadArchive(keyId string, cmd string, args string) (*exec.Cmd, error) {
	logger.Trace("Handle git-upload-archive: cmd: %s, args: %s, keyId: %s", cmd, args, keyId)
	return nil, nil
}

func handleReceivePack(keyId string, cmd string, args string) (*exec.Cmd, error) {
	logger.Trace("Handle git-receive-pack: cmd: %s, args: %s, keyId: %s", cmd, args, keyId)
	return exec.Command("ls"), nil
}

func main() {
	logger = &Logger{LogLevel: 0, Prefix: "example"}

	fmt.Println("Start program")

	commandsHandlers := map[string]func(string, string, string) (*exec.Cmd, error){
		"git-upload-pack":    handleUploadPack,
		"git-upload-archive": handleUploadArchive,
		"git-receive-pack":   handleReceivePack,
	}

	config := &sshooks.ServerConfig{
		Host:              "localhost",
		Port:              1337,
		PrivatekeyPath:    "key.rsa",
		KeygenConfig:      sshooks.SSHKeygenConfig{"rsa", ""},
		PublicKeyCallback: publicKeyHandler,
		CommandsCallbacks: commandsHandlers,
		Log:               logger,
	}

	fmt.Println("Run server")
	sshooks.Listen(config)

	// Keep the program running
	for {
	}
}
