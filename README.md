## fping Monitoring with Infping/InfluxDB/Grafana + Daemon SystemD
Parse fping output, store result in influxdb 1.2, and visualizing with grafana.

#### Requirement:
##### Golang:
Install golang : https://golang.org/doc/install
##### Fping
```
$ sudo apt-get install fping
```

#### Edit config.toml:

```
[influxdb]

host = "192.168.114.30"
port = "8086"
db = "fping"
measurement = "ping"
precision = "ms"
retentionpolicy = "infinite"
user = "fping"
pass = "fpingdakjwgkawjnmbjhwtuia"

[consul]

url = "http://a:a@consul1.dx/v1/catalog/nodes"
```
#### Install fping:
```
$ ./setup.sh
$ sudo systemctl status infping.service

```

#### Output
```
2017/06/21 20:01:02 Connected to influxdb! (dur:1.996646ms, ver:1.2.4)
2017/06/21 20:01:02 Going to ping the following ips: [192.168.200.121 192.168.114.30]
2017/06/21 20:01:12 Node:b827eb3068d3am13k, IP:192.168.200.121, loss: 0, min: 5.83, avg: 17.3, max: 76.4
2017/06/21 20:01:12 Node:consulnode, IP:192.168.114.30, loss: 0, min: 0.47, avg: 0.59, max: 0.68
```
