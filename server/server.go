package server

import (
	// "encoding/hex"
	// "fmt"
	"log"
	"net"
	// "os"
	// "time"
)

type Server struct{}

func (s Server) Start() {
	log.Println("Starting DHCP Server...")

	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:67")
	if err != nil {
		log.Fatalf("Error resolving UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalf("Error starting UDP listener on port 67: %v\nMake sure no other DHCP server or dnsmasq is running locally, and you may need sudo.", err)
	}
	defer conn.Close()

	log.Printf("Listening for DHCP packets on %s\n", addr.String())

	buffer := make([]byte, 4096)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		
		// Packet logging for debugging
		// if err == nil {
		// 	// 1. Create a hex dump of the raw packet
		// 	hexData := hex.EncodeToString(buffer[:n])
		//
		// 	// 2. Append it to a debug file
		// 	f, _ := os.OpenFile("packets.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		// 	f.WriteString(fmt.Sprintf("TIME: %d | FROM: %s | LEN: %d | DATA: %s\n",
		// 		time.Now().Unix(), remoteAddr.String(), n, hexData))
		// 	f.Close()
		// }
		// if err != nil {
		// 	log.Printf("Error reading from UDP: %v", err)
		// 	continue
		// }

		packet, err := ParsePacket(buffer[:n])
		if err != nil {
			log.Printf("Parsing error for packet: %v", err)
			continue
		}

		mac := net.HardwareAddr(packet.Chaddr[:packet.Hlen])
		log.Printf("DHCP Packet received from %s (MAC: %s, XID: %d, Options length: %d)", remoteAddr.IP.String(), mac.String(), packet.Xid, len(packet.Options))

		if msgType, ok := packet.Options[53]; ok && len(msgType) == 1 {
			switch msgType[0] {
			case 1: // DHCPDISCOVER
				log.Printf("DHCPDISCOVER packet recieved")
			case 2: // DHCPOFFER
			case 3: // DHCPREQUEST
			case 4: // DHCPDECLINE
			case 5: // DHCPACK
			case 6: // DHCPNAK
			case 7: // DHCPRELEASE
			case 8: // DHCPINFORM
				log.Printf("DHCPINFORM packet recieved")
			default:
				log.Printf("Received DHCP message type: %d", msgType[0])
			}
		}
	}
}
