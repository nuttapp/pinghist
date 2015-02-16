package dal

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)
import "fmt"

const (
	IPStatsSerializationError    = "Could not serialize IPStats"
	IPStatsDerserializationError = "Could not deserialize IPStats"
	IPStatsRequiredError         = "IPStats cannot be nil"
)

// IPStats keep track of useful summary info about a particular IP address
type IPStats struct {
	IP            string    // The ip address
	FirstPingKey  string    // first key of pings_by_minute bucket
	FirstPingTime time.Time // The timestamp of the first ping attempt
	LastPingKey   string    // last key ...
	LastPingTime  time.Time // The timestamp of the last ping attempt
	Received      uint64
	Lost          uint64
}

func (dal *DAL) GetAllIPStats() ([]*IPStats, error) {
	db, err := bolt.Open(dal.fileName, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	allStats := []*IPStats{}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dal.ipStatsBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var s IPStats
			err := json.Unmarshal(v, &s)
			if err != nil {
				return nil
			}
			allStats = append(allStats, &s)
		}
		return nil
	})

	return allStats, err
}

func (dal *DAL) GetIPStats(ip string) (*IPStats, error) {
	db, err := bolt.Open(dal.fileName, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var ipStats *IPStats
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(dal.ipStatsBucket))
		ipStats, err = dal.GetIPStatsFromBucket(ip, bucket)
		if err != nil {
			return err
		}
		return nil
	})

	return ipStats, err
}

func (dal *DAL) GetIPStatsFromBucket(ip string, bucket *bolt.Bucket) (*IPStats, error) {
	if len(ip) == 0 {
		return nil, fmt.Errorf("dal.GetIPStatsFromBucket: %s", IPRequiredError)
	}
	if bucket == nil {
		return nil, fmt.Errorf("dal.GetIPStatsFromBucket: %s %s", BucketNotFoundError, dal.ipStatsBucket)
	}

	b := bucket.Get([]byte(ip))
	if b == nil {
		return nil, nil
	}
	var ipStats *IPStats
	err := json.Unmarshal(b, &ipStats)
	if err != nil {
		return nil, fmt.Errorf("dal.GetIPStatsFromBucket: %s: %s", IPStatsDerserializationError, err)
	}
	return ipStats, nil
}

func (dal *DAL) SaveIPStats(stats *IPStats) error {
	db, err := bolt.Open(dal.fileName, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(dal.ipStatsBucket))
		return dal.SaveIPStatsInBucket(stats, bucket)
	})

	return err
}

func (dal *DAL) SaveIPStatsInBucket(stats *IPStats, bucket *bolt.Bucket) error {
	if stats == nil {
		return fmt.Errorf("dal.SaveIPStatsInBucket: %s", IPStatsRequiredError)
	}
	if len(stats.IP) == 0 {
		return fmt.Errorf("dal.SaveIPStatsInBucket: %s", IPRequiredError)
	}
	if bucket == nil {
		return fmt.Errorf("dal.SaveIPStatsInBucket: %s %s", BucketNotFoundError, dal.ipStatsBucket)
	}

	b, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("dal.SaveIPStatsInBucket: %s: %s", IPStatsSerializationError, err)
	}

	return bucket.Put([]byte(stats.IP), b)
}
