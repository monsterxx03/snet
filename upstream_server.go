package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"snet/config"
	"snet/utils"
)

func runTLSServer(c *config.Config) {
	if c.UpstreamTLSToken == "" {
		exitOnError(errors.New("missing upstream-tls-token"), nil)
	}
	cert, err := tls.LoadX509KeyPair(c.UpstreamTLSCRT, c.UpstreamTLSKey)
	exitOnError(err, nil)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", c.UpstreamTLSServerListen, tlsCfg)
	exitOnError(err, nil)
	l.Info("TLS server running:", c.UpstreamTLSServerListen)
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			l.Error(err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			b := make([]byte, 2)
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			tlen := binary.BigEndian.Uint16(b)
			b = make([]byte, int(tlen))
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			if string(b) != c.UpstreamTLSToken {
				l.Error("invalid token", string(b))
				return
			}

			b = make([]byte, 2)
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			hlen := binary.BigEndian.Uint16(b)
			b = make([]byte, int(hlen))
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			host := string(b)
			b = make([]byte, 2)
			if _, err := conn.Read(b); err != nil {
				l.Error(err)
				return
			}
			port := int(binary.BigEndian.Uint16(b))
			dstConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
			if err != nil {
				l.Error(err)
				return
			}
			defer dstConn.Close()
			if err := utils.Pipe(context.Background(), conn, dstConn, time.Duration(30)*time.Second); err != nil {
				l.Error(err)
			}
		}(conn)
	}
}
