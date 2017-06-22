package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	path = "config.toml"
)

var (
	newnodes      = make(map[string]string)
	oldnodes      = make(map[string]string)
	change   bool = false
)

type Consul []struct {
	ID              string `json:"ID"`
	Node            string `json:"Node"`
	Address         string `json:"Address"`
	Datacenter      string `json:"Datacenter"`
	TaggedAddresses struct {
		Lan string `json:"lan"`
		Wan string `json:"wan"`
	} `json:"TaggedAddresses"`
	Meta struct {
	} `json:"Meta"`
	CreateIndex int `json:"CreateIndex"`
	ModifyIndex int `json:"ModifyIndex"`
}

func watchNodes(url string) {
	var first bool = true
	for {
		newnodes = getNodes(url)
		// Check if any change of consul nodes
		if reflect.DeepEqual(newnodes, oldnodes) {
			// give time for requests newnodes
			time.Sleep(5 * time.Second)
		} else {
			if first {
				first = false
			} else {
				change = true
			}
			oldnodes = newnodes
		}
	}
}

func getNodes(url string) (nodes map[string]string) {
	nodes = make(map[string]string)
	data := getJson(url)
	cfg := Consul{}
	err := json.Unmarshal([]byte(data), &cfg)
	perr(err)
	for v := range cfg {
		nodes[cfg[v].Address] = cfg[v].Node
	}
	return
}

func getJson(url string) (json string) {
	resp, err := http.Get(url)
	herr(err)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	perr(err)

	if resp.StatusCode == 200 {
		json = string(body)
	} else {
		log.Println(resp.Status)
		json = "[]"
	}

	return json
}

func herr(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func perr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func slashSplitter(c rune) bool {
	return c == '/'
}

func readPoints(config *toml.Tree, con client.Client) {
START:
	nodes := newnodes
	args := []string{"-B 1", "-D", "-r0", "-O 0", "-Q 10", "-p 1000", "-l"}
	list := []string{}
	for u := range nodes {
		ip := u
		args = append(args, ip)
		list = append(list, ip)

	}

	log.Printf("Going to ping the following ips: %v", list)
	cmd := exec.Command("/usr/bin/fping", args...)

	stdout, err := cmd.StdoutPipe()
	herr(err)
	stderr, err := cmd.StderrPipe()
	herr(err)
	cmd.Start()
	perr(err)

	buff := bufio.NewScanner(stderr)
	for buff.Scan() {
		text := buff.Text()
		fields := strings.Fields(text)
		// Ignore timestamp
		if len(fields) > 1 {
			ip := fields[0]
			data := fields[4]
			dataSplitted := strings.FieldsFunc(data, slashSplitter)
			// Remove ,
			dataSplitted[2] = strings.TrimRight(dataSplitted[2], "%,")
			sent, recv, lossp := dataSplitted[0], dataSplitted[1], dataSplitted[2]
			min, max, avg := "", "", ""
			// Ping times
			if len(fields) > 5 {
				times := fields[7]
				td := strings.FieldsFunc(times, slashSplitter)
				min, avg, max = td[0], td[1], td[2]
			}
			log.Printf("Node:%s, IP:%s, send:%s, recv:%s loss: %s, min: %s, avg: %s, max: %s", nodes[ip], ip, sent, recv, lossp, min, avg, max)
			writePoints(config, con, nodes, ip, sent, recv, lossp, min, avg, max)
		}

		// Restart scan if nodes change
		if change {
			log.Println("Nodes updated : Restarting fping")
			change = false
			cmd.Process.Kill()
			cmd.Wait()
			goto START
		}
	}
	std := bufio.NewReader(stdout)
	line, err := std.ReadString('\n')
	perr(err)
	log.Printf("stdout:%s", line)
}

func writePoints(config *toml.Tree, con client.Client, nodes map[string]string, ip string, sent string, recv string, lossp string, min string, avg string, max string) {
	db := config.Get("influxdb.db").(string)
	ms := config.Get("influxdb.measurement").(string)
	ps := config.Get("influxdb.precision").(string)
	rp := config.Get("influxdb.retentionpolicy").(string)

	loss, _ := strconv.Atoi(lossp)
	fields := map[string]interface{}{}
	if min != "" && avg != "" && max != "" {
		min, _ := strconv.ParseFloat(min, 64)
		avg, _ := strconv.ParseFloat(avg, 64)
		max, _ := strconv.ParseFloat(max, 64)
		fields = map[string]interface{}{
			"loss": loss,
			"min":  min,
			"avg":  avg,
			"max":  max,
		}
	} else {
		fields = map[string]interface{}{
			"loss": loss,
		}
	}

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        db,
		Precision:       ps,
		RetentionPolicy: rp,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a point and add to batch
	tags := map[string]string{
		"node": nodes[ip],
		"addr": ip,
	}

	pt, err := client.NewPoint(ms, tags, fields, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	bp.AddPoint(pt)

	// Write the batch
	if err := con.Write(bp); err != nil {
		log.Fatal(err)
	}
}

func main() {
	config, err := toml.LoadFile(path)
	herr(err)

	host := config.Get("influxdb.host").(string)
	port := config.Get("influxdb.port").(string)
	username := config.Get("influxdb.user").(string)
	password := config.Get("influxdb.pass").(string)

	addr := fmt.Sprintf("http://%s:%s", host, port)

	// Create a new HTTPClient
	con, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	})
	herr(err)

	dur, ver, err := con.Ping(1)
	herr(err)

	log.Printf("Connected to influxdb! (dur:%v, ver:%s)", dur, ver)

	url := config.Get("consul.url").(string)
	newnodes = getNodes(url)

	go watchNodes(url)

	readPoints(config, con)

}
