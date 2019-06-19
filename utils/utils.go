package utils

import (
	"log"
	exec "os/exec"
	"strings"
)

func Sh(cmds ...string) (result string, err error) {
	cmdStr := strings.Join(cmds, " ")
	log.Println(cmdStr)
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(output))
		return string(output), err
	}
	return string(output), nil
}

func DomainMatch(domain string, patterns []string) bool {
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
