package main

import "DHPC-ipher/server"

func main() {
	srvr := new(server.Server)

	srvr.Start()
}
