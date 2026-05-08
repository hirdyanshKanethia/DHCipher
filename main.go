package main

import "DHCP-ipher/server"

func main() {
	srvr := server.NewServer()

	srvr.Start()
}
