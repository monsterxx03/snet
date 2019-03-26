package main

import (
	exec "os/exec"
	"strings"
)

func Sh(cmds ...string) (result string, err error) {
	cmdStr := strings.Join(cmds, " ")
	LOG.Debug(cmdStr)
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		LOG.Debug(string(output))
		return string(output), err
	}
	return string(output), nil
}
