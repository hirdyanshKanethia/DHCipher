package main

import "DHCP-ipher/server"

func main() {
	srvr := new(server.Server)

	srvr.Start()
}
