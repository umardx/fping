package main

import (
	"fmt"
	"net/http"
	"time"
)

func startTask(task string) {
	fmt.Println("doing something", task)
}

func startPolling1() {
	for {
		time.Sleep(2 * time.Second)
		go startTask("from polling 1")
	}
}

func startPolling2() {
	for {
		<-time.After(2 * time.Second)
		go startTask("from polling 2")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
	go startPolling1()
	go startPolling2()

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
