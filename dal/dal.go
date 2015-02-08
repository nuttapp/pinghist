package dal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

const (
	// Bolt errors
	BucketNotFoundError = "could not find bucket"
	KeyNotFoundError    = "could not find key"
	// SavePing Errors
	IPRequiredError             = "IP can't be empty"
	ResponseTimeOutOfRangeError = "Response time must be >= -1"
	// Deserialize PingRes Errors
	TimeDeserializationError = "second offset is too large (> 59)"
	InvalidByteLength        = "invaid # of bytes"
	// GetPings Errors
	KeyTimestampParsingError = "Can't parse key timestamp"
)

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
	path        string
	fileName    string
	pingsBucket string
}

// NewDAL creates a new Data Access Layer with defaults for all fields
func NewDAL() *DAL {
	dal := &DAL{
		path:        "",
		fileName:    "pinghist.db",
		pingsBucket: "pings_by_minute",
	}
	return dal
}

// SavePingWithTransaction will save a ping to bolt using the given bolt transaction
func (dal *DAL) SavePingWithTransaction(ip string, startTime time.Time, responseTime float32, tx *bolt.Tx) error {
	pings := tx.Bucket([]byte(dal.pingsBucket))
	if pings == nil {
		return fmt.Errorf("dal.SavePingWithTransaction: %s %s", BucketNotFoundError, dal.pingsBucket)
	}

	key := GetPingKey(ip, startTime)
	val := SerializePingRes(startTime, responseTime)

	v := pings.Get(key)
	if v != nil {
		// Don't change the byte array that boltdb gives us, make our own new one
		// + the extra room for the next value
		newVal := make([]byte, 0, len(val)+PingResByteCount)
		newVal = append(newVal, v...)
		newVal = append(newVal, val...)
		val = newVal
	}

	err := pings.Put(key, val)
	if err != nil {
		return fmt.Errorf("dal.SavePingWithTransaction: error writing key: %s", err)
	}

	return nil
}

// SavePing will save a ping to bolt
// Pings are keyed by minute, so, every minute can store a max of 60 pings (1 p/sec)
// The pings within a minute are stored as an array of bytes for fast
// serialization/deserialization and to minimize the size of the value (see SerializePingRes)
func (dal *DAL) SavePing(ip string, startTime time.Time, responseTime float32) error {
	if len(ip) == 0 {
		return fmt.Errorf("dal.SavePing: %s", IPRequiredError)
	}
	if responseTime < -1 {
		return fmt.Errorf("dal.SavePing: %s", ResponseTimeOutOfRangeError)
	}

	db, err := bolt.Open(dal.fileName, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		return dal.SavePingWithTransaction(ip, startTime, responseTime, tx)
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
	PingResByteCount          = 7 // total bytes = time bytes + 1 + float32 bytes + 1
	PingResTimestampByteCount = 1 // time.Time.Second()
	PingResTimeByteCount      = 4 // float32
)

// SerializePingRes converts startTime and resTime to a 7 byte array
// startTime is the time the ping was initated
// resTime is the amount of time it took to return the ping packet
// endTime = baseKey + startTime + resTime
// Format: 7 bytes
// | 1 byte  | 1 byte  | 4 bytes | 1 byte
// | seconds | padding | resTime | padding
// TODO Convert to PingRes struct w/ method MarshalBinary()
func SerializePingRes(startTime time.Time, resTime float32) []byte {
	buff := make([]byte, PingResByteCount)
	floatBytes := Float32bytes(resTime)

	timeBytes := []byte{uint8(startTime.Second())}

	copy(buff[0:PingResTimestampByteCount], timeBytes)
	responseTimeOffset := PingResTimestampByteCount + 1
	copy(buff[responseTimeOffset:responseTimeOffset+PingResTimeByteCount], floatBytes)

	return buff
}

// DeserializePingRes does the opposite of SerializePingRes
func DeserializePingRes(data []byte) (uint8, float64, error) {
	if len(data) != PingResByteCount {
		return 0, 0, errors.New(InvalidByteLength)
	}
	secondOffset := data[0]
	if secondOffset > 59 {
		return 0, 0, errors.New(TimeDeserializationError)
	}

	responseTimeOffset := PingResTimestampByteCount + 1
	resTime := Float32frombytes(data[responseTimeOffset : responseTimeOffset+PingResTimeByteCount])

	return secondOffset, float64(resTime), nil
}

func StripNano(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

// GetPings returns pings between the start time and end time, for the given IP,
// grouped by the given duration.
// Start and end time should be in UTC
// gruupBy can be any valid time.Duration, ex: 1 * time.Hour
// Returns a summary for each PingGroup with avg and std deviation
func (dal *DAL) GetPings(ipAddress string, start, end time.Time, groupBy time.Duration) ([]*PingGroup, error) {
	db, err := bolt.Open(dal.fileName, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	groups := make([]*PingGroup, 0, 5)
	// fmt.Printf("GetPings() %s - %s\n", start.Format("01/02/06 3:04:05 pm"), end.Format("01/02/06 3:04:05 pm"))
	start = StripNano(start)
	end = StripNano(end)

	err = db.View(func(tx *bolt.Tx) error {
		pings := tx.Bucket([]byte("pings_by_minute"))
		if pings == nil {
			return fmt.Errorf("dal.GetPings: %s: %s", BucketNotFoundError, dal.pingsBucket)
		}
		c := pings.Cursor()

		min := GetPingKey(ipAddress, start)
		max := GetPingKey(ipAddress, end)
		currGroup := NewPingGroup(start, start.Add(groupBy))
		// fmt.Printf("GRPstart: %s \nGRP  end: %s\n", currGroup.Start.Format(time.RFC3339Nano), currGroup.End.Format(time.RFC3339Nano))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) >= -1; k, v = c.Next() {
			keyParts := strings.Split(string(k), "_")
			baseTime, err := time.Parse(time.RFC3339, keyParts[1])
			// fmt.Printf("baseTime: %s\n", baseTime.Format(time.RFC3339Nano))
			if err != nil {
				return fmt.Errorf("dal.GetPings: %s: %s", KeyTimestampParsingError, err)
			}

			for i := 0; i < len(v); i += PingResByteCount {
				secondsOffset, resTime, err := DeserializePingRes(v[i : i+PingResByteCount])
				if err != nil {
					return fmt.Errorf("dal.GetPings: %s", err)
				}
				pingTime := baseTime.Add(time.Duration(secondsOffset) * time.Second)
				// fmt.Printf("pingTime: %s\n", pingTime.Format(time.RFC3339Nano))

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
