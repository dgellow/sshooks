package sshgit

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
)

func ExecCmd(cmdName string, args ...string) (string, string, error) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr

	err := cmd.Run()

	return string(bufOut.Bytes()), string(bufErr.Bytes()), err
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func IntToStr(i int) string {
	return strconv.FormatInt(int64(i), 10)
}
