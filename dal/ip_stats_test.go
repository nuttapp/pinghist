package dal

import (
	"os"
	"sort"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_ip_stats_unit(t *testing.T) {
	Convey("[]*IPStats", t, func() {
		Convey("should sort list in ascending order by LastPingTime", func() {
			n := time.Now()
			stats := []*IPStats{
				&IPStats{IP: "3", LastPingTime: n.Add(15 * time.Second)},
				&IPStats{IP: "1", LastPingTime: n.Add(1 * time.Second)},
				&IPStats{IP: "2", LastPingTime: n.Add(10 * time.Second)},
			}

			sort.Stable(ByLastPingTime(stats))
			So(stats[0].IP, ShouldEqual, "1")
			So(stats[1].IP, ShouldEqual, "2")
			So(stats[2].IP, ShouldEqual, "3")
		})
	})
}

func Test_ip_stats_integration(t *testing.T) {
	Convey("IPStats", t, func() {
		dal := NewDAL()
		dal.DeleteBuckets()
		dal.CreateBuckets()
		Reset(func() {
			os.Remove(dal.fileName)
		})

		now := time.Now().UTC()
		ip := "127.0.0.1"
		stats := &IPStats{
			IP:           ip,
			FirstPingKey: string(GetPingKey(ip, now)),
			LastPingKey:  string(GetPingKey(ip, now)),
			Received:     1,
			Lost:         2,
		}

		Convey("SaveIPStats()", func() {
			Convey("should save IPStats", func() {
				err := dal.SaveIPStats(stats)
				So(err, ShouldBeNil)
			})
			Convey("should return error with invalid IP", func() {
				stats.IP = ""
				err := dal.SaveIPStats(stats)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, IPRequiredError)
			})
			Convey("should return error when it can't open db ", func() {
				dal.fileName = ""
				err := dal.SaveIPStats(stats)
				So(err, ShouldNotBeNil)
			})
			Convey("should return error when bucket doesn't exist", func() {
				dal.DeleteBuckets()
				err := dal.SaveIPStats(stats)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, BucketNotFoundError)
			})
			Convey("should return error when given nil stats", func() {
				dal.DeleteBuckets()
				err := dal.SaveIPStats(nil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, IPStatsRequiredError)
			})
		})

		Convey("GetIPStats()", func() {
			Convey("Should save & be able to update IPStats", func() {
				err := dal.SaveIPStats(stats)
				So(err, ShouldBeNil)
				sSaved, err := dal.GetIPStats(stats.IP)

				So(err, ShouldBeNil)
				So(sSaved, ShouldNotBeNil)
				So(sSaved.IP, ShouldEqual, stats.IP)
				So(sSaved.FirstPingKey, ShouldEqual, stats.FirstPingKey)
				So(sSaved.LastPingKey, ShouldEqual, stats.LastPingKey)
				So(sSaved.Received, ShouldEqual, stats.Received)
				So(sSaved.Lost, ShouldEqual, stats.Lost)

				sSaved.Received = 500
				err = dal.SaveIPStats(sSaved)
				So(err, ShouldBeNil)
				sUpdated, err := dal.GetIPStats(sSaved.IP)
				So(err, ShouldBeNil)
				So(sUpdated.Received, ShouldEqual, sSaved.Received)
			})
			Convey("should return nil when key doesn't exist", func() {
				sSaved, err := dal.GetIPStats("this key wont exist")
				So(err, ShouldBeNil)
				So(sSaved, ShouldBeNil)
			})
			Convey("should return error with invalid IP", func() {
				_, err := dal.GetIPStats("")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, IPRequiredError)
			})
			Convey("should return error when it can't open db ", func() {
				dal.fileName = ""
				_, err := dal.GetIPStats(stats.IP)
				So(err, ShouldNotBeNil)
			})
			Convey("should return error when bucket doesn't exist", func() {
				dal.DeleteBuckets()
				_, err := dal.GetIPStats(stats.IP)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, BucketNotFoundError)
			})
			Convey("should return error when it can't deserialize IPStats", func() {
				dal.Put(stats.IP, []byte("bogus data"), dal.ipStatsBucket)
				_, err := dal.GetIPStats(stats.IP)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, IPStatsDerserializationError)
			})
		})

		Convey("GetAllIPStats()", func() {
			Convey("should insert 3 and return 3 IPSstats", func() {
				// save 3 stats
				err := dal.SaveIPStats(stats)
				So(err, ShouldBeNil)
				stats.IP = "192.168.1.1"
				err = dal.SaveIPStats(stats)
				So(err, ShouldBeNil)
				stats.IP = "192.168.1.2"
				err = dal.SaveIPStats(stats)
				So(err, ShouldBeNil)

				allStats, err := dal.GetAllIPStats()
				So(err, ShouldBeNil)
				So(allStats, ShouldNotBeNil)
				So(len(allStats), ShouldEqual, 3)
			})
		})

	})
}
