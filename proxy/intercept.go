package proxy

import (
	"errors"
	"log"
	"net"
	"time"
)

const (
	TLSRecordLayerTypeChangeCipherSpec = 20
	TLSRecordLayerTypeAlert            = 21
	TLSRecordLayerTypeHandShake        = 22
	TLSRecordLayerTypeApplicationData  = 23
	TLSHandshakeTypeClientHello        = 1
)

func parseServerNameFromSNI(data []byte) (string, error) {
	/*
		handshake header(5): content-type(1) + version(2), len (2)
		38: handshake-type(1) + len(2) + version(2) + random:ts(4) + random(28)
	*/
	index := 5 + 38

	index += 1 + int(data[index]) // sension id len(1) + session id

	index += 2 + int(data[index])<<8 + int(data[index+1]) // cipher suites length(2) + sipher suites

	index += 1 + int(data[index]) // compression methods length(1) + compression methods

	if (index + 2) > len(data) {
		return "", errors.New("Not enough bytes to arrive extension length")
	}
	extLen := int(data[index])<<8 + int(data[index+1])
	index += 2 // extension block length
	if (index + extLen) > len(data) {
		return "", errors.New("Not enough bytes to arrive extension block")
	}
	extBlock := data[index : index+extLen]
	index = 0
	// walkthrough extension blocks to SNI block
	for {
		if index > len(extBlock) {
			break
		}
		if int(extBlock[index])<<8+int(extBlock[index+1]) == 0x00 {
			// server name block
			index += 2 // ext type
			index += 2 // extLen
			index += 3 // sni list length(2) + sni type(1)
			snLen := int(extBlock[index])<<8 + int(extBlock[index+1])
			index += 2
			sn := extBlock[index : index+snLen]
			return string(sn), nil
		} else {
			// skip non sni block
			index += 2 // ext type
			extLen := int(extBlock[index])<<8 + int(extBlock[index+1])
			index += 2 + extLen
		}
	}
	return "", errors.New("SNI block not found")
}

type Intercept struct {
	dstHost string
	dstPort int
	conn    *net.TCPConn
}

func NewIntercept(conn *net.TCPConn, dstHost string, dstPort int) *Intercept {
	return &Intercept{dstHost, dstPort, conn}
}

func (i *Intercept) explore(b []byte) {
	if i.dstPort == 443 {
		if b[0] == TLSRecordLayerTypeHandShake && b[5] == TLSHandshakeTypeClientHello {
			serverName, err := parseServerNameFromSNI(b)
			if err != nil {
				log.Println("failed to parse server name:", err)
			}
			log.Printf("https  %s %s", serverName, i.dstHost)
		}
	}
}

func (i *Intercept) Read(b []byte) (int, error) {
	n, err := i.conn.Read(b)
	if err == nil {
		i.explore(b[:n])
	}
	return n, err
}

func (i *Intercept) Write(b []byte) (int, error) {
	n, err := i.conn.Write(b)
	return n, err
}

func (i *Intercept) Close() error {
	return i.conn.Close()
}

func (i *Intercept) LocalAddr() net.Addr {
	return i.conn.LocalAddr()
}

func (i *Intercept) RemoteAddr() net.Addr {
	return i.conn.RemoteAddr()
}

func (i *Intercept) SetDeadline(t time.Time) error {
	return i.conn.SetDeadline(t)
}

func (i *Intercept) SetReadDeadline(t time.Time) error {
	return i.conn.SetReadDeadline(t)
}

func (i *Intercept) SetWriteDeadline(t time.Time) error {
	return i.conn.SetWriteDeadline(t)
}
