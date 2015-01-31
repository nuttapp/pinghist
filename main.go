package main

import (
	"flag"
	"fmt"
	"log"
	"time"
)

var (
	start            string
	end              string
	inputTimeFormats = []string{
		// full
		"01/02 03:04 pm",

		// short day/month
		"1/02 03:04 pm",
		"01/2 03:04 pm",
		"1/2 03:04 pm",

		// short hour
		"01/02 3:04 pm",
		"1/02 3:04 pm",
		"01/2 3:04 pm",
		"1/2 3:04 pm",

		// time
		"03:04 pm",
		"3:04 pm",

		// 24hr time
		"01/02 15:04",
		"01/2 15:04",
		"1/02 15:04",
		"15:04",

		time.RFC3339,
	}
)

const (
	timeFormat      = "01/02 03:04 pm"
	timeShortformat = "03:04 pm"
)

func init() {
	t := time.Now().Add(-1 * time.Hour)
	td := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	defaultSart := td.Format(timeFormat)
	defaultEnd := td.Add(1 * time.Hour).Format(timeFormat)
	const (
		startUsage = "The time to start querying ping times"
		endUsage   = "The time to end querying ping times (inclusive)"
	)

	flag.StringVar(&start, "start", defaultSart, startUsage)
	flag.StringVar(&end, "s", defaultEnd, startUsage+" (shorthand)")

	flag.StringVar(&start, "end", defaultSart, endUsage)
	flag.StringVar(&end, "e", defaultEnd, endUsage+" (shorthand)")
}

func main() {
	// fmt.Printf("start: %s\nend:   %s\n", start, end)
	flag.Parse()

	l := time.Now().Location()
	var t time.Time
	var err error
	for _, f := range inputTimeFormats {
		t, err = time.ParseInLocation(f, start, l)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Fatal("unkown tie format")
	}

	fmt.Printf("start: %s\nend:   %s\n", t.Format(timeFormat), end)
	fmt.Println(t)
	// pr, err := ping.Ping("127.0.0.1")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//
	// fmt.Printf("%#v", pr)

	fmt.Println("END")
}

func parseFormat(dateTime string) {
}
