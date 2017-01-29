package main

import (
	"strings"
	"os"
	"fmt"
	"errors"
	"bufio"

	"github.com/gogits/gogs/modules/log"
	"github.com/qrclabs/go-ssh-server/sshgit"
	"golang.org/x/crypto/ssh"
)

func checkPubKey(key ssh.PublicKey) (*ssh.PublicKey, error) {
	keystr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))

	filename := "authorized_keys.txt"
	file, err := os.Open(filename)
	if err!= nil {
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

func PublicKeyHandler(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	_, err := checkPubKey(key);
	if err != nil {
		log.Error(3, "Cannot find key: %v", err)
		return nil, err
	}
	return &ssh.Permissions{}, nil
}

func main() {
	fmt.Println("Start program")

	config := sshgit.ServerConfig{
		Host: "localhost",
		Port: 1337,
		PrivatekeyPath: "key.rsa",
		KeygenConfig: sshgit.SSHKeygenConfig{"rsa", ""},
		PublicKeyCallback: PublicKeyHandler,
	}

	fmt.Println("Run server")
	sshgit.Listen(config)

	// Keep the program running
	for {}
}
