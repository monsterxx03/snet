package utils

import (
	"bytes"
	"log"
	exec "os/exec"
	"strings"
	"text/template"
)

func Sh(cmds ...string) (result string, err error) {
	cmdStr := strings.Join(cmds, " ")
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

func NamedFmt(msg string, args map[string]interface{}) (string, error) {
	var result bytes.Buffer
	tpl, err := template.New("fmt").Parse(msg)
	if err != nil {
		return "", err
	}
	if err := tpl.Execute(&result, args); err != nil {
		return "", err
	}
	return result.String(), nil
}
