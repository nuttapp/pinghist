package dal

import (
	"math"
	"time"
)

// PingGroup is used to summarize the output of the pings_by_minute bucket
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
