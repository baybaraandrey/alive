Introduction
============

The program provides a simple application for monitoring the availability of hosts on the network
using icmp protocol.


Installation
============

```sh
make build
```


Configuration
============
Alive uses the yaml format to describe configuration. For example if you want to watching
localhost and google just create a config.yaml file with the following content.
```yaml
hosts:
  - addr: 127.0.0.1
    interval: 1s
    read-timeout: 10s
    packet-size: 0
    ttl: 64
  - addr: google.com
    interval: 5s
    read-timeout: 10s
    packet-size: 0
    ttl: 64
```

Alive will be send packets to localhost every second
and for google will send every 5 seconds.
The response timeout will be 10 seconds for both.
Packet size is the same for both and time to live 64 for both too.

Sample output:
```sh
WATCHER : 2021/12/27 14:50:59.148378 alive.go:80: WATCHER : started : pid 44816
WATCHER : 2021/12/27 14:50:59.149084 alive.go:37: WATCHER : init : hosts : 2
WATCHER : 2021/12/27 14:50:59.149116 alive.go:38: WATCHER : packets listening on : 0.0.0.0
WATCHER : 2021/12/27 14:50:59.149133 alive.go:71: WATCHER : init : 127.0.0.1
WATCHER : 2021/12/27 14:50:59.149138 alive.go:72: WATCHER :        read-timeout : 10s
WATCHER : 2021/12/27 14:50:59.149141 alive.go:73: WATCHER :        interval : 1s
WATCHER : 2021/12/27 14:50:59.149144 alive.go:74: WATCHER :        packet-size : 0
WATCHER : 2021/12/27 14:50:59.149757 alive.go:71: WATCHER : init : google.com
WATCHER : 2021/12/27 14:50:59.149764 alive.go:72: WATCHER :        read-timeout : 10s
WATCHER : 2021/12/27 14:50:59.149766 alive.go:73: WATCHER :        interval : 5s
WATCHER : 2021/12/27 14:50:59.149769 alive.go:74: WATCHER :        packet-size : 0
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=112.427µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=121.179µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=97.475µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=96.635µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=98.367µs
google.com | 142.251.39.46: icmp_seq=0 ttl=64 time=25.08946ms
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=122.347µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=102.515µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=129.561µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=108.011µs
127.0.0.1 | 127.0.0.1: icmp_seq=0 ttl=64 time=114.11µs
google.com | 142.251.39.46: icmp_seq=0 ttl=64 time=24.849796ms
^CWATCHER : 2021/12/27 14:51:09.854806 alive.go:109: WATCHER: interrupt : start shutdown
WATCHER : 2021/12/27 14:51:09.854861 alive.go:111: WATCHER : stop : 127.0.0.1
WATCHER : 2021/12/27 14:51:09.854885 alive.go:111: WATCHER : stop : google.com
WATCHER : 2021/12/27 14:51:09.854904 alive.go:116: WATCHER: completed
```
 

Usage
=====

```sh
# run help
./alive --help
Usage of ./alive:
  -address string
        listen address. (default "0.0.0.0")
  -config string
        path to config file. (default "./config.yaml")
  -proto string
        'udp'|'icmp'. Setting to 'icmp' requires that it be run with super-user privileges. (default "udp"

# add your hosts or ip addresses to config.yaml and run
./alive -config ./config.yaml
```
