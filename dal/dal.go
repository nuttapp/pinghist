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
	Start     time.Time
	End       time.Time
	Received  int     // # of ping packets received
	Timedout  int     // # packets timed out
	TotalTime float64 // sum of resTime of all received
	AvgTime   float64 // TotalTime / Recieved
	StdDev    float64 // for AvgTime
	MaxTime   float64
	MinTime   float64
	keys      []string  // used for debugging
	resTimes  []float64 // Response times, used to calc std dev, nil after calling calcAvgAndStdDev()
}

// addResTime will add a ping response time to group
func (pg *PingGroup) addResTime(resTime float64) {
	if resTime >= 0 {
		pg.TotalTime += resTime
		pg.Received++
		if pg.MinTime == 0 || resTime < pg.MinTime {
			pg.MinTime = resTime
		}
		if resTime > pg.MaxTime {
			pg.MaxTime = resTime
		}
		pg.resTimes = append(pg.resTimes, resTime)
	} else {
		pg.Timedout++
	}
}

// calc std dev for the group before creating a new one
// https://www.khanacademy.org/math/probability/descriptive-statistics/variance_std_deviation/v/population-standard-deviation
func (pg *PingGroup) calcAvgAndStdDev() {
	if pg.TotalTime == 0 {
		pg.StdDev = 0
		pg.AvgTime = 0
		return
	}

	avgPingResTime := pg.TotalTime / float64(pg.Received)
	sumDiffSq := 0.0
	for i := 0; i < pg.Received; i++ {
		resTime := pg.resTimes[i]
		sumDiffSq += math.Pow(resTime-avgPingResTime, 2)
	}

	pg.StdDev = math.Sqrt(sumDiffSq / float64(pg.Received))
	pg.AvgTime = avgPingResTime
	pg.resTimes = nil // free this mem
}

func NewPingGroup(start, end time.Time) *PingGroup {
	pg := &PingGroup{
		Start:     start,
		End:       end,
		TotalTime: 0,
		MinTime:   0,
		MaxTime:   0,
		Received:  0,
		keys:      []string{},
		resTimes:  []float64{},
	}
	return pg
}

type DAL struct {
	path     string
	fileName string
}

func NewDAL() *DAL {
	dal := &DAL{
		path:     "",
		fileName: "pinghist.db",
	}
	return dal
}

// SavePingWithTransaction will save a ping to bolt using the given bolt transaction
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
		// Don't change the byte array that boltdb gives us, make our own new one
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

// SavePing Errors
const (
	IPRequiredError             = "IP can't be empty"
	ResponseTimeOutOfRangeError = "Response time must be >= -1"
)

// SavePing will save a ping to bolt
// Pings are keyed by minute, so, every minute can store a max of 60 pings (1 p/sec)
// The pings within a minute are stored as an array of bytes for fast
// serialization/deserialization and to minimize the size of the value (see SerializePingRes)
func SavePing(ip string, starTime time.Time, responseTime float32) error {
	if len(ip) == 0 {
		return errors.New(IPRequiredError)
	}
	if responseTime < -1 {
		return errors.New(ResponseTimeOutOfRangeError)
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
	PingResByteCount          = 21 // total bytes = time bytes + 1 + float32 bytes + 1
	PingResTimestampByteCount = 15 // time.Time
	PingResTimeByteCount      = 4  // float32
)

// SerializePingRes converts startTime and resTime to a 21 byte array
// startTime is the time the ping was initated
// resTime is the amount of time it took to return the ping packet
// endTime = startTime + resTime
// Format: 21 bytes
// | 15 bytes  | 1 byte  | 4 bytes | 1 byte
// | startTime | padding | resTime | padding
// TODO Convert to PingRes struct w/ method MarshalBinary()
// TODO Remove serialization of datetime entirely, use a single byte as an offset in seconds
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

// GetPings returns pings between the start time and end time, for the given IP,
// grouped by the given duration.
// Start and end time should be in UTC
// gruupBy can be any valid time.Duration, ex: 1 * time.Hour
// Returns a summary for each PingGroup with avg and std deviation
func GetPings(ipAddress string, start, end time.Time, groupBy time.Duration) ([]*PingGroup, error) {
	db, err := bolt.Open("pinghist.db", 0600, nil)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	groups := make([]*PingGroup, 0, 5)
	// fmt.Printf("GetPings() %s - %s\n", start.Format("01/02/06 3:04:05 pm"), end.Format("01/02/06 3:04:05 pm"))

	err = db.View(func(tx *bolt.Tx) error {
		pings := tx.Bucket([]byte("pings_by_minute"))
		if pings == nil {
			return errors.New("Couldn't find pings_by_minute bucket")
		}
		c := pings.Cursor()

		min := []byte(ipAddress + "_" + start.Format(time.RFC3339))
		max := []byte(ipAddress + "_" + end.Format(time.RFC3339))
		currGroup := NewPingGroup(start, start.Add(groupBy))
		// fmt.Printf("GRP: s:%s - e:%s\n", currGroup.Start.Format("01/02/06 3:04:05 pm"), currGroup.End.Format("01/02/06 3:04:05 pm"))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) >= -1; k, v = c.Next() {
			// keyParts := strings.Split(string(k), "_")
			for i := 0; i < len(v); i += PingResByteCount {
				pingTime, resTime, err := DeserializePingRes(v[i : i+PingResByteCount])
				if err != nil {
					return err
				}

				// Make sure we don't go beyond our end time
				if pingTime.Equal(end) || pingTime.After(end) {
					break
				}

				// Keep creating groups until one fits our bucket, this is here
				// because it's possible for a person to query a start time before there is any data
				// So return empty groups to the consumer (no pings), there is definitely a better way.
				// Why 50... because I pulled it out of my butt. Infinite loop protection, BRO
				for x := 0; x < 50; x++ {
					if pingTime.Equal(currGroup.Start) || (pingTime.After(currGroup.Start) && pingTime.Before(currGroup.End)) {
						currGroup.addResTime(resTime)
						break
					} else {
						currGroup.calcAvgAndStdDev()
						groups = append(groups, currGroup)

						currGroup = NewPingGroup(currGroup.End, currGroup.End.Add(groupBy))
						// fmt.Printf("%s - %s\n", currGroup.Start.Format("01/02 3:04:05 pm"), currGroup.End.Format("01/02 3:04:05 pm"))
					}
				}
			}
		}

		currGroup.calcAvgAndStdDev()
		groups = append(groups, currGroup)
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
