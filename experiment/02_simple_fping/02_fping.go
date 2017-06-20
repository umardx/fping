package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := make(chan string)
	msg := []string{"itb.ac.id", "et.itb.ac.id", "bukalapak.co", "bni.co.id", "167.205.8.2"}
	for _, host := range msg {
		go ping(host, c)
		fmt.Printf("%s", <-c)

	}
}

func ping(host string, c chan string) {
	out, _ := exec.Command("fping", "-c", "3", host).Output()
	c <- fmt.Sprintf("%s", out)
}
