#!/bin/sh
size=`du -h ./dal/pinghist.db`
kcount=`bolt keys ./dal/pinghist.db pings_by_minute | wc -l | tr -d ' '`
echo "$size: $kcount keys"
