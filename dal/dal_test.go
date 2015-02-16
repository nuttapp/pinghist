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
	Convey("SerializePingRes()", t, func() {
		Convey("Should serialize a ping response", func() {
			startTime := time.Now()
			resTime := float32(1.0)
			bytes := SerializePingRes(startTime, resTime)
			So(len(bytes), ShouldEqual, PingResByteCount)
		})
	})

	Convey("DeserializePingRes()", t, func() {
		Convey("should deserialize a ping response", func() {
			fb := Float32bytes(1.1)
			serializedPingRes := []byte{2, 0x0, fb[0], fb[1], fb[2], fb[3], 0x0}
			secondsOffset, resTime, err := DeserializePingRes(serializedPingRes)
			So(err, ShouldEqual, nil)
			So(secondsOffset, ShouldEqual, 2)
			So(resTime, ShouldEqual, 1.1)
		})
		Convey("should return error with invalid date", func() {
			fb := Float32bytes(1.1)
			serializedPingRes := []byte{250, 0x0, fb[0], fb[1], fb[2], fb[3], 0x0}
			_, _, err := DeserializePingRes(serializedPingRes)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, TimeDeserializationError)
		})
		Convey("should return error with byte length ", func() {
			serializedPingRes := []byte{}
			_, _, err := DeserializePingRes(serializedPingRes)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidByteLength)
		})
	})

}

func Test_dal_integration(t *testing.T) {
	Convey("DAL", t, func() {
		d := NewDAL()
		d.DeleteBuckets()
		d.CreateBuckets()
		Reset(func() {
			os.Remove(d.fileName)
		})

		Convey("SavePing()", func() {
			ip := "127.0.0.1"
			l, _ := time.LoadLocation("UTC")
			startTime := time.Date(2015, time.January, 1, 12, 30, 0, 0, l) // 2015-01-01 12:30:00 +0000 UTC
			responseTime := float32(1.1)

			Convey("should create 1 key w/ 1 ping", func() {
				err := d.SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				keys := getAllPingKeys(d)
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("should create 1 key when 2 pings are < 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Second) // add a second

				err := d.SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = d.SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getAllPingKeys(d)
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("should create 2 keys when 2 pings are > 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Minute) // add a minute

				err := d.SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = d.SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getAllPingKeys(d)
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
				So(keys[1], ShouldEqual, string(GetPingKey(ip, startTime2)))
			})
			Convey("should return error w/ blank IP", func() {
				err := d.SavePing("", time.Now(), 0)
				So(err.Error(), ShouldContainSubstring, IPRequiredError)
			})
			Convey("should return error w/ response time < -1", func() {
				err := d.SavePing(ip, time.Now(), -2.0)
				So(err.Error(), ShouldContainSubstring, ResponseTimeOutOfRangeError)
			})
			Convey("should return error when opening invalid db", func() {
				d := &DAL{}
				err := d.SavePing(ip, time.Now(), 1.0)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("GetPings()", func() {
			ip := "127.0.0.1"
			tfmt := "01/02/06 03:04:05 pm"

			Convey("Should seed db with differnt IPs and not return other IPs", func() {
				ip1 := ip
				ip2 := "167.206.245.222"
				seedTestDB(d, ip1, "01/03/15 04:00:00 pm", "01/03/15 04:02:00 pm")
				seedTestDB(d, ip2, "01/03/15 04:00:00 pm", "01/03/15 04:01:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 04:00:00 pm", time.UTC)
				endti, _ := time.ParseInLocation(tfmt, "01/04/15 04:07:00 pm", time.UTC)
				groupBy := 1 * time.Minute

				groups, err := d.GetPings(ip1, start, endti, groupBy)
				So(err, ShouldBeNil)
				So(groups, ShouldNotBeNil)
				// before this fix it would return 4 groups
				So(len(groups), ShouldEqual, 2)
			})

			Convey("should return 24 groups, 1 hour in each group", func() {
				seedTestDB(d, ip, "01/03/15 04:00:00 pm", "01/04/15 06:00:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 05:00:00 pm", time.UTC)
				endti, _ := time.ParseInLocation(tfmt, "01/04/15 05:00:00 pm", time.UTC)
				groupBy := 1 * time.Hour
				// fmt.Printf("%s - %s\n", start.Format(tfmt), endti.Format(tfmt))

				groups, err := d.GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(len(groups), ShouldEqual, 24)            // there should be 1 group per hour
				So(sumReceived(groups), ShouldEqual, 86400) // there should 1 ping for every second in a day
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
			})
			Convey("should return  4 groups, 15 minutes in each group", func() {
				seedTestDB(d, ip, "01/03/15 03:00:00 pm", "01/03/15 06:00:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 04:00:00 pm", time.UTC)
				endti, _ := time.ParseInLocation(tfmt, "01/03/15 05:00:00 pm", time.UTC)
				groupBy := 15 * time.Minute

				groups, err := d.GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(len(groups), ShouldEqual, 4)
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
				So(sumReceived(groups), ShouldEqual, 3600)
			})
			Convey("should return 60 groups, 1 second in each group", func() {
				seedTestDB(d, ip, "01/03/15 03:00:00 pm", "01/03/15 03:05:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 03:02:00 pm", time.UTC)
				endti, _ := time.ParseInLocation(tfmt, "01/03/15 03:03:00 pm", time.UTC)
				groupBy := 1 * time.Second

				groups, err := d.GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(sumReceived(groups), ShouldEqual, 60)
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
			})
			Convey("should return 30 groups, 1 second in each group", func() {
				seedTestDB(d, ip, "01/03/15 02:00:00 pm", "01/03/15 04:00:00 pm")

				start, _ := time.ParseInLocation(tfmt, "01/03/15 03:00:00 pm", time.UTC)
				endti, _ := time.ParseInLocation(tfmt, "01/03/15 03:00:30 pm", time.UTC)
				groupBy := 1 * time.Second

				groups, err := d.GetPings("127.0.0.1", start, endti, groupBy)
				So(err, ShouldBeNil)
				So(len(groups), ShouldEqual, 30)
				So(sumReceived(groups), ShouldEqual, 30)
				So(groups[0].Start, ShouldHappenOnOrAfter, start)
				So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
			})
			Convey("should return error when it can't find bucket", func() {
				db, err := bolt.Open(d.fileName, 0600, nil)
				So(err, ShouldBeNil)
				err = db.Update(func(tx *bolt.Tx) error {
					for _, name := range d.Buckets() {
						tx.DeleteBucket([]byte(name))
					}
					return nil
				})
				db.Close()
				_, err = d.GetPings("127.0.0.1", time.Now(), time.Now(), 1*time.Second)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, BucketNotFoundError)
			})
			Convey("should return error when it can't open db ", func() {
				d.fileName = ""
				_, err := d.GetPings("127.0.0.1", time.Now(), time.Now(), 1*time.Second)
				So(err, ShouldNotBeNil)
			})
			Convey("should return error when it deserialize key timestamp", func() {
				ip := "127.0.0.1"
				startTime := time.Now()

				db, err := bolt.Open(d.fileName, 0600, nil)
				So(err, ShouldBeNil)
				// add a garbage value to our pings bucket manually
				err = db.Update(func(tx *bolt.Tx) error {
					pings, err := tx.CreateBucketIfNotExists([]byte(d.pingsBucket))
					So(err, ShouldBeNil)
					key := GetPingKey(ip, startTime)
					key = append(key, []byte("break-the-RFC3399-timestamp")...)
					return pings.Put(key, nil)
				})
				db.Close()

				So(err, ShouldBeNil)
				groups, err := d.GetPings(ip, startTime, startTime, 1*time.Second)
				So(groups, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, KeyTimestampParsingError)
			})
			Convey("should return error when it can't deserialize ping response (value of key)", func() {
				ip := "127.0.0.1"
				startTime := time.Now()

				db, err := bolt.Open(d.fileName, 0600, nil)
				So(err, ShouldBeNil)
				// add a garbage value to our pings bucket manually
				err = db.Update(func(tx *bolt.Tx) error {
					pings, err := tx.CreateBucketIfNotExists([]byte(d.pingsBucket))
					So(err, ShouldBeNil)
					key := GetPingKey(ip, startTime)
					val := make([]byte, 25)
					val[0] = 60 // the seconds offset should be between 0-59...
					return pings.Put(key, val)
				})
				db.Close()
				So(err, ShouldBeNil)

				groups, err := d.GetPings(ip, startTime, startTime, 1*time.Second)
				So(groups, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, TimeDeserializationError)
			})
		})

		Convey("SavePingWithTransaction()", func() {
			Convey("should return error when key is too large", func() {
				db, err := bolt.Open(d.fileName, 0600, nil)
				So(err, ShouldBeNil)
				defer db.Close()
				err = db.Update(func(tx *bolt.Tx) error {
					b := make([]byte, 130000)
					largeKey := string(b)
					return d.SavePingWithTransaction(largeKey, time.Time{}, 1.0, tx)
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key too large")
			})
			Convey("should return error when it can't find bucket", func() {
				db, err := bolt.Open(d.fileName, 0600, nil)
				So(err, ShouldBeNil)
				defer db.Close()
				err = db.Update(func(tx *bolt.Tx) error {
					for _, name := range d.Buckets() {
						tx.DeleteBucket([]byte(name))
					}
					return nil
				})
				So(err, ShouldBeNil)

				err = db.Update(func(tx *bolt.Tx) error {
					return d.SavePingWithTransaction("", time.Time{}, 1.0, tx)
				})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, BucketNotFoundError)
			})
		})

	})
}

// func Test_dal_seed(t *testing.T) {
// Convey("seed", t, func() {
// 	fmt.Println()
// 	d := NewDAL()
// 	Convey("should return 30 groups, 1 second in each group", func() {
// 		// fmt.Println()
// 		// l := time.Now().Location()
// 		// tfmt := "01/02/06 03:04:05 pm"
// 		seedTestDB(d, "01/01/15 03:00:00 pm", "01/01/15 03:00:00 pm")
//
// 		// start, _ := time.ParseInLocation(tfmt, "01/03/15 03:00:00 pm", time.UTC)
// 		// endti, _ := time.ParseInLocation(tfmt, "01/03/15 03:00:30 pm", time.UTC)
// 		// start = start.UTC()
// 		// endti = endti.UTC()
// 		// groupBy := 1 * time.Second
// 		//
// 		// groups, err := d.GetPings("127.0.0.1", start, endti, groupBy)
// 		// So(err, ShouldBeNil)
// 		// So(len(groups), ShouldEqual, 30)
// 		// So(sumReceived(groups), ShouldEqual, 30)
// 		// So(groups[0].Start, ShouldHappenOnOrAfter, start)
// 		// So(groups[len(groups)-1].End, ShouldHappenOnOrBefore, endti)
// 	})
// })
// }

func sumReceived(groups []*PingGroup) int {
	totalPings := 0
	for _, group := range groups {
		totalPings += group.Received
	}
	return totalPings
}

// seedTestDB will seed the db every second betwene the given times
func seedTestDB(dal *DAL, ip, startTime, endTime string) {
	const tfmt = "01/02/06 03:04:05 pm"

	db, err := bolt.Open(dal.fileName, 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	maxRes, minRes := float32(1500.0), float32(5.0)
	rand.Seed(time.Now().UnixNano())

	l, _ := time.LoadLocation("UTC")
	start, _ := time.ParseInLocation(tfmt, startTime, l)
	end, _ := time.ParseInLocation(tfmt, endTime, l)
	// start = start.UTC()
	// end = end.UTC()

	err = db.Update(func(tx *bolt.Tx) error {
		// pt == ping timestamp
		for pt := start; pt.Sub(end) != 0; pt = pt.Add(1 * time.Second) {
			resTime := rand.Float32()*(maxRes-minRes) + minRes

			err := dal.SavePingWithTransaction(ip, pt, resTime, tx)
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

func getAllPingKeys(dal *DAL) []string {
	db, err := bolt.Open(dal.fileName, 0600, nil)
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
