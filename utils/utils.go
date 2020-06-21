package utils

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	exec "os/exec"
	"strings"
	"text/template"
	"time"
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

func Pipe(ctx context.Context, src, dst net.Conn, timeout time.Duration) error {
	count := 2
	doneCh := make(chan bool, count)
	errCh := make(chan error, count)
	cp := func(r, w net.Conn) {
		if err := r.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			errCh <- err
			return
		}
		if err := w.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			errCh <- err
			return
		}
		buf := make([]byte, 1024)
	COPY:
		for {
			select {
			case <-ctx.Done():
				break COPY
			default:
				n, err := r.Read(buf)
				if err != nil && err != io.EOF {
					// ignore idle timeout error
					errCh <- err
					break COPY
				}
				if n == 0 {
					break COPY
				}
				if err := r.SetReadDeadline(time.Now().Add(timeout)); err != nil {
					errCh <- err
					break COPY
				}
				_, err = w.Write(buf[:n])
				if err != nil {
					errCh <- err
					break COPY
				}
				if err := w.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
					errCh <- err
					break COPY
				}
			}
		}
		doneCh <- true
	}
	go cp(src, dst)
	go cp(dst, src)
	errs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		select {
		case <-doneCh:
		case err := <-errCh:
			if err, ok := err.(net.Error); ok && !err.Timeout() {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
