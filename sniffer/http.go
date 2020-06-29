package sniffer

import (
	"bytes"
	"errors"
	"strings"
)

var invalidHttpErr = errors.New("invalid http protocol")

const (
	minFirstLineLen = 14 // GET / HTTP/1.1
)

func parseServerNameFromHTTPHeader(data []byte) (string, error) {
	if len(data) < minFirstLineLen+2 { // \r\n
		return "", invalidHttpErr
	}
	bs := bytes.Split(data, []byte{'\r', '\n'})
	for _, b := range bs {
		if len(b) < 6 { // Host: x
			continue
		}
		l := strings.ToLower(string(b))
		kv := strings.Split(l, ":")
		if len(kv) != 2 {
			continue
		}
		if strings.TrimSpace(kv[0]) != "host" {
			continue
		}
		return strings.TrimSpace(kv[1]), nil
	}
	return "", errors.New("no host header found")
}
