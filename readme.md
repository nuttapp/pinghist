Ping History
===========

### A tool for measuring uptime

*Pinghist* is a command line tool to capture the latency between servers over time. The goal of the tool is to provide a simple and reliable command line app that's better than ping but not as not as complex as hosted monitoring (sites like Pingdom).

```
ping                 pinghist                Pingdom
|-----------------------|--------------------------|
                      booya
```

#### Great for

- backend engineers that want troubleshoot network connectivity between servers
- network engineers that want figure out when a router/switch/hub is acting up
- on-call types that want fine grained latency stats to help them fix a problem
- nerds that want to know when their internet poops out
- gamers obsessed with their ping

#### Useful when

- you need ping one to 10s of servers
- you need to measure latency over minutes, hours or weeks


#### Not as useful when
- you need to ping 100s of servers
- you need month-over-month reporting

-

### Examples 

Let's ping our home router to measure how often our wifi signal dies. Pinghist will ping 192.168.1.1 every second until it's killed.
```
$ pinghist -h 192.168.1.1
4.653
4.259
4.306
...
```

Suppose you've been running the command above for 3 hours. Assuming you started pinghist on Jan 3rd at 5pm the following will detail 3 hours of pings. The min, avg, max, std dev, recevied/lost count are all calculated based on the value of `-groupby`.

```
$ pinghist -start "1/3 5:00 pm" -end "1/3 8:00 pm" -groupby 1h
```
```
      TIME      | MIN |  AVG  |  MAX   | STD DEV | RECEIVED | LOST
+---------------+-----+-------+--------+---------+----------+------+
  01/03 05:00pm | 6ms | 749ms | 1500ms |    34ms |     3600 |    0
  01/03 06:00pm | 5ms | 750ms | 1500ms |    41ms |     3600 |    0
  01/03 07:00pm | 5ms | 749ms | 1500ms |    34ms |     3600 |    0
```

Same as above using 24hr time.
```
$ pinghist -start "01/03 17:00" -end "1/04 20:00" -groupby 1hr
```

You can also omit the date entirely, pinghist assumes the current date
```
$ pinghist -start "17:00" -end "20:00" -groupby 1hr
```

###Example 2

Detail 1 hour of pings, starting on Jan 3rd @ 4PM, and group them by 15 minutes. Things are pretty cool, avg isn't great but it's consistent, as is standard deviation. We didn't drop a single ping packet.
```
$ pinghist -start "1/3 4:00 pm" -end "1/3 5:00 pm" -groupby 15min
```
```
      TIME      |  MIN  |  AVG   |   MAX   | STD DEV | RECEIVED | LOST
+---------------+-------+--------+---------+---------+----------+------+
  01/03 04:00pm | 11 ms | 730 ms | 1500 ms |   32 ms |      900 |    0
  01/03 04:15pm |  6 ms | 758 ms | 1499 ms |   35 ms |      900 |    0
  01/03 04:30pm |  9 ms | 735 ms | 1500 ms |   32 ms |      900 |    0
  01/03 04:45pm |  7 ms | 785 ms | 1500 ms |   22 ms |      900 |    0
```

###Example 3

Detail 1 hour of pings, starting on Jan 3rd @ 6PM, and group them by 15 minutes. Something fishy happened between 6 & 6:15pm, the average jumped to 3 seconds. Standard deviation is large, which means average is swinging wildly. And 217 pings timed out.

```
$ pinghist -start "1/3 6:00 pm" -end "1/3 7:00 pm" -groupby 15min
```
```
      TIME      |  MIN   |  AVG    |   MAX   | STD DEV | RECEIVED | LOST
+---------------+--------+---------+---------+---------+----------+------+
  01/03 06:00pm | 300 ms | 3050 ms | 7500 ms | 1050 ms |      683 |  217
  01/03 06:15pm |   6 ms |   58 ms |  299 ms |   45 ms |      900 |    0
  01/03 06:30pm |   9 ms |   35 ms |   85 ms |   32 ms |      900 |    0
  01/03 06:45pm |   7 ms |   85 ms |  217 ms |   22 ms |      900 |    0
```

-

#### [Download](https://github.com/nuttapp/pinghist/releases/tag/v0.1)
