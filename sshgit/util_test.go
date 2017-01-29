package sshgit

import (
	"testing"
	"os"
	"io/ioutil"
)

type TestDataUIntToStr struct {
	in uint
	out string
}

func TestUIntToStr(t *testing.T) {
	tests := []TestDataUIntToStr{
		{0, "0"},
		{10, "10"},
		{65535, "65535"},
	}

	for i, test := range tests {
		actual := UIntToStr(test.in)
		if test.out != actual {
			t.Errorf("#%d: UIntToStr(%d)=%s; expected %s", i, test.in, actual, test.out)
		}
	}
}

type TestDataFileExists struct {
	in string
	out bool
}

func TestFileExists(t *testing.T) {
	existingDir, err := ioutil.TempDir("", "tempdir")
	if err != nil {
		t.Errorf("Couldn't create temp directory: %s. Tests not run", existingDir)
	}

	existingFile, err := ioutil.TempFile("", "tempfile")
	if err != nil {
		t.Errorf("Couldn't create temp file: %s. Tests not run", existingFile.Name())
	}

	defer os.RemoveAll(existingDir)
	defer os.Remove(existingFile.Name())

	tests := []TestDataFileExists{
		{"missing_directory", false},
		{existingDir, true},
		{"missing_file", false},
		{existingFile.Name(), true},
	}

	for i, test := range tests {
		actual := FileExists(test.in)
		if test.out != actual {
			t.Errorf("#%d: FileExists(%s)=%t; expected %t", i, test.in, actual, test.out)
		}
	}
}

type TestDataExecCmd struct {
	in string
	inArgs []string
	stdout string
	stderr string
	err string
}

func TestExecCmd(t *testing.T) {
	tests := []TestDataExecCmd{
		{"", []string{}, "", "", "fork/exec : no such file or directory"},
		{"echo", []string{"hello", "you"}, "hello you\n", "", ""},
		{"ls", []string{"foo/bar/nopnop"}, "", "ls: foo/bar/nopnop: No such file or directory\n", "exit status 1"},
	}

	for i, test := range tests {
		stdout, stderr, err := ExecCmd(test.in, test.inArgs...)
		if test.stdout != stdout {
			t.Errorf("#%d: stdout, _, _ := ExecCmd(%s) == %s; expected %s", i, test.in, stdout, test.stdout)
		}
		if test.stderr != stderr {
			t.Errorf("#%d: _, stderr, _ := ExecCmd(%s) == %s; expected %s", i, test.in, stderr, test.stderr)
		}
		if (err == nil && test.err != "") || (err != nil && err.Error() != test.err) {
			t.Errorf("#%d: _, _, err := ExecCmd(%s) == %v; expected %v", i, test.in, err, test.err)
		}
	}
}
