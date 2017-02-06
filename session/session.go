package session

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"

	sshooks "github.com/qrclabs/sshooks/config"
	"github.com/qrclabs/sshooks/errors"
	"github.com/qrclabs/sshooks/util"
	"golang.org/x/crypto/ssh"
)

var packageName = "sshooks"

func (s *Session) formatLog(str string) string {
	return fmt.Sprintf("%s: [%s] %s", s.conn.RemoteAddr(), packageName, str)
}

type Session struct {
	conn      net.Conn
	sshConn   *ssh.ServerConn
	config    *sshooks.ServerConfig
	sshConfig *ssh.ServerConfig
	channels  <-chan ssh.NewChannel
	requests  <-chan *ssh.Request
}

func NewSession(config *sshooks.ServerConfig, sshConfig *ssh.ServerConfig, conn net.Conn) (*Session, error) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, sshConfig)
	if err != nil {
		return nil, err
	}
	return &Session{conn, sshConn, config, sshConfig, channels, requests}, nil
}

func (s *Session) Run() {
	go ssh.DiscardRequests(s.requests)
	go s.handleChannels()
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

func (s *Session) handleCommand(keyId string, payload string) (*exec.Cmd, error) {
	s.config.Log.Trace(s.formatLog("handleCommand"))
	cmdName := strings.TrimLeft(payload, "'()")
	execName, args := parseCommand(cmdName)
	cmdHandler, present := s.config.CommandsCallbacks[execName]
	if !present {
		s.config.Log.Trace(s.formatLog("No handler for command: %s, args: %v"),
			execName, args)
		return exec.Command(""), nil
	}
	return cmdHandler(keyId, cmdName, args)
}

func (s *Session) envRequest(payload string) error {
	s.config.Log.Trace(s.formatLog("envRequest"))
	s.config.Log.Trace(s.formatLog("payload: %s"), payload)
	args := strings.Split(strings.Replace(payload, "\x00", "", -1), "\v")
	if len(args) != 2 {
		return errors.ErrInvalidEnvArgs
	}

	args[0] = strings.TrimLeft(args[0], "\x04")
	s.config.Log.Trace(s.formatLog("util.execCmd: args[0]: %s, args[1]: %s"),
		args[0], args[1])
	_, _, err := util.ExecCmd("env", args[0]+"="+args[1])
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) execRequest(keyId string, payload string, ch ssh.Channel, req *ssh.Request) error {
	s.config.Log.Trace(s.formatLog("execRequest"))
	s.config.Log.Trace(s.formatLog("payload: %s"), payload)
	cmd, err := s.handleCommand(keyId, payload)
	if cmd == nil {
		s.config.Log.Trace(s.formatLog("Cmd object returned by handleCommand is nil"))
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

func (s *Session) handleChannels() error {
	s.config.Log.Trace(s.formatLog("handleChannels"))
	for ch := range s.channels {
		if t := ch.ChannelType(); t != "session" {
			s.config.Log.Trace(s.formatLog("Ignore channel type: %s"), t)
			ch.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		} else {
			s.config.Log.Trace(s.formatLog("Session channel found"))
			c, requests, err := ch.Accept()
			if err != nil {
				return err
			}
			go s.handleRequests(c, requests)
			return nil
		}
	}
	return errors.ErrNoSessionChannel
}

func (s *Session) handleRequests(ch ssh.Channel, reqs <-chan *ssh.Request) {
	s.config.Log.Trace(s.formatLog("handleRequests"))
	keyId := s.sshConn.Permissions.Extensions["key-id"]
	defer ch.Close()

	go func(in <-chan *ssh.Request) {
		for req := range in {
			s.config.Log.Trace(s.formatLog("Request: type : %s, payload: "),
				req.Type, string(req.Payload))
			payload := cleanCommand(string(req.Payload))
			switch req.Type {
			case "env":
				err := s.envRequest(payload)
				if err != nil {
					s.config.Log.Error(s.formatLog("%v"), err)
				}
			case "exec":
				err := s.execRequest(keyId, payload, ch, req)
				if err != nil {
					s.config.Log.Error(s.formatLog("%v"), err)
				}
			}
		}
	}(reqs)
}
