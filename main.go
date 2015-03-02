package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/nuttapp/pinghist/dal"
	"github.com/nuttapp/pinghist/ping"
	"github.com/olekukonko/tablewriter"
)

var (
	d                *dal.DAL
	host             string
	showExamples     bool
	start            string
	end              string
	groupBy          string
	ip               string
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
		"1/2 15:04",
		"15:04",

		time.RFC3339,
	}
)

const (
	timeFormat      = "01/02/2006 03:04 pm"
	tableTimeFmt    = "01/02 03:04 pm"
	timeShortformat = "03:04 pm"
)

func init() {
	d = dal.NewDAL()
	d.CreateBuckets()

	const (
		hostUsage         = "The host IP or hostname to ping"
		ipUsage           = "The ip to query"
		showExamplesUsage = "Show example usage"
		startUsage        = "The time to start querying ping times"
		endUsage          = "The time to end querying ping times (all time up to this point)"
		groupUsage        = "The duration by which to group the results, supports (s)econds, (m)inutes, (h)ours"
	)

	flag.BoolVar(&showExamples, "examples", false, showExamplesUsage)

	flag.StringVar(&ip, "ip", "", ipUsage)

	flag.StringVar(&host, "host", "", hostUsage)
	flag.StringVar(&host, "h", "", "-host")

	flag.StringVar(&start, "start", "", startUsage)
	flag.StringVar(&start, "s", "", "-start")

	flag.StringVar(&start, "end", "", endUsage)
	flag.StringVar(&end, "e", "", "-end")

	flag.StringVar(&groupBy, "groupby", "1h", groupUsage)
	flag.StringVar(&groupBy, "g", "", "-groupby")
}

func main() {
	flag.Parse()

	if host != "" {
		PingHost(host)
		return
	}

	if showExamples {
		fmt.Println("examples...")
		return
	}

	var st, et time.Time
	if start == "" && end == "" {
		t := time.Now().Add(-60 * time.Minute)
		mi := math.Floor(float64(t.Minute())/10.0) * 10 // drop the second digit of the minute 1:27 -> 1:20
		st = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), int(mi), 0, 0, t.Location())
		et = time.Now()
	} else {
		var err error
		st, err = ParseTime(start)
		if err != nil {
			log.Fatal("Can't parse start time")
		}

		if end == "*" || end == "" {
			et = time.Now().AddDate(5, 0, 0)
		} else {
			et, err = ParseTime(end)
			if err != nil {
				log.Fatal("Can't parse end time")
			}
		}
	}
	if ip == "" {
		ip = GetLastPingedIP()
	}
	if groupBy == "" {
		groupBy = "10m"
	}

	dur, err := time.ParseDuration(groupBy)
	if err != nil {
		log.Fatal("Can't parse groupby: " + err.Error())
	}

	toText := et.Format(tableTimeFmt)
	if end == "*" || end == "" {
		toText = "*"
	}

	fmt.Printf("\nResults for %s, from %s, to %s, grouped by %s\n\n", ip, st.Format(tableTimeFmt), toText, groupBy)

	groups, err := d.GetPings(ip, st, et, dur)
	if err != nil {
		log.Fatal("Couldn't retreive pings: %s", err)
	}

	WriteTable(groups)
}

func PingHost(host string) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	tick := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-tick.C:
			startTime := time.Now()
			pr, err := ping.Ping(host)

			if err != nil {
				if te, ok := err.(ping.TimeoutError); ok {
					fmt.Println(err)
					err = d.SavePing(te.IP(), startTime, -1)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					log.Fatal(err)
				}
			} else {
				fmt.Printf("%.3f \n", pr.Time)
				err = d.SavePing(pr.IP, startTime, float32(pr.Time))
			}
			if err != nil {
				log.Fatal(err)
			}
		case <-signalChan:
			os.Exit(0)
		}
	}
}

func ParseTime(str string) (time.Time, error) {
	now := time.Now()
	t := time.Time{}
	var err error

	isShortTime := false
	for _, tfmt := range inputTimeFormats {
		t, err = time.ParseInLocation(tfmt, str, time.Local)
		if err == nil {
			if tfmt == "03:04 pm" || tfmt == "3:04 pm" || tfmt == "15:04" {
				isShortTime = true
			}
			break
		}
		// fmt.Printf("%s \n", tfmt)
	}

	// if no year or month or day was given fill those in with todays date
	if isShortTime {
		t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
	} else if t.Year() == 0 { // no year was specified
		t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
	}

	return t, err
}

func GetLastPingedIP() string {
	allIPStats, err := d.GetAllIPStats()
	if err != nil {
		log.Fatal(err)
	}
	sort.Stable(dal.ByLastPingTime(allIPStats))
	ip = allIPStats[len(allIPStats)-1].IP
	return ip
}

func WriteTable(groups []*dal.PingGroup) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Time",
		"min",
		"avg",
		"max",
		"std dev",
		"Received",
		"Lost",
	})

	table.SetBorder(false) // Set Border to false
	table.SetAlignment(tablewriter.ALIGN_RIGHT)

	for _, g := range groups {
		row := []string{
			fmt.Sprintf("%s", g.Start.In(time.Local).Format(tableTimeFmt)),
			fmt.Sprintf("%.0f ms", g.MinTime),
			fmt.Sprintf("%.0f ms", g.AvgTime),
			fmt.Sprintf("%.0f ms", g.MaxTime),
			fmt.Sprintf("%.0f ms", g.StdDev),
			fmt.Sprintf("%d", g.Received),
			fmt.Sprintf("%d", g.Timedout),
		}
		table.Append(row)
	}
	table.Render()
}
