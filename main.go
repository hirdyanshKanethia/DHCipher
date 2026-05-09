package main

import (
	"log"

	"DHCP-ipher/server"
)

func main() {
	cfg, err := server.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("[ERROR] config file couldn't be read")
	}

	srvr := server.NewServer(cfg)

	srvr.Start()
}
