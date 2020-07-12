package gpg

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

func DeleteSecretKey(fingerprint string) error {
	command := []string{"gpg", "--batch", "--yes", "--delete-secret-key", fingerprint}

	c := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func ImportSecretKey(input io.Reader) error {
	command := []string{"gpg", "--import", "/dev/stdin"}

	c := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	c.Stdin = input
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func Fingerprint(input io.Reader) (string, error) {
	command := []string{"gpg", "--import", "--import-options", "show-only"}

	c := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	c.Stdin = input
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		return "", err
	}

	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) != 40 {
			continue
		}
		regex, _ := regexp.Compile("^[A-Z0-9]+$")
		if regex.MatchString(line) {
			return line, nil
		}
	}

	return "", errors.New("fingerprint not found")
}
