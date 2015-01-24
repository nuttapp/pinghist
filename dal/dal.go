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
	Timestamp time.Time
	Received  int // The # of pings in the group
	Timedout  int
	TotalTime float32
	MaxTime   float32
	MinTime   float32
	keys      []string // used for debugging
}

func NewPingGroup(timestamp time.Time, responseTime float32) *PingGroup {
	pg := &PingGroup{
		Timestamp: timestamp,
		TotalTime: responseTime,
		MinTime:   responseTime,
		MaxTime:   responseTime,
		Received:  1,
		keys:      []string{},
	}

	if responseTime == -1.0 {
		pg.Received = 0
		pg.Timedout = 1
	}
	return pg
}

func (pg PingGroup) Avg() float32 {
	return pg.TotalTime / float32(pg.Received)
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
					group.Received++
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
