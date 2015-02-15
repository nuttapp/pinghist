package dal

import "fmt"

// IPStats keep track of useful summary infomration about a particular IP address
type IPStats struct {
	IP           string // The ip address
	FirstPingKey string // first key of pings_by_minute bucket
	LastPingKey  string // last key ...
	TotalPings   uint64
}

func (dal *DAL) GetIPStats(IP string) (*IPStats, error) {
	return nil, fmt.Errorf("not implemented")
}

func (dal *DAL) SaveIPStats(stats IPStats) error {
	return fmt.Errorf("not implemented")
}
