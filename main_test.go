package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_main_unit(t *testing.T) {
	Convey("main", t, func() {
		parts := strings.Split(time.Now().Format("2006-01-02-07:00"), "-")
		y, m, d, z := parts[0], parts[1], parts[2], parts[3]

		testTable := map[string]string{
			"01/01 01:52 pm": fmt.Sprintf("%s-01-01T13:52:00-%s", y, z),       // full
			"1/01 01:00 pm":  fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short day/month
			"01/1 01:00 pm":  fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short day/month
			"1/1 01:00 pm":   fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short day/month
			"01/01 1:00 pm":  fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short hour
			"1/01 1:00 pm":   fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short hour
			"01/1 1:00 pm":   fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short hour
			"1/1 1:00 pm":    fmt.Sprintf("%s-01-01T13:00:00-%s", y, z),       // short hour
			"01:52 pm":       fmt.Sprintf("%s-%s-%sT13:52:00-%s", y, m, d, z), // time
			"1:52 pm":        fmt.Sprintf("%s-%s-%sT13:52:00-%s", y, m, d, z), // time
			"01/01 13:52":    fmt.Sprintf("%s-01-01T13:52:00-%s", y, z),       // 24hr time
			"1/01 13:52":     fmt.Sprintf("%s-01-01T13:52:00-%s", y, z),       // 24hr time
			"01/1 13:52":     fmt.Sprintf("%s-01-01T13:52:00-%s", y, z),       // 24hr time
			"1/1 13:52":      fmt.Sprintf("%s-01-01T13:52:00-%s", y, z),       // 24hr time
			"13:52":          fmt.Sprintf("%s-%s-%sT13:52:00-%s", y, m, d, z), // 24hr time
		}

		for teststr, goodstr := range testTable {
			Convey("Given "+teststr, func() {
				Convey("time should equal "+goodstr, func() {
					t, err := ParseTime(teststr)
					So(err, ShouldBeNil)
					So(t.Format(time.RFC3339), ShouldEqual, goodstr)
				})
			})
		}
	})
}
