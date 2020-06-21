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

	"snet/stats"
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

func Pipe(ctx context.Context, src, remote net.Conn, timeout time.Duration, rxCh, txCh chan *stats.P, dstHost string) error {
	count := 2
	doneCh := make(chan bool, count)
	errCh := make(chan error, count)
	const toRemote = 1
	const toLocal = 0
	p := &stats.P{Host: dstHost}
	cp := func(r, w net.Conn, direction int) {
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
				if direction == toLocal && rxCh != nil {
					p.Rx = uint64(n)
					rxCh <- p
				}
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
				n, err = w.Write(buf[:n])
				if direction == toRemote && txCh != nil {
					p.Tx = uint64(n)
					txCh <- p
				}
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
	go cp(src, remote, toRemote)
	go cp(remote, src, toLocal)
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
