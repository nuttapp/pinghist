package dal

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_ping_group_unit(t *testing.T) {
	Convey("PingGroup", t, func() {
		pg := NewPingGroup(time.Now(), time.Now())

		Convey("addResTime()", func() {
			Convey("should increment Received, sum Total, & set Min/Max", func() {
				pg.addResTime(1.1)
				So(pg.Received, ShouldEqual, 1)
				So(pg.MinTime, ShouldEqual, 1.1)
				So(pg.MaxTime, ShouldEqual, 1.1)
				pg.addResTime(1.2)
				So(pg.Received, ShouldEqual, 2)
				So(pg.TotalTime, ShouldEqual, 2.3)
				So(pg.MinTime, ShouldEqual, 1.1)
				So(pg.MaxTime, ShouldEqual, 1.2)
			})
			Convey("should increment TimedOut ", func() {
				pg.addResTime(-1)
				So(pg.Timedout, ShouldEqual, 1)
				pg.addResTime(-1)
				So(pg.Timedout, ShouldEqual, 2)
				So(pg.MinTime, ShouldEqual, 0)
				So(pg.MaxTime, ShouldEqual, 0)
			})
			Convey("should append to resTimes", func() {
				pg.addResTime(1.1)
				So(len(pg.resTimes), ShouldEqual, 1)
			})
			Convey("should not append to resTimes", func() {
				pg.addResTime(-1)
				So(len(pg.resTimes), ShouldEqual, 0)
			})
		})

		Convey("calcAvgAndStdDev()", func() {
			Convey("should calculate Avg and StdDev", func() {
				resTimes := []float64{10.190, 17.039, 14.165, 13.950, 14.488, 14.295, 19.534, 13.865, 12.782,
					19.113, 15.523, 17.922, 18.841, 18.680, 40.791, 13.798, 17.049, 21.680, 18.660, 21.077,
					14.487, 13.538, 13.666, 13.512, 17.300, 13.480, 14.460, 13.860, 15.185, 18.411, 13.789,
					14.262, 13.232, 11.794, 17.672, 15.491, 18.298, 16.718, 15.182, 13.835, 12.196, 13.142,
					15.329, 10.543, 15.527, 18.212, 15.957, 13.989, 13.492, 24.896, 13.535, 9.689, 17.656,
					14.776, 14.508, 12.150, 13.335, 14.171, 10.721, 13.028, 15.609, 14.225, 20.640, 14.229,
					12.222, 10.949, 12.263, 29.830, 12.987, 13.239, 18.613, -1, 15.019, 16.007, 15.599}

				for _, resTime := range resTimes {
					pg.addResTime(resTime)
				}

				pg.calcAvgAndStdDev()
				So(pg.AvgTime, ShouldEqual, 15.674283783783785)
				So(pg.StdDev, ShouldEqual, 4.3960093436202446)
				So(pg.resTimes, ShouldBeNil)
			})
			Convey("should not calculate Avg and StdDev", func() {
				pg.calcAvgAndStdDev()
				So(pg.AvgTime, ShouldEqual, 0)
				So(pg.StdDev, ShouldEqual, 0)
			})
			Convey("should free memory of resTimes array", func() {
				pg.addResTime(1.1)
				pg.addResTime(1.2)
				pg.calcAvgAndStdDev()
				So(pg.resTimes, ShouldBeNil)
			})
		})

	})
}
