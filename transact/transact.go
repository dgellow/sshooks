package transact

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	sshooks "github.com/qrclabs/sshooks/config"
	"github.com/qrclabs/sshooks/errors"
	"github.com/qrclabs/sshooks/util"
	"golang.org/x/crypto/ssh"
)

var packageName = "sshooks"

func formatLog(s string) string {
	return fmt.Sprintf("%s: %s", packageName, s)
}

// Remove unwanted characters in the received command
func cleanCommand(cmd string) string {
	i := strings.Index(cmd, "git")
	if i == -1 {
		return cmd
	}
	return cmd[i:]
}

func parseCommand(cmd string) (exec string, args string) {
	ss := strings.SplitN(cmd, " ", 2)
	if len(ss) != 2 {
		return "", ""
	}
	return ss[0], strings.Replace(ss[1], "'/", "'", 1)
}

func handleCommand(config *sshooks.ServerConfig, keyId string, payload string) (*exec.Cmd, error) {
	cmdName := strings.TrimLeft(payload, "'()")
	config.Log.Trace(formatLog("Cleaned payload: %v"), cmdName)
	execName, args := parseCommand(cmdName)
	cmdHandler, present := config.CommandsCallbacks[execName]
	if !present {
		config.Log.Trace(formatLog("No handler for command: %s, args: %v"), execName, args)
		return exec.Command(""), nil
	}
	return cmdHandler(keyId, cmdName, args)
}

func envRequest(config *sshooks.ServerConfig, payload string) error {
	args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
	if len(args) != 2 {
		return errors.ErrInvalidEnvArgs
	}

	args[0] = strings.TrimLeft(args[0], "\x04")
	_, _, err := util.ExecCmd("env", args[0]+"="+args[1])
	if err != nil {
		return err
	}
	return nil
}

func execRequest(config *sshooks.ServerConfig, keyId string, payload string, ch ssh.Channel, req *ssh.Request) error {
	cmd, err := handleCommand(config, keyId, payload)
	if cmd == nil {
		config.Log.Trace("Cmd object returned by handleCommand is nil")
		return nil
	}
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	// FIXME: check timeout
	if err = cmd.Start(); err != nil {
		return err
	}

	req.Reply(true, nil)
	go io.Copy(stdin, ch)
	io.Copy(ch, stdout)
	io.Copy(ch.Stderr(), stderr)

	ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
	return nil
}

func NewSession(config *sshooks.ServerConfig, conn *ssh.ServerConn, channels <-chan ssh.NewChannel) error {
	for ch := range channels {
		if t := ch.ChannelType(); t != "session" {
			config.Log.Trace("Ignore channel type: %s, from: %s", t, conn.RemoteAddr())
			ch.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		} else {
			c, requests, err := ch.Accept()
			if err != nil {
				return err
			}
			go HandleRequest(config, conn, c, requests)
			return nil
		}
	}
	return errors.ErrNoSessionChannel
}

func HandleRequest(config *sshooks.ServerConfig, conn *ssh.ServerConn, ch ssh.Channel, reqs <-chan *ssh.Request) {
	keyId := conn.Permissions.Extensions["key-id"]
	go func(in <-chan *ssh.Request) {
		defer ch.Close()
		for req := range in {
			config.Log.Trace(formatLog("Request: %#v"), req)
			config.Log.Trace(formatLog("Payload (as string): %s"), string(req.Payload))
			payload := cleanCommand(string(req.Payload))
			switch req.Type {
			case "env":
				if err := envRequest(config, payload); err != nil {
					// Do something with err
				}
				return
			case "exec":
				if err := execRequest(config, keyId, payload, ch, req); err != nil {
					// Do something with err
				}
				return
			}
		}
	}(reqs)
}
