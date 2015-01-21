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

// Ping will run the ping command and send 1 ping packet to the given hostOrIP
func Ping(hostOrIP string) (*PingResponse, error) {
	output, err := exec.Command("ping", "-c", "1", hostOrIP).CombinedOutput()
	if err != nil {
		return nil, errors.New(string(output))
	}

	pr, err := ParsePingOutput(output)
	if err != nil {
		return nil, err
	}

	// Lookup the hostname if we were provided an IP address
	if ipRegex.MatchString(hostOrIP) {
		hostname, err := net.LookupAddr(hostOrIP)
		if err != nil {
			fmt.Println(err)
		} else {
			pr.Host = hostname[0]
		}
	} else {
		pr.Host = hostOrIP
	}
	return pr, nil
}
