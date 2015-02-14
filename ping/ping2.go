// Based on https://github.com/golang/go/blob/master/src/net/ipraw_test.go#L95
// Copyright 2009 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// https://github.com/golang/go/blob/master/LICENSE

package ping

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// Timeout sets the ping timeout in milliseconds
var TimeOut = 3000 * time.Millisecond

// Ping sends a ping command to a given host, returns whether is host answers or not
func Ping2(host string) (up bool, err error) {

	// Don't panic, just return nil
	defer func() {
		if r := recover(); r != nil {
			up = false
			err = r.(error)
			return
		}
	}()

	if os.Geteuid() != 0 {
		log.Fatal("skipping ping, root perimisisons missing")
	}

	c, err := net.Dial("ip:icmp", host)
	if err != nil {
		return false, err
	}

	c.SetDeadline(time.Now().Add(TimeOut))
	defer c.Close()

	xid, xseq := os.Getpid()&0xffff, 1
	// b, err := (&icmpMessage{
	// 	Type: icmpv4EchoRequest,
	// 	Code: 0,
	// 	Body: &icmpEcho{
	// 		ID: xid, Seq: xseq,
	// 		Data: []byte("ping.gg.ping.gg.ping.gg"),
	// 	},
	// }).Marshal()
	dataBytes := make([]byte, 64)
	// dataBytes := []byte("ping.gg.ping.gg.ping.gg")
	b, err := ICMPMsg(icmpv4EchoRequest, 0, xid, xseq, dataBytes)
	if err != nil {
		return false, err
	}

	start := time.Now()
	if _, err := c.Write(b); err != nil {
		return false, err
	}

	if _, err := c.Read(b); err != nil {
		return false, err
	}
	end := time.Now()
	ms := float64(end.Sub(start)) / float64(time.Millisecond)
	fmt.Printf("%.3f ms\n", ms)

	b = ipv4Payload(b)

	var m *icmpMessage
	if m, err = parseICMPMessage(b); err != nil {
		return false, err
	}

	if m.Type != icmpv4EchoReply && m.Type != icmpv6EchoReply {
		return false, errors.New("invalid reply type")
	}

	switch p := m.Body.(type) {
	case *icmpEcho:
		if p.ID != xid {
			fmt.Printf("err: id: %d, xid: %d\n", p.ID, xid)
			return false, errors.New("invalid reply indentifier")
		}
		if p.Seq != xseq {
			return false, errors.New("invalid reply sequence")
		}
		return true, nil // UP!
	default:
		return false, errors.New("invalid reply payload")
	}
}

func ipv4Payload(b []byte) []byte {
	if len(b) < 20 {
		return b
	}
	hdrlen := int(b[0]&0x0f) << 2
	return b[hdrlen:]
}

const (
	icmpv4EchoRequest = 8
	icmpv4EchoReply   = 0
	icmpv6EchoRequest = 128
	icmpv6EchoReply   = 129
)

// icmpMessage represents an ICMP message.
type icmpMessage struct {
	Type     int             // type
	Code     int             // code
	Checksum int             // checksum
	Body     icmpMessageBody // body
}

// icmpMessageBody represents an ICMP message body.
type icmpMessageBody interface {
	Len() int
	Marshal() ([]byte, error)
}

func ICMPMsg(msgType, code, id, seq int, data []byte) ([]byte, error) {
	b := []byte{byte(msgType), byte(code), 0, 0}

	bodyBytes := make([]byte, 4+len(data))
	bodyBytes[0], bodyBytes[1] = byte(id>>8), byte(id&0xff)
	bodyBytes[2], bodyBytes[3] = byte(seq>>8), byte(seq&0xff)
	copy(bodyBytes[4:], data)
	b = append(b, bodyBytes...)

	switch msgType {
	case icmpv6EchoRequest, icmpv6EchoReply:
		return b, nil
	}
	csumcv := len(b) - 1 // checksum coverage
	s := uint32(0)
	for i := 0; i < csumcv; i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
	}
	if csumcv&1 == 0 {
		s += uint32(b[csumcv])
	}
	s = s>>16 + s&0xffff
	s = s + s>>16
	// Place checksum back in header; using ^= avoids the
	// assumption the checksum bytes are zero.
	b[2] ^= byte(^s & 0xff)
	b[3] ^= byte(^s >> 8)
	return b, nil
}

// Marshal returns the binary enconding of the ICMP echo request or
// reply message m.
func (m *icmpMessage) Marshal() ([]byte, error) {
	b := []byte{byte(m.Type), byte(m.Code), 0, 0}
	if m.Body != nil && m.Body.Len() != 0 {
		mb, err := m.Body.Marshal()
		if err != nil {
			return nil, err
		}
		b = append(b, mb...)
	}
	switch m.Type {
	case icmpv6EchoRequest, icmpv6EchoReply:
		return b, nil
	}
	csumcv := len(b) - 1 // checksum coverage
	s := uint32(0)
	for i := 0; i < csumcv; i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
	}
	if csumcv&1 == 0 {
		s += uint32(b[csumcv])
	}
	s = s>>16 + s&0xffff
	s = s + s>>16
	// Place checksum back in header; using ^= avoids the
	// assumption the checksum bytes are zero.
	b[2] ^= byte(^s & 0xff)
	b[3] ^= byte(^s >> 8)
	return b, nil
}

// parseICMPMessage parses b as an ICMP message.
func parseICMPMessage(b []byte) (*icmpMessage, error) {
	msglen := len(b)
	if msglen < 4 {
		return nil, errors.New("message too short")
	}
	m := &icmpMessage{Type: int(b[0]), Code: int(b[1]), Checksum: int(b[2])<<8 | int(b[3])}
	if msglen > 4 {
		var err error
		switch m.Type {
		case icmpv4EchoRequest, icmpv4EchoReply, icmpv6EchoRequest, icmpv6EchoReply:
			m.Body, err = parseICMPEcho(b[4:])
			if err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

// imcpEcho represents an ICMP echo request or reply message body.
type icmpEcho struct {
	ID   int    // identifier
	Seq  int    // sequence number
	Data []byte // data
}

func (p *icmpEcho) Len() int {
	if p == nil {
		return 0
	}
	return 4 + len(p.Data)
}

// Marshal returns the binary encoding of the ICMP echo request or
// reply message body p.
func (p *icmpEcho) Marshal() ([]byte, error) {
	b := make([]byte, 4+len(p.Data))
	b[0], b[1] = byte(p.ID>>8), byte(p.ID&0xff)
	b[2], b[3] = byte(p.Seq>>8), byte(p.Seq&0xff)
	copy(b[4:], p.Data)
	return b, nil
}

// parseICMPEcho parses b as an ICMP echo request or reply message
// body.
func parseICMPEcho(b []byte) (*icmpEcho, error) {
	bodylen := len(b)
	p := &icmpEcho{ID: int(b[0])<<8 | int(b[1]), Seq: int(b[2])<<8 | int(b[3])}
	if bodylen > 4 {
		p.Data = make([]byte, bodylen-4)
		copy(p.Data, b[4:])
	}
	return p, nil
}
