package dal

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_ip_stats_integration(t *testing.T) {
	dal := NewDAL()
	dal.CreateBuckets()

	SkipConvey("IPStats", t, func() {

		Convey("Should save IPStats to DB", func() {
			stats := IPStats{
				IP:           "127.0.0.1",
				FirstPingKey: "foo",
			}
			err := dal.SaveIPStats(stats)
			So(err, ShouldBeNil)
		})

		Convey("Should get IPStats from DB", func() {
			ipStats, err := dal.GetIPStats("127.0.0.1")
			So(err, ShouldBeNil)
			So(ipStats, ShouldNotBeNil)
		})

	})
}
