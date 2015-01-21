package dal

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_dal_integration(t *testing.T) {
	createTestDB()

	Convey("dal", t, func() {
		// u, err := user.Current()
		// So(err, ShouldBeNil)

		Convey("SavePing()", func() {
			resetTestDB() // These will be run for *every* convey below, which resets the DB between tests
			Reset(func() {
				resetTestDB()
			})

			ip := "127.0.0.1"
			l, _ := time.LoadLocation("UTC")
			startTime := time.Date(2015, time.January, 1, 12, 30, 0, 0, l) // 2015-01-01 12:30:00 +0000 UTC
			responseTime := float32(1.1)

			Convey("Should create one key w/ 1 ping", func() {
				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				keys := getPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("Should create one key when 2 pings are < 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Second) // add a second

				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("Should create 2 keys when 2 pings are > 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Minute) // add a minute

				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
				So(keys[1], ShouldEqual, string(GetPingKey(ip, startTime2)))
			})
		})

		Convey("GetPings()", func() {
			seedDB()

			end := time.Now()
			start := end.Add(-25 * time.Hour)
			groups, err := GetPings("127.0.0.1", start, end, 1*time.Hour)

			So(err, ShouldBeNil)
			So(len(groups), ShouldEqual, 24) // there should be 1 group per hour

			totalPings := 0
			for _, group := range groups {
				totalPings += group.Count
			}

			So(totalPings, ShouldEqual, 86400) // there should 1 ping for every second in a day

			// fmt.Println()
			// for i, g := range groups {
			// 	avg := g.TotalTime / float32(g.Count)
			// 	fmt.Printf("%d: %s, count: %d, avg: %.2f, min: %.2f, max %.2f\n",
			// 		i+1, g.Timestamp.Format(time.RFC3339), g.Count, avg, g.MinTime, g.MaxTime)
			// 	// for _, key := range g.Keys {
			// 	// 	fmt.Printf("key: %s\n", key)
			// 	// }
			// }
		})
	})
}

// seedDB will seed the db with 24 hours of pings for every second
// it adds 1441 rows to the pings_by_minute bucket
func seedDB() {
	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	ip := "127.0.0.1"
	max := float32(15000.0)
	min := float32(5.0)
	timestamp := time.Now().Add(-86400 * time.Second)

	err = db.Update(func(tx *bolt.Tx) error {
		pings := tx.Bucket([]byte("pings_by_minute"))
		if pings == nil {
			return errors.New("Couldn't find pings_by_minute bucket")
		}

		for x := 0; x < 86400; x++ {
			pingStartTime := timestamp.Add(time.Duration(x) * time.Second)
			resTime := rand.Float32()*(max-min) + min

			key := GetPingKey(ip, pingStartTime)
			val, err := SerializePingRes(pingStartTime, resTime)
			if err != nil {
				return err
			}

			v := pings.Get(key)
			if v != nil {
				val = append(v, val...)
			}

			err = pings.Put(key, val)
			if err != nil {
				return fmt.Errorf("Error writing key: %s", err)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
}

func getPingKeys() []string {
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

func DeleteDB() {
	// os.Remove("pinghist.db")
}

var boltBuckets = []string{"hosts", "pings_by_minute"}

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

	db.Close()

	createTestDB()
}
