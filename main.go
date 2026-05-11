package main

import (
	"log"
	"os"

	"DHCipher/server"
)

func main() {
	cfg, err := server.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("[ERROR] config file couldn't be read")
	}

	logFile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		log.Fatalf("Could not open log file: %v", err)
	} else {
		log.SetOutput(logFile)
	}

	srvr := server.NewServer(cfg)

	srvr.Start()
}
