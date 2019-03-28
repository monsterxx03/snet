package main

import (
	"testing"
)

func TestDMSMsg(t *testing.T) {
	// query baidu.com's A record
	query := []byte{
		25, 190, 1, 32, 0, 1, 0, 0, 0, 0, 0, 1, 5, 98, 97, 105, 100, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 0, 0, 41, 16, 0, 0, 0, 0, 0, 0, 12, 0, 10, 0, 8, 153, 50, 127, 128, 9, 66, 231, 17,
	}
	// query response
	resp := []byte{
		25, 190, 129, 128, 0, 1, 0, 2, 0, 0, 0, 0, 5, 98, 97, 105, 100, 117, 3, 99, 111, 109, 0, 0, 1, 0, 1, 192, 12, 0, 1, 0, 1, 0, 0, 1, 44, 0, 4, 123, 125, 114, 144, 192, 12, 0, 1, 0, 1, 0, 0, 1, 44, 0, 4, 220, 181, 57, 216,
	}
	expected := []*DNSMsg{
		&DNSMsg{ID: 6590, qr: 0, QDCount: 1, ANCount: 0,
			QDomain: "baidu.com", QType: 1, QClass: 1,
			ARecords: []*ARecord{},
		},
		&DNSMsg{ID: 6590, qr: 1, QDCount: 1, ANCount: 2,
			QDomain: "baidu.com", QType: 1, QClass: 1,
			ARecords: []*ARecord{
				NewARecord([]byte{123, 125, 114, 144}, 300),
				NewARecord([]byte{220, 181, 57, 216}, 300),
			},
		},
	}

	queryMsg, err := NewDNSMsg(query)
	if err != nil {
		t.Error("Failed to parse query msg", err)
	}
	if err := queryMsg.Equal(expected[0]); err != nil {
		t.Error("Invalid query msg", err)
	}

	respMsg, err := NewDNSMsg(resp)
	if err != nil {
		t.Error("Failed to parse resp msg", err)
	}
	if err := respMsg.Equal(expected[1]); err != nil {
		t.Error("Invalid resp msg", err)
	}
}
