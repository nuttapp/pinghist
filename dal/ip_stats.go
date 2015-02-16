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

func (dal *DAL) GetIPStats(ip string) (IPStats, error) {
	if len(ip) == 0 {
		return IPStats{}, fmt.Errorf("dal.GetIPStats: %s", IPRequiredError)
	}

	db, err := bolt.Open(dal.fileName, 0600, nil)
	if err != nil {
		return IPStats{}, err
	}
	defer db.Close()

	var ipStats IPStats
	err = db.View(func(tx *bolt.Tx) error {
		statsbucket := tx.Bucket([]byte(dal.ipStatsBucket))
		b := statsbucket.Get([]byte(ip))
		err := json.Unmarshal(b, &ipStats)
		if err != nil {
			return fmt.Errorf("dal.GetIPStats: %s: %s", IPStatsDerserializationError, err)
		}
		return nil
	})

	return ipStats, err

}

func (dal *DAL) SaveIPStats(stats IPStats) error {
	if len(stats.IP) == 0 {
		return fmt.Errorf("dal.SaveIPStats: %s", IPRequiredError)
	}

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

func (dal *DAL) SaveIPStatsInBucket(stats IPStats, bucket *bolt.Bucket) error {
	b, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("dal.SaveIPStats: %s: %s", IPStatsSerializationError, err)
	}

	return bucket.Put([]byte(stats.IP), b)
}
