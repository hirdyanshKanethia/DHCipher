package main

import (
	"encoding/binary"
	"log"
	"net"
)

func main() {
	// Let's connect to our local DHCP server
	serverAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:67")
	if err != nil {
		log.Fatalf("Error resolving address: %v", err)
	}

	conn, err := net.DialUDP("udp4", nil, serverAddr)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	// Let's manually construct a 240-byte bare minimum DHCP packet!
	packet := make([]byte, 244)

	// 1. Fixed Headers
	packet[0] = 1 // Opcode: 1 = BootRequest (Client to Server)
	packet[1] = 1 // Hardware Type: 1 = Ethernet
	packet[2] = 6 // Hardware Length: 6 = MAC Address length
	packet[3] = 0 // Hops

	// Transaction ID (Xid) - 4 bytes at index 4-7
	binary.BigEndian.PutUint32(packet[4:8], 0x12345678)

	// 2. Client MAC Address (Chaddr) - starts at index 28
	// Let's use fake MAC: 00:11:22:33:44:55
	packet[28] = 0x00
	packet[29] = 0x11
	packet[30] = 0x22
	packet[31] = 0x33
	packet[32] = 0x44
	packet[33] = 0x55

	// 3. The Magic Cookie! (Must be exactly these bytes at index 236-239)
	packet[236] = 0x63
	packet[237] = 0x82
	packet[238] = 0x53
	packet[239] = 0x63

	// 4. Options
	// Option 53: DHCP Message Type (length 1, value 1 = DISCOVER)
	packet[240] = 53
	packet[241] = 1
	packet[242] = 1

	// Option 255: End of Options marker
	packet[243] = 255

	log.Println("Sending custom DHCP DISCOVER packet...")
	_, err = conn.Write(packet)
	if err != nil {
		log.Fatalf("Error sending packet: %v", err)
	}

	log.Println("Packet sent successfully! Check your server terminal.")
}
