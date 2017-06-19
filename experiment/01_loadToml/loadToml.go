package main

import (
	"github.com/BurntSushi/toml"
	"log"
)

func main() {
	config, _ := toml.LoadFile("./config.toml")
	// retrieve data directly
	user := config.Get("postgres.user")
	pass := config.Get("postgres.password")

	log.Printf("User:%s", user)
	log.Printf("Pass:%s", pass)
}
