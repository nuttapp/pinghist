package ping

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const DestinationUnreachableError = "Destination unreachable"

var (
	ipRegex   = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	hostRegex = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
)

type PingResponse struct {
	ID      string
	Host    string
	IP      string
	TTL     int
	Time    float64
	ICMPSeq int
}

// ParsePingResponseLine parses a successful ping reply and returns a corresponding PingResponse struct
func ParsePingResponseLine(s string) (*PingResponse, error) {
	parts := strings.Split(s, " ")

	pr := &PingResponse{}
	for _, part := range parts {
		part = strings.Trim(part, ":()")

		if part == "64" || part == "from" || part == "bytes" || part == "ms" {
			continue
		}

		if strings.HasPrefix(part, "icmp_seq") {
			icmpParts := strings.Split(part, "=")
			if len(icmpParts) != 2 {
				return nil, errors.New("Unexpected # of parts found while parsing icmp_seq")
			}

			icmpSeq, err := strconv.Atoi(icmpParts[1])
			if err != nil {
				return nil, errors.New("Failed parsing icmp_seq:" + err.Error())
			}
			pr.ICMPSeq = icmpSeq
		} else if strings.HasPrefix(part, "ttl") {
			ttlParts := strings.Split(part, "=")
			if len(ttlParts) != 2 {
				return nil, errors.New("Unexpected # of parts found while parsing ttl")
			}

			ttl, err := strconv.Atoi(ttlParts[1])
			if err != nil {
				return nil, errors.New("Failed parsing ttl:" + err.Error())
			}
			pr.TTL = ttl
		} else if strings.HasPrefix(part, "time") {
			timeParts := strings.Split(part, "=")
			if len(timeParts) != 2 {
				return nil, errors.New("Unexpected # of parts found while parsing time")
			}

			time, err := strconv.ParseFloat(timeParts[1], 64)
			if err != nil {
				return nil, errors.New("Failed parsing time:" + err.Error())
			}
			pr.Time = time
		} else if ipRegex.MatchString(part) {
			pr.IP = part
		}
	}
	return pr, nil
}

// ParsePingOutput will parse the entire output of a ping command.
// If there is more than one reply only the first one is parsed.
// If there is no reply it returns a DestinationUnreachableError
func ParsePingOutput(res []byte) (*PingResponse, error) {
	lines := strings.Split(string(res), "\n")
	if len(lines[1]) == 0 {
		return nil, errors.New(DestinationUnreachableError)
	}

	pr, err := ParsePingResponseLine(lines[1])
	if err != nil {
		return nil, err
	}
	return pr, nil
}

type TimeoutError interface {
	Timeout() bool
	IP() string
}

type PingError struct {
	ip        string
	msg       string
	Output    string
	IsTimeout bool
}

func (pe PingError) IP() string {
	return pe.ip
}

func (pe PingError) Error() string {
	return pe.msg
}

func (pe PingError) Timeout() bool {
	return pe.IsTimeout
}

// Ping will run the ping command and send 1 ping packet to the given hostOrIP
// TODO: Add ipv6 regex
func Ping(hostOrIP string) (*PingResponse, error) {
	var ip string
	if hostOrIP == "localhost" || hostOrIP == "::1" || hostOrIP == "fe80::1%lo0" {
		ip = "127.0.0.1"
	} else if ipRegex.MatchString(hostOrIP) {
		ip = hostOrIP
	} else {
		addrs, err := net.LookupHost(hostOrIP)
		if err != nil {
			return nil, fmt.Errorf("ping.Ping: %s", err)
		}
		ip = addrs[0]
	}

	output, err := exec.Command("ping", "-c", "1", ip).CombinedOutput()
	if err != nil {
		err = &PingError{
			ip:        ip,
			IsTimeout: true,
			Output:    string(output),
			msg:       err.Error(),
		}
		return nil, err
	}

	pr, err := ParsePingOutput(output)
	if err != nil {
		return nil, err
	}

	pr.IP = hostOrIP
	pr.Host = hostOrIP
	return pr, nil
}

// Ping will run the ping command and send 1 ping packet to the given hostOrIP
func PingNative(hostOrIP string) (*PingResponse, error) {
	_, ms, err := Ping2(hostOrIP)
	if err != nil {
		return nil, err
	}

	pr := &PingResponse{
		IP:   hostOrIP,
		Host: hostOrIP,
		Time: ms,
	}

	return pr, nil
}
