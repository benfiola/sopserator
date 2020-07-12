package sops

import (
	"bytes"
	"io"
	"os/exec"
)

type DecryptYAMLOptions struct {
	IgnoreMac bool
	Verbose   bool
}

func DecryptYAML(input io.Reader, output io.Writer, options DecryptYAMLOptions) error {
	command := []string{"sops", "--decrypt"}
	if options.IgnoreMac == true {
		command = append(command, "--ignore-mac")
	}
	if options.Verbose == true {
		command = append(command, "--verbose")
	}
	command = append(command, "--input-type", "yaml")
	command = append(command, "--output-type", "yaml")
	command = append(command, "/dev/stdin")

	c := exec.Command(command[0], command[1:]...)
	var stderr bytes.Buffer
	c.Stdin = input
	c.Stdout = output
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}
