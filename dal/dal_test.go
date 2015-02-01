package dal

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/olekukonko/tablewriter"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_dal_unit(t *testing.T) {
	Convey("dal.PingGroup", t, func() {
		pg := NewPingGroup(time.Now(), time.Now())

		Convey("addResTime()", func() {
			Convey("should increment Received, sum Total, & set Min/Max", func() {
				pg.addResTime(1.1)
				So(pg.Received, ShouldEqual, 1)
				So(pg.MinTime, ShouldEqual, 1.1)
				So(pg.MaxTime, ShouldEqual, 1.1)
				pg.addResTime(1.2)
				So(pg.Received, ShouldEqual, 2)
				So(pg.TotalTime, ShouldEqual, 2.3)
				So(pg.MinTime, ShouldEqual, 1.1)
				So(pg.MaxTime, ShouldEqual, 1.2)
			})
			Convey("should increment TimedOut ", func() {
				pg.addResTime(-1)
				So(pg.Timedout, ShouldEqual, 1)
				pg.addResTime(-1)
				So(pg.Timedout, ShouldEqual, 2)
				So(pg.MinTime, ShouldEqual, 0)
				So(pg.MaxTime, ShouldEqual, 0)
			})
			Convey("should append to resTimes", func() {
				pg.addResTime(1.1)
				So(len(pg.resTimes), ShouldEqual, 1)
			})
			Convey("should not append to resTimes", func() {
				pg.addResTime(-1)
				So(len(pg.resTimes), ShouldEqual, 0)
			})
		})

		Convey("calcAvgAndStdDev()", func() {
			Convey("should calculate Avg and StdDev", func() {
				resTimes := []float64{10.190, 17.039, 14.165, 13.950, 14.488, 14.295, 19.534, 13.865, 12.782,
					19.113, 15.523, 17.922, 18.841, 18.680, 40.791, 13.798, 17.049, 21.680, 18.660, 21.077,
					14.487, 13.538, 13.666, 13.512, 17.300, 13.480, 14.460, 13.860, 15.185, 18.411, 13.789,
					14.262, 13.232, 11.794, 17.672, 15.491, 18.298, 16.718, 15.182, 13.835, 12.196, 13.142,
					15.329, 10.543, 15.527, 18.212, 15.957, 13.989, 13.492, 24.896, 13.535, 9.689, 17.656,
					14.776, 14.508, 12.150, 13.335, 14.171, 10.721, 13.028, 15.609, 14.225, 20.640, 14.229,
					12.222, 10.949, 12.263, 29.830, 12.987, 13.239, 18.613, -1, 15.019, 16.007, 15.599}

				for _, resTime := range resTimes {
					pg.addResTime(resTime)
				}

				pg.calcAvgAndStdDev()
				So(pg.AvgTime, ShouldEqual, 15.674283783783785)
				So(pg.StdDev, ShouldEqual, 4.3960093436202446)
				So(pg.resTimes, ShouldBeNil)
			})
			Convey("should not calculate Avg and StdDev", func() {
				pg.calcAvgAndStdDev()
				So(pg.AvgTime, ShouldEqual, 0)
				So(pg.StdDev, ShouldEqual, 0)
			})
			Convey("should free memory of resTimes array", func() {
				pg.addResTime(1.1)
				pg.addResTime(1.2)
				pg.calcAvgAndStdDev()
				So(pg.resTimes, ShouldBeNil)
			})
		})
	})
}

func Test_dal_integration(t *testing.T) {

	Convey("dal", t, func() {
		createTestDB() // run before every Convey(...)
		Reset(func() {
			os.Remove("pinghist.db")
		})

		Convey("SavePing()", func() {
			ip := "127.0.0.1"
			l, _ := time.LoadLocation("UTC")
			startTime := time.Date(2015, time.January, 1, 12, 30, 0, 0, l) // 2015-01-01 12:30:00 +0000 UTC
			responseTime := float32(1.1)

			Convey("should create 1 key w/ 1 ping", func() {
				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				keys := getAllPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("should create 1 key when 2 pings are < 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Second) // add a second

				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getAllPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("should create 2 keys when 2 pings are > 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Minute) // add a minute

				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getAllPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
				So(keys[1], ShouldEqual, string(GetPingKey(ip, startTime2)))
			})
			Convey("should return error w/ blank IP", func() {
				err := SavePing("", time.Now(), 0)
				So(err.Error(), ShouldEqual, IPRequiredError)
			})
			Convey("should return error w/ response time < -1", func() {
				err := SavePing(ip, time.Now(), -2.0)
				So(err.Error(), ShouldEqual, ResponseTimeOutOfRangeError)
			})
		})

		Convey("GetPings()", func() {
			l := time.Now().Location()
			tfmt := "01/02/06 03:04:05 pm"

			Convey("should return 24 groups, 1 hour in each group", func() {
				seedTestDB("01/03/15 04:00:00 pm", "01/04/15 06:00:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 05:00:00 pm", l)
				endti, _ := time.ParseInLocation(tfmt, "01/04/15 05:00:00 pm", l)
				start = start.UTC()
				endti = endti.UTC()
				groupBy := 1 * time.Hour
				// fmt.Printf("%s - %s\n", start.Format(tfmt), endti.Format(tfmt))

				groups, err := GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(len(groups), ShouldEqual, 24)            // there should be 1 group per hour
				So(sumReceived(groups), ShouldEqual, 86400) // there should 1 ping for every second in a day
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
			})
			Convey("should return  4 groups, 15 minutes in each group", func() {
				seedTestDB("01/03/15 03:00:00 pm", "01/03/15 06:00:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 04:00:00 pm", l)
				endti, _ := time.ParseInLocation(tfmt, "01/03/15 05:00:00 pm", l)
				start = start.UTC()
				endti = endti.UTC()
				groupBy := 15 * time.Minute

				groups, err := GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(len(groups), ShouldEqual, 4)
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
				So(sumReceived(groups), ShouldEqual, 3600) // there should 1 ping for every second in a day
			})
			Convey("should return 60 groups, 1 second in each group", func() {
				seedTestDB("01/03/15 03:00:00 pm", "01/03/15 03:05:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 03:02:00 pm", l)
				endti, _ := time.ParseInLocation(tfmt, "01/03/15 03:03:00 pm", l)
				start = start.UTC()
				endti = endti.UTC()
				groupBy := 1 * time.Second

				groups, err := GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(sumReceived(groups), ShouldEqual, 60)
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
			})
			Convey("should return 30 groups, 1 second in each group", func() {
				seedTestDB("01/03/15 02:00:00 pm", "01/03/15 04:00:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 03:00:00 pm", l)
				endti, _ := time.ParseInLocation(tfmt, "01/03/15 03:00:30 pm", l)
				start = start.UTC()
				endti = endti.UTC()
				groupBy := 1 * time.Second

				groups, err := GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(len(groups), ShouldEqual, 30)
				So(sumReceived(groups), ShouldEqual, 30)
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
			})
		})
	})
}

func sumReceived(groups []*PingGroup) int {
	totalPings := 0
	for _, group := range groups {
		totalPings += group.Received
	}
	return totalPings
}

// seedTestDB will seed the db every second betwene the given times
func seedTestDB(startTime, endTime string) {
	const tfmt = "01/02/06 03:04:05 pm"

	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	ip := "127.0.0.1"
	maxRes, minRes := float32(1500.0), float32(5.0)
	rand.Seed(time.Now().UnixNano())

	l := time.Now().Location()
	start, _ := time.ParseInLocation(tfmt, startTime, l)
	end, _ := time.ParseInLocation(tfmt, endTime, l)
	start = start.UTC()
	end = end.UTC()

	err = db.Update(func(tx *bolt.Tx) error {
		// pt == ping timestamp
		for pt := start; pt.Sub(end) != 0; pt = pt.Add(1 * time.Second) {
			resTime := rand.Float32()*(maxRes-minRes) + minRes

			err := SavePingWithTransaction(ip, pt, resTime, tx)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func getAllPingKeys() []string {
	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	keys := []string{}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("pings_by_minute"))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, string(k))
		}

		return nil
	})

	return keys
}

func writeTable(groups []*PingGroup) {
	fmt.Println("\n")

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

	l := time.Now().Location()

	for _, g := range groups {
		row := []string{
			fmt.Sprintf("%s", g.Start.In(l).Format("01/02 03:04pm")),
			fmt.Sprintf("%.0f ms", g.MinTime),
			fmt.Sprintf("%.0f ms", g.AvgTime),
			fmt.Sprintf("%.0f ms", g.MaxTime),
			fmt.Sprintf("%.0f ms", g.StdDev),
			fmt.Sprintf("%d", g.Received),
			"0",
		}
		table.Append(row)
	}
	table.Render()
}

var boltBuckets = []string{"pings_by_minute"}

// createTestDB will create an empty Bolt DB and buckets
func createTestDB() {
	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	db.Update(func(tx *bolt.Tx) error {
		// create buckets
		for _, bucketName := range boltBuckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}

		return nil
	})
}

func resetTestDB() {
	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	db.Update(func(tx *bolt.Tx) error {
		// create buckets
		for _, name := range boltBuckets {
			tx.DeleteBucket([]byte(name))
		}

		return nil
	})

	db.Update(func(tx *bolt.Tx) error {
		// create buckets
		for _, bucketName := range boltBuckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}

		return nil
	})
}
