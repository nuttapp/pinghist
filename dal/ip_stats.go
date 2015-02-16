package dal

import (
	"encoding/json"

	"github.com/boltdb/bolt"
)
import "fmt"

const (
	IPStatsSerializationError    = "Could not serialize IPStats"
	IPStatsDerserializationError = "Could not deserialize IPStats"
)

// IPStats keep track of useful summary info about a particular IP address
type IPStats struct {
	IP           string // The ip address
	FirstPingKey string // first key of pings_by_minute bucket
	LastPingKey  string // last key ...
	Received     uint64
	Lost         uint64
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
	b, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("dal.SaveIPStats: %s: %s", IPStatsSerializationError, err)
	}

	return bucket.Put([]byte(stats.IP), b)
}
