Ping History
===========

### A tool for measuring uptime

*Pinghist* is a command line tool to capture the latency between servers and report on the result over extended periods of time. 
The goal of the tool is to provide a simple, and reliable command line app that's better than `ping` but not as not as complex as fully hosted monitoring (Pingdom, etc...).

```
Ping                 pinghist                Pingdom
|-----------------------|--------------------------|
                      booya
```


### Use cases

- For sys admins that want to keep track of network connectivity between servers
- For network engineers that want measure when a router/switch/hub is dropping packets
- For backend engineers that want to troubleshoot why services have trouble communicating (maybe you see pings drop when your services start barfing up errors)
- For nerds that want to track when their ISP gets flakey
- For on-call types that want fine grained uptime stats
- For you, because you're beautiful, and I made it for you


### Design goals

Pinghist is designed to scale pretty well on modest hardware. Here are a few design goals I had in mind when first building it. 
* ping 10s of servers, not 100s
* store millions of measurements, not billions
* Run for minutes to months, not years
* Report on fine grained detail by default (1 ping p/sec)
* Favor realtime reporting over historical reporting (no materialized reports)

-

###Examples 

Detail 24 hours of pings, starting on Jan 3rd @ 5PM, and group them by 1 hour. The avg isn't great but at least it's consistent, as is the standard deviation. 

```
$ pinghist -start 01/03 5:00pm -end 1/04 5:00pm -groupby 1hr
```
```
      TIME      | MIN |  AVG  |  MAX   | STD DEV | RECEIVED | LOST
+---------------+-----+-------+--------+---------+----------+------+
  01/03 05:00pm | 6ms | 749ms | 1500ms |    34ms |     3600 |    0
  01/03 06:00pm | 5ms | 750ms | 1500ms |    41ms |     3600 |    0
  01/03 07:00pm | 5ms | 749ms | 1500ms |    34ms |     3600 |    0
  01/03 08:00pm | 5ms | 742ms | 1500ms |    31ms |     3600 |    0
  01/03 09:00pm | 5ms | 741ms | 1499ms |    32ms |     3600 |    0
  01/03 10:00pm | 6ms | 763ms | 1499ms |    32ms |     3600 |    0
  01/03 11:00pm | 6ms | 752ms | 1500ms |    33ms |     3600 |    0
  01/04 12:00am | 5ms | 758ms | 1500ms |    34ms |     3600 |    0
  01/04 01:00am | 5ms | 755ms | 1500ms |    32ms |     3600 |    0
  01/04 02:00am | 5ms | 751ms | 1500ms |    31ms |     3600 |    0
  01/04 03:00am | 5ms | 768ms | 1500ms |    30ms |     3600 |    0
  01/04 04:00am | 5ms | 752ms | 1500ms |    30ms |     3600 |    0
  01/04 05:00am | 6ms | 746ms | 1500ms |    32ms |     3600 |    0
  01/04 06:00am | 5ms | 753ms | 1499ms |    34ms |     3600 |    0
  01/04 07:00am | 5ms | 757ms | 1500ms |    37ms |     3600 |    0
  01/04 08:00am | 5ms | 758ms | 1499ms |    36ms |     3600 |    0
  01/04 09:00am | 5ms | 767ms | 1500ms |    31ms |     3600 |    0
  01/04 10:00am | 5ms | 754ms | 1498ms |    31ms |     3600 |    0
  01/04 11:00am | 5ms | 744ms | 1499ms |    30ms |     3600 |    0
  01/04 12:00pm | 5ms | 743ms | 1500ms |    31ms |     3600 |    0
  01/04 01:00pm | 5ms | 745ms | 1499ms |    36ms |     3600 |    0
  01/04 02:00pm | 5ms | 751ms | 1499ms |    34ms |     3600 |    0
  01/04 03:00pm | 5ms | 747ms | 1499ms |    33ms |     3600 |    0
  01/04 04:00pm | 5ms | 756ms | 1500ms |    35ms |     3600 |    0

```

Same detail as above using 24hr time.
```
$ pinghist -start 01/03 17:00 -end 1/04 17:00 -groupby 1hr
```
```
      TIME      | MIN |  AVG  |  MAX   | STD DEV | RECEIVED | LOST
+---------------+-----+-------+--------+---------+----------+------+
  01/03 05:00pm | 6ms | 749ms | 1500ms |    34ms |     3600 |    0
  01/03 06:00pm | 5ms | 750ms | 1500ms |    41ms |     3600 |    0
...etc...

```

###Example 2

Detail 1 hour of pings, starting on Jan 3rd @ 4PM, and group them by 15 minutes. Things are pretty cool, avg isn't great but it's consistent, as is standard deviation. We didn't drop a single ping packet.
```
$ pinghist -start 01/03 4:00pm -end 1/03 5:00pm -groupby 15min
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
$ pinghist -start 01/03 6:00pm -end 1/03 7:00pm -groupby 15min
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

### Usage & documentation - http://www.pinghist.io
