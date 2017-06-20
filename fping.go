package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-systemd/daemon" // Add this for daemonized Infping
	"github.com/influxdata/influxdb/client"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	_ "reflect"
	"strconv"
	"strings"
	"time"
)

const (
	path       = "config.toml"
	url_consul = "http://a:a@consul1.dx/v1/catalog/nodes"
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

func getHost(url string) (ipmap map[string]string, nodes []string, hosts []string) {
	ipmap = make(map[string]string)
	data := getJson(url)
	cfg := Consul{}
	err := json.Unmarshal([]byte(data), &cfg)
	perr(err)
	for v := range cfg {
		nodes = append(nodes, cfg[v].Node)
		hosts = append(hosts, cfg[v].Address)
		ipmap[cfg[v].Node] = cfg[v].Address
	}

	for v := range ipmap {
		log.Printf("%s:%s", v, ipmap[v])
	}
	log.Println(nodes)
	return
}

func herr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func perr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func slashSplitter(c rune) bool {
	return c == '/'
}

func readPoints(config *toml.Tree, con *client.Client) {
	args := []string{"-B 1", "-D", "-r0", "-O 0", "-Q 10", "-p 1000", "-l"}
	// Parsing hosts

	hosts := config.Get("hosts.hosts").([]interface{})
	for _, v := range hosts {
		host, _ := v.(string)
		args = append(args, host)
	}

	log.Printf("Going to ping the following hosts: %q", hosts)
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
			host := fields[0]
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
			log.Printf("Host:%s, loss: %s, min: %s, avg: %s, max: %s", host, lossp, min, avg, max)
			writePoints(config, con, host, sent, recv, lossp, min, avg, max)
		}
	}
	std := bufio.NewReader(stdout)
	line, err := std.ReadString('\n')
	perr(err)
	log.Printf("stdout:%s", line)
}

func writePoints(config *toml.Tree, con *client.Client, host string, sent string, recv string, lossp string, min string, avg string, max string) {
	db := config.Get("influxdb.db").(string)
	loss, _ := strconv.Atoi(lossp)
	pts := make([]client.Point, 1)
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
	pts[0] = client.Point{
		Measurement: "ping",
		Tags: map[string]string{
			"host": host,
		},
		Fields:    fields,
		Time:      time.Now(),
		Precision: "",
	}

	bps := client.BatchPoints{
		Points:          pts,
		Database:        db,
		RetentionPolicy: "infinite",
	}
	_, err := con.Write(bps)
	if err != nil {
		log.Fatal(err)
	}
	daemon.SdNotify(false, "READY=1") // daemon fping
}

func main() {
	getHost(url_consul)
	config, err := toml.LoadFile(path)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	host := config.Get("influxdb.host").(string)
	port := config.Get("influxdb.port").(string)
	//measurement := config.Get("influxdb.measurement").(string)
	username := config.Get("influxdb.user").(string)
	password := config.Get("influxdb.pass").(string)

	u, err := url.Parse(fmt.Sprintf("http://%s:%s", host, port))
	if err != nil {
		log.Fatal(err)
	}
	conf := client.Config{
		URL:      *u,
		Username: username,
		Password: password,
	}

	con, err := client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	dur, ver, err := con.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connected to influxdb! %v, %s", dur, ver)

	readPoints(config, con)
}
