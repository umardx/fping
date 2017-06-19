package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	hosts := []string{}
	file, err := os.Open("hosts")
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		host := scanner.Text()
		hosts = append(hosts, host)
	}

	res := PingHost(hosts)

	for i := 0; i < len(hosts); i++ {
		fmt.Println(<-res)
	}

}

func PingHost(hosts []string) <-chan string {
	c := make(chan string)
	for _, host := range hosts {
		go func(host string) {
			cmd := exec.Command("fping", "-c3", host)
			var stdOut bytes.Buffer
			var stdErr bytes.Buffer
			cmd.Stdout = &stdOut
			cmd.Stderr = &stdErr
			err := cmd.Run()
			if err != nil {
				c <- fmt.Sprintf("[FALSE] %s", host)
			} else {
				c <- fmt.Sprintf("[TRUE] %s", host)
			}
		}(host)
	}
	return c
}
