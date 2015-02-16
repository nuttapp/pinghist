package dal

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_ip_stats_integration(t *testing.T) {
	Convey("IPStats", t, func() {
		dal := NewDAL()
		dal.CreateBuckets()

		stats := &IPStats{
			IP:           "127.0.0.1",
			FirstPingKey: "foo",
			LastPingKey:  "bar",
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
			Convey("Should get & update IPStats", func() {
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
	})
}
