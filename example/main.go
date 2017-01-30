package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gogits/gogs/modules/log"
	"github.com/qrclabs/sshooks"
	"golang.org/x/crypto/ssh"
)

func checkPubKey(key ssh.PublicKey) (*ssh.PublicKey, error) {
	keystr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))

	filename := "authorized_keys.txt"
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(4, "Cannot open file: %s, error: %v", filename, err)
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
			return &key, nil
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(4, "Error while reading file: %s, error: %v", file, err)
	}

	fmt.Println("found nothing :(")
	return nil, errors.New("key not found")
}

func publicKeyHandler(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	_, err := checkPubKey(key)
	if err != nil {
		log.Error(3, "Cannot find key: %v", err)
		return nil, err
	}
	return &ssh.Permissions{}, nil
}

func handleUploadPack(args string) error {
	log.Trace("Handle git-upload-pack: args: %s", args)
	return nil
}

func handleUploadArchive(args string) error {
	log.Trace("Handle git-upload-archive: args: %s", args)
	return nil
}

func handleReceivePack(args string) error {
	log.Trace("Handle git-receive-pack: args: %s", args)
	return nil
}

func main() {
	log.NewLogger(0, "console", `{"level": 0}`)

	fmt.Println("Start program")

	commandsHandlers := map[string]func (string) error {
		"git-upload-pack": handleUploadPack,
		"git-upload-archive": handleUploadArchive,
		"git-receive-pack": handleReceivePack,
	}

	config := &sshooks.ServerConfig{
		Host:              "localhost",
		Port:              1337,
		PrivatekeyPath:    "key.rsa",
		KeygenConfig:      sshooks.SSHKeygenConfig{"rsa", ""},
		PublicKeyCallback: publicKeyHandler,
		CommandsCallbacks: commandsHandlers,
	}

	fmt.Println("Run server")
	sshooks.Listen(config)

	// Keep the program running
	for {
	}
}
