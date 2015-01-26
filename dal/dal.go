package dal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/boltdb/bolt"
)

const BucketNotFoundError = "Could not find bucket"
const KeyNotFoundError = "Could not find key"

type PingGroup struct {
	Timestamp     time.Time
	Received      int // The # of pings in the group
	Timedout      int
	TotalTime     float64
	AvgTime       float64
	StdDev        float64
	MaxTime       float64
	MinTime       float64
	keys          []string  // used for debugging
	groupResTimes []float64 // used for calculating std dev
}

func (pg *PingGroup) addPingResTime(resTime float64) {
	if resTime >= 0 {
		pg.TotalTime += resTime
		pg.Received++
		if resTime < pg.MinTime {
			pg.MinTime = resTime
		}
		if resTime > pg.MaxTime {
			pg.MaxTime = resTime
		}
		pg.groupResTimes = append(pg.groupResTimes, resTime)
	} else {
		pg.Timedout++
	}
}

func (pg *PingGroup) calcAvgAndStdDev() {
	// calc std dev for the group before creating a new one
	// https://www.khanacademy.org/math/probability/descriptive-statistics/variance_std_deviation/v/population-standard-deviation
	avgPingResTime := pg.TotalTime / float64(pg.Received)
	sumDiffSq := 0.0
	for i := 0; i < len(pg.groupResTimes); i++ {
		resTime := pg.groupResTimes[i]
		// ignore timeouts (-1)
		if resTime > 0 {
			sumDiffSq += math.Pow(resTime-avgPingResTime, 2)
		}
	}

	pg.StdDev = math.Sqrt(sumDiffSq / float64(len(pg.groupResTimes)))
	pg.AvgTime = avgPingResTime
	pg.groupResTimes = nil // free this mem
}

func NewPingGroup(timestamp time.Time) *PingGroup {
	pg := &PingGroup{
		Timestamp:     timestamp,
		TotalTime:     0,
		MinTime:       0,
		MaxTime:       0,
		Received:      0,
		keys:          []string{},
		groupResTimes: []float64{},
	}
	return pg
}

func SavePingWithTransaction(ip string, starTime time.Time, responseTime float32, tx *bolt.Tx) error {
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
		return SavePingWithTransaction(ip, starTime, responseTime, tx)
	})

	return err
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
func DeserializePingRes(data []byte) (*time.Time, float64, error) {
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

	return pingTime, float64(resTime), nil
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
		// hold on to the pings so we can calculate std dev for a group
		groupResTimes := make([]float64, 0)

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) >= -1; k, v = c.Next() {
			// keyParts := strings.Split(string(k), "_")

			for i := 0; i < len(v); i += PingResByteCount {
				pingTime, resTime, err := DeserializePingRes(v[i : i+PingResByteCount])
				if err != nil {
					return err
				}

				// on first loop assign the group
				if count == 0 {
					group = NewPingGroup(*pingTime)
					// group.Keys = append(group.Keys, keyParts[1])
					groups = append(groups, group)
					group.addPingResTime(resTime)

				} else if math.Abs(group.Timestamp.Sub(*pingTime).Seconds()) < groupSeconds { // add to group when it's in the range
					group.addPingResTime(resTime)
					// group.Keys = append(group.Keys, keyParts[1])

				} else { // start a new group
					group.calcAvgAndStdDev()

					group = NewPingGroup(*pingTime)
					// group.Keys = append(group.Keys, keyParts[1])
					groups = append(groups, group)
					if resTime > 0 {
						groupResTimes = append(groupResTimes, resTime)
					}
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
