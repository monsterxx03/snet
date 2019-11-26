package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	exec "os/exec"
	"strings"
	"sync"
	"syscall"
	"text/template"
)

func Sh(cmds ...string) (result string, err error) {
	cmdStr := strings.Join(cmds, " ")
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
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

func Pipe(src, dst net.Conn) error {
	var wg sync.WaitGroup
	cp := func(r, w net.Conn) {
		defer wg.Done()
		_, err := io.Copy(r, w)
		if err != nil && !errors.Is(err, syscall.EPIPE) {
			if err, ok := err.(net.Error); ok && !err.Timeout() {
				fmt.Println(err)
			}
		}
	}
	wg.Add(2)
	go cp(src, dst)
	go cp(dst, src)
	wg.Wait()
	return nil
}
