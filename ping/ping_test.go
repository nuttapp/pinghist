package ping

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_ping_unit(t *testing.T) {

	Convey("ping", t, func() {

		Convey("ParsePingResponseLine()", func() {
			tests := []string{
				"64 bytes from 127.0.0.1: icmp_seq=0 ttl=64 time=0.052 ms",                                             // bsd
				"64 bytes from iad23s06-in-f0.1e100.net (74.125.228.32): icmp_seq=1 ttl=46 time=1.76 ms",               // gnu
				"64 bytes from ip-23-229-234-162.ip.secureserver.net (23.229.234.162): icmp_seq=2 ttl=45 time=47.8 ms", // gnu
			}

			results := []PingResponse{
				PingResponse{Host: "", IP: "127.0.0.1", ICMPSeq: 0, TTL: 64, Time: 0.052},
				PingResponse{Host: "", IP: "74.125.228.32", ICMPSeq: 1, TTL: 46, Time: 1.76},
				PingResponse{Host: "", IP: "23.229.234.162", ICMPSeq: 2, TTL: 45, Time: 47.8},
			}

			for i, test := range tests {
				pr, err := ParsePingResponseLine(test)

				So(pr, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(pr.Host, ShouldEqual, results[i].Host)
				So(pr.IP, ShouldEqual, results[i].IP)
				So(pr.ICMPSeq, ShouldEqual, results[i].ICMPSeq)
				So(pr.Time, ShouldEqual, results[i].Time)
			}
		})

		Convey("ParsePingOutput()", func() {

			Convey("Should return PingResponse given a valid reply", func() {
				tests := [][]byte{
					// BSD ping with reply
					[]byte(`PING google.com (167.206.252.108): 56 data bytes
64 bytes from 167.206.252.108: icmp_seq=0 ttl=59 time=13.886 ms

--- google.com ping statistics ---
1 packets transmitted, 1 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 13.886/13.886/13.886/0.000 ms`),
					// GNU ping with reply
					[]byte(`PING google.com (74.125.228.40) 56(84) bytes of data.
64 bytes from iad23s06-in-f8.1e100.net (74.125.228.40): icmp_seq=1 ttl=46 time=1.67 ms

--- google.com ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 1.674/1.674/1.674/0.000 ms`),
				}

				results := []PingResponse{
					PingResponse{Host: "", IP: "167.206.252.108", ICMPSeq: 0, TTL: 59, Time: 13.886},
					PingResponse{Host: "", IP: "74.125.228.40", ICMPSeq: 1, TTL: 46, Time: 1.67},
				}

				for i, test := range tests {
					pr, err := ParsePingOutput(test)
					So(pr, ShouldNotBeNil)
					So(err, ShouldBeNil)
					So(pr.Host, ShouldEqual, results[i].Host)
					So(pr.IP, ShouldEqual, results[i].IP)
					So(pr.ICMPSeq, ShouldEqual, results[i].ICMPSeq)
					So(pr.Time, ShouldEqual, results[i].Time)
				}
			})

			Convey("Should return destination uncreachable error given no reply", func() {
				tests := [][]byte{
					// BSD timeout
					[]byte(`PING msn.com (23.101.196.141): 56 data bytes

--- msn.com ping statistics ---
1 packets transmitted, 0 packets received, 100.0% packet loss
`),
					// GNU timeout
					[]byte(` PING msn.com (23.101.196.141) 56(84) bytes of data.

--- msn.com ping statistics ---
1 packets transmitted, 0 received, 100% packet loss, time 0ms`),
				}
				for _, test := range tests {
					pr, err := ParsePingOutput(test)
					So(pr, ShouldBeNil)
					So(err.Error(), ShouldEqual, DestinationUnreachableError)
				}
			})
		})
	})
}

func Test_ping_integration(t *testing.T) {
	Convey("ping", t, func() {
		Convey("Ping()", func() {
			Convey("Should ping localhost", func() {
				pr, err := Ping("localhost")
				// So(pr.Host, ShouldEqual, "localhost")
				So(err, ShouldBeNil)
				So(pr, ShouldNotBeNil)
			})
			Convey("Should ping 127.0.0.1", func() {
				pr, err := Ping("127.0.0.1")
				// So(pr.Host, ShouldEqual, "localhost")
				// So(pr.IP, ShouldEqual, "localhost")
				So(err, ShouldBeNil)
				So(pr, ShouldNotBeNil)
			})
			Convey("Should ping google", func() {
				pr, err := Ping("google.com")
				So(err, ShouldBeNil)
				So(pr, ShouldNotBeNil)
				// So(pr.Host, ShouldEqual, "google.com")
			})
			Convey("Should return error with invalid host", func() {
				pr, err := Ping("=2lsakjf2k34")
				So(err, ShouldNotBeNil)
				So(pr, ShouldBeNil)
			})
		})

	})
}
