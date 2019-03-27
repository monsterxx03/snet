package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
)

type RType uint16

func (t RType) String() string {
	switch t {
	case 1:
		return "A"
	case 2:
		return "NS"

	case 5:
		return "CNAME"
	case 6:
		return "SOA"
	case 12:
		return "PTR"
	case 15:
		return "MX"
	case 16:
		return "TXT"
	case 28:
		return "AAAA"
	case 33:
		return "SRV"
	default:
		return fmt.Sprintf("UNKNOWN %d", t)
	}
}

type ARecord struct {
	IP  net.IP
	TTL uint32
}

func (a *ARecord) String() string {
	return fmt.Sprintf("ip: %v, ttl: %d", a.IP, a.TTL)
}

func NewARecord(ip []byte, ttl uint32) *ARecord {
	return &ARecord{net.IPv4(ip[0], ip[1], ip[2], ip[3]), ttl}
}

type DNSMsg struct {
	ID       uint16
	qr       int    // query or answer
	QDCount  uint16 // query count
	ANCount  uint16 // answer count
	QDomain  string // query domain
	QType    RType  // query type
	QClass   uint16
	ARecords []*ARecord // returned A record list
}

func (m *DNSMsg) String() string {
	t := "query"
	if m.qr == 1 {
		t = "answer"
	}
	return fmt.Sprintf(`%s id:%d qdcount:%d ancount: %d  query: %s qtype: %s qclass: %d arecords: %s`,
		t, m.ID, m.QDCount, m.ANCount, m.QDomain, m.QType, m.QClass, m.ARecords)
}

func NewDNSMsg(data []byte) (*DNSMsg, error) {
	id := binary.BigEndian.Uint16(data[:2])
	isAnswer := false
	if data[2]&0x80 == 0x80 {
		isAnswer = true
	}
	qdcount := binary.BigEndian.Uint16(data[4:6])
	// multi query is allowed(qdcount >1), but I didn't see this in real life(just don't handle it)!
	if qdcount > 1 {
		LOG.Err("multi query found,don't support")
		return nil, errors.New("mutil query in a dns message found")
	}
	ancount := binary.BigEndian.Uint16(data[6:8])
	body := data[12:]
	queryDomainLabels := []string{}
	var qtype uint16
	var qclass uint16
	for {
		// parse question
		offset := 0
		labelLen := int(body[0])
		offset++ // 1 byte label length
		label := body[offset : offset+labelLen]
		offset += labelLen
		queryDomainLabels = append(queryDomainLabels, string(label))
		if body[offset+1] == 0x0 {
			// finish parse question domain
			body = body[offset+1:]
			qtype = binary.BigEndian.Uint16(body[:2])
			body = body[2:]
			qclass = binary.BigEndian.Uint16(body[:2])
			body = body[2:]
			break
		}
		// continue to next label
		body = body[offset:]
	}
	arecords := []*ARecord{}
	if isAnswer {
		// parse answer
		if len(body) < 12 {
			// at least Pointer(2) + Type(2) + Class(2) + TTL(4) + RLEN(2) = 12
			return nil, errors.New("Bad answer message")
		}
		for i := 0; i < int(ancount); i++ {
			c1 := body[0]
			c2 := body[1]
			body = body[2:]
			switch c1 & 0xC0 {
			case 0x0:
				if c1 != 0 || c2 != 0 {
					// Normal dns server's response NAME field always a pointer to existing data(NAME in QUESTION)
					// if top 2 bytes not equal 0, means it's an uncompressed response
					return nil, errors.New("Maybe this dns server didn't compress response data?")
				}
			case 0xC0:
				atype := RType(binary.BigEndian.Uint16(body[:2]))
				body = body[4:] // skip TYPE + CLASS
				ttl := binary.BigEndian.Uint32(body[:4])
				body = body[4:] // skip ttl
				rdLen := binary.BigEndian.Uint16(body[:2])
				body = body[2:] // skip rdLen
				if atype.String() != "A" {
					// only intereseted in A type, skip left rdata
					body = body[rdLen:]
					continue
				}
				rdata := body[:rdLen]
				arecords = append(arecords, NewARecord(rdata, ttl))
				body = body[rdLen:] // skip rdata
			default:
				// 0x80 and 0x40 are reversed bits, shouldn't appear here
				return nil, errors.New("bad message")
			}
		}
	}
	qr := 0
	if isAnswer {
		qr = 1
	}
	return &DNSMsg{
		ID: id, QDCount: qdcount, ANCount: ancount,
		qr:      qr,
		QDomain: strings.Join(queryDomainLabels, "."),
		QType:   RType(qtype), QClass: qclass,
		ARecords: arecords}, nil
}
