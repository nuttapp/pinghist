#!/bin/sh
size=`du -h pinghist.db`
kcount=`bolt keys pinghist.db pings_by_minute | wc -l | tr -d ' '`
echo "$size: $kcount keys"
