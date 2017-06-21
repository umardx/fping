## fping Monitoring with Infping/InfluxDB/Grafana + Daemon SystemD
Parse fping output, store result in influxdb 0.9, and visualizing with grafana.
```
$ go install $GOPATH/src/github.com/umardx/fping
$ mv $GOPATH/bin/fping $GOPATH/src/github.com/umardx/fping/
$ ./fping

```
