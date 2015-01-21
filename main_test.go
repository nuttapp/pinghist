package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMain_Integration(t *testing.T) {
	createTestDB()

	Convey("main (integration)", t, func() {
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

			Convey("Should create one key w/ one ping", func() {
				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				keys := getPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("Should create one key when pings are < 1 minute apart", func() {
				startTime2 := startTime.Add(1 * time.Second) // add a second

				err := SavePing(ip, startTime, responseTime)
				So(err, ShouldBeNil)

				err = SavePing(ip, startTime2, responseTime)
				So(err, ShouldBeNil)

				keys := getPingKeys()
				So(keys[0], ShouldEqual, string(GetPingKey(ip, startTime)))
			})
			Convey("Should create 2 keys when pings are > 1 minute apart", func() {
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

// GetPingKey returns a key for the given ip and time, seconds and nanoseconds are removed
// from pingStartTime in order to group pings by minute
func GetPingKey(ip string, pingStartTime time.Time) []byte {
	keyTimestamp := time.Date(pingStartTime.Year(), pingStartTime.Month(),
		pingStartTime.Day(), pingStartTime.Hour(), pingStartTime.Minute(), 0, 0, pingStartTime.Location())

	key := fmt.Sprintf("%s_%s", ip, keyTimestamp.Format(time.RFC3339))

	return []byte(key)
}

const (
	PingResByteCount          = 21 // total bytes
	PingResTimestampByteCount = 15 // time.Time
	PingResTimeByteCount      = 4  // float32
)

// SerializePingRes converts startTime and resTime to a 21 byte array
// startTime is  the time the ping was initated
// resTime is the amount of time it took to return the ping packet
// endTime = startTime + resTime
// Format: 21 bytes
// | 15 bytes  | 1 byte  | 4 bytes | 1 byte
// | startTime | padding | resTime | padding
// TODO Convert to PingRes struct w/ method MarshalBinary()
func SerializePingRes(startTime time.Time, resTime float32) ([]byte, error) {
	buff := make([]byte, PingResByteCount)
	floatBytes := Float32bytes(resTime)
	timeBytes, err := startTime.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("Couldn't marshal startTime as binary: %s", err)
	}

	copy(buff[0:PingResTimestampByteCount], timeBytes)
	responseTimeOffset := PingResTimestampByteCount + 1
	copy(buff[responseTimeOffset:responseTimeOffset+PingResTimeByteCount], floatBytes)

	return buff, nil
}

// DeserializePingRes does the opposite of SerializePingRes
func DeserializePingRes(data []byte) (*time.Time, float32, error) {
	pingTime := &time.Time{}
	if len(data) != PingResByteCount {
		return nil, 0, errors.New("Invalid data length")
	}
	err := pingTime.UnmarshalBinary(data[0:PingResTimestampByteCount])
	if err != nil {
		return nil, 0, errors.New("Couldn't unmarshal bytes to time")
	}

	responseTimeOffset := PingResTimestampByteCount + 1
	resTime := Float32frombytes(data[responseTimeOffset : responseTimeOffset+PingResTimeByteCount])

	return pingTime, resTime, nil
}

func GetPings(ipAddress string, start, end time.Time, groupBy time.Duration) ([]*PingGroup, error) {
	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	groups := make([]*PingGroup, 0, 5)

	err = db.View(func(tx *bolt.Tx) error {
		pings := tx.Bucket([]byte("pings_by_minute"))
		if pings == nil {
			return errors.New("Couldn't find pings_by_minute bucket")
		}
		c := pings.Cursor()

		groupSeconds := groupBy.Seconds()
		min := []byte(ipAddress + "_" + start.Format(time.RFC3339))
		max := []byte(ipAddress + "_" + end.Format(time.RFC3339))
		count := 0
		var group *PingGroup

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) == -1; k, v = c.Next() {
			// keyParts := strings.Split(string(k), "_")

			for i := 0; i < len(v); i += PingResByteCount {
				pingTime, resTime, err := DeserializePingRes(v[i : i+PingResByteCount])
				if err != nil {
					return err
				}

				// on first loop assign the group
				if count == 0 {
					group = NewPingGroup(*pingTime, resTime)
					// group.Keys = append(group.Keys, keyParts[1])
					groups = append(groups, group)
				} else if math.Abs(group.Timestamp.Sub(*pingTime).Seconds()) < groupSeconds { // add to group when it's in the range
					group.TotalTime += resTime
					group.Count++
					if resTime < group.MinTime {
						group.MinTime = resTime
					}
					if resTime > group.MaxTime {
						group.MaxTime = resTime
					}
					// group.Keys = append(group.Keys, keyParts[1])
				} else {
					group = NewPingGroup(*pingTime, resTime)
					// group.Keys = append(group.Keys, keyParts[1])
					groups = append(groups, group)
				}
				count++
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return groups, nil
}

func DeleteDB() {
	// os.Remove("pinghist.db")
}

const BucketNotFoundError = "Could not find bucket"
const KeyNotFoundError = "Could not find key"

func GetPing(id string) (float32, error) {
	if len(id) == 0 {
		return 0, errors.New("id can't be empty")
	}

	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	var pingTime float32
	err = db.View(func(tx *bolt.Tx) error {
		pings := tx.Bucket([]byte("pings"))
		if pings == nil {
			return fmt.Errorf("%s: hosts", BucketNotFoundError)
		}

		pt := pings.Get([]byte(id))
		if pt == nil {
			return errors.New(KeyNotFoundError)
		}

		pingTime = Float32frombytes(pt)
		return nil
	})

	if err != nil {
		return 0, err
	}

	return pingTime, nil
}

func SavePing(ip string, starTime time.Time, responseTime float32) error {
	if len(ip) == 0 {
		return errors.New("ip can't be empty")
	}

	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		pings := tx.Bucket([]byte("pings_by_minute"))
		if pings == nil {
			return fmt.Errorf("%s: pings_by_minute", BucketNotFoundError)
		}

		key := GetPingKey(ip, starTime)
		val, err := SerializePingRes(starTime, responseTime)
		if err != nil {
			return err
		}

		v := pings.Get(key)
		if v != nil {
			// Do not change the byte array that boltdb gives us, make our own new one
			// + the extra room for the next value
			newVal := make([]byte, 0, len(val)+PingResByteCount)
			newVal = append(newVal, v...)
			newVal = append(newVal, val...)
			val = newVal
		}

		err = pings.Put(key, val)
		if err != nil {
			return fmt.Errorf("Error writing key: %s", err)
		}

		return nil
	})

	return err
}

// func SavePingResponse(pr *PingResponse) error {
// 	if len(pr.ID) == 0 {
// 		return errors.New("Pingresponse id can't be empty")
// 	}
//
// 	// Open the my.db data file in your current directory. It will be created if it doesn't exist.
// 	db, err := bolt.Open("pinghist.db", 0600, nil)
// 	defer db.Close()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	err = db.Update(func(tx *bolt.Tx) error {
// 		hosts := tx.Bucket([]byte("hosts"))
// 		if hosts == nil {
// 			return errors.New("Can't find hosts bucket")
// 		}
// 		pings := tx.Bucket([]byte("pings"))
// 		if pings == nil {
// 			return errors.New("Can't find pings bucket")
// 		}
//
// 		err := hosts.Put([]byte(pr.IP), []byte(pr.Host))
// 		if err != nil {
// 			return fmt.Errorf("Error while saving hosts bucket: %s", err)
// 		}
//
// 		err = pings.Put([]byte(pr.ID), Float32bytes(pr.Time))
// 		if err != nil {
// 			return fmt.Errorf("Error while saving to pings bucket: %s", err)
// 		}
//
// 		return nil
// 	})
//
// 	return err
// }

func NewPingGroup(timestamp time.Time, responseTime float32) *PingGroup {
	pg := &PingGroup{
		Timestamp: timestamp,
		TotalTime: responseTime,
		MinTime:   responseTime,
		MaxTime:   responseTime,
		Count:     1,
		keys:      []string{},
	}
	return pg
}

type PingGroup struct {
	Timestamp time.Time
	Count     int // The # of pings in the group
	TotalTime float32
	MaxTime   float32
	MinTime   float32
	keys      []string // used for debugging
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

func Float32frombytes(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func Float32bytes(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}
