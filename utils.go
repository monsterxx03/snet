package main

import (
	exec "os/exec"
	"strings"
)

func Sh(cmds ...string) (result string, err error) {
	cmd := exec.Command("sh", "-c", strings.Join(cmds, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
