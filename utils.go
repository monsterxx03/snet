package main

import (
	"fmt"
	exec "os/exec"
	"strconv"
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

func printBytes(data []byte) {
	debug := []string{}
	for _, b := range data {
		debug = append(debug, strconv.Itoa(int(b)))
	}
	fmt.Println("[", strings.Join(debug, ","), "]")
}

func domainMatch(domain string, patterns []string) bool {
	for _, p := range patterns {
		if strings.HasPrefix(p, "*") {
			parts := strings.SplitAfter(p, "*.")
			if len(parts) > 2 {
				panic("invalid pattern:" + p)
			}
			if strings.HasSuffix(domain, parts[1]) {
				return true
			}
		} else if domain == p {
			return true
		}
	}
	return false
}
