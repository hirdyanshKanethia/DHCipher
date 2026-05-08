package server

import (
	// "encoding/hex"
	// "fmt"
	"encoding/binary"
	"log"
	"net"
	"time"
	// "os"
	// "time"
)

type Server struct {
	Pool *IPPool
}

func NewServer() *Server {
	pool := NewIPPool(
		net.ParseIP("192.168.1.1"),
		net.ParseIP("192.168.1.100"),
		net.ParseIP("192.168.1.200"),
		net.IPv4Mask(255, 255, 255, 0),
		net.ParseIP("192.168.1.1"),
		24*time.Hour,
	)

	return &Server{Pool: pool}
}

func (s *Server) Start() {
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
			// DHCPDISCOVER: Replies with a DHCP offer packet if IP allocation is successful
			case 1:
				log.Printf("DHCPDISCOVER packet recieved")

				lease, err := s.Pool.AllocateIP(mac)
				if err != nil {
					log.Printf("[ERROR] Ran out of available IPs while allocating IP address to mac address (%s)", mac.String())
				}

				offerPacket := s.buildReply(packet, lease, 2)
				offerBytes := offerPacket.Serialize()
				conn.WriteToUDP(offerBytes, &net.UDPAddr{IP: net.ParseIP("192.168.1.255"), Port: 68})
				log.Printf("Sent DHCPOFFER for IP %s to MAC %s", lease.LeasedIP.String(), mac.String())

			case 2: // DHCPOFFER
			case 3: // DHCPREQUEST
				requestedIPBytes, ok := packet.Options[50]
				lease, _ := s.Pool.AllocateIP(mac)
				if ok {
					requestedIP := net.IP(requestedIPBytes)
					if !requestedIP.Equal(lease.LeasedIP) {
						log.Printf("DHCPNAK: Client requested %s but we offered %s", requestedIP.String(), lease.LeasedIP.String())

						nakPacket := s.buildReply(packet, lease, 6)
						conn.WriteToUDP(nakPacket.Serialize(), &net.UDPAddr{IP: net.ParseIP("192.168.1.255"), Port: 68})
						continue
					}
				}

				ackPacket := s.buildReply(packet, lease, 5)
				conn.WriteToUDP(ackPacket.Serialize(), &net.UDPAddr{IP: net.ParseIP("192.168.1.255"), Port: 68})
				log.Printf("Sent DHCPACK for IP %s to MAC %s", lease.LeasedIP.String(), mac.String())

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

// builds reply DHCPPacket objects depending upon msgType provided
func (s *Server) buildReply(req *DHCPPacket, lease *Lease, msgType byte) *DHCPPacket {
	replyPacket := DHCPPacket{
		Op:     2,
		Htype:  req.Htype,
		Hlen:   req.Hlen,
		Hops:   req.Hops,
		Xid:    req.Xid,
		Flags:  req.Flags,
		Yiaddr: lease.LeasedIP,
		Giaddr: req.Giaddr,
		Chaddr: req.Chaddr,
	}

	if msgType == 6 {
		replyPacket.Yiaddr = net.ParseIP("0.0.0.0")
	} else {
		replyPacket.Yiaddr = lease.LeasedIP
	}

	replyPacket.Options = make(map[byte][]byte)

	replyPacket.Options[53] = []byte{msgType}
	replyPacket.Options[1] = []byte(s.Pool.SubnetMask)
	replyPacket.Options[3] = []byte(s.Pool.RouterIP.To4())
	replyPacket.Options[54] = []byte(s.Pool.ServerIP.To4())

	leaseBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(leaseBytes, uint32(s.Pool.LeaseDuration.Seconds()))
	replyPacket.Options[51] = leaseBytes

	return &replyPacket
}
