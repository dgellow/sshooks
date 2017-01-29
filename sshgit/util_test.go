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
