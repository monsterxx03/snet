package sniffer

import (
	"errors"
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
	if data[0] != TLSRecordLayerTypeHandShake || data[5] != TLSHandshakeTypeClientHello {
		return "", errors.New("Not tls client hello packet")
	}

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
		if index >= len(extBlock)-1 {
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
