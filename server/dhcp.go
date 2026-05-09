// Package server implements a custom DHCPv4 server.
//
// It handles the full DORA state machine (Discover, Offer,
// Request, Acknowledge) as defined in RFC 2131, and manages
// a pool of IP addresses based on a YAML configuration.
package server

import (
	"encoding/binary"
	"errors"
	"net"
)

type DHCPPacket struct {
	Op      byte
	Htype   byte
	Hlen    byte
	Hops    byte
	Xid     uint32
	Secs    uint16
	Flags   uint16
	Ciaddr  net.IP
	Yiaddr  net.IP
	Siaddr  net.IP
	Giaddr  net.IP
	Chaddr  []byte
	Options map[byte][]byte
}

func ParsePacket(data []byte) (*DHCPPacket, error) {
	if len(data) < 240 {
		return nil, errors.New("packet data too short")
	}

	// Magic cookie check. If not found, return error
	if binary.BigEndian.Uint32(data[236:240]) != 0x63825363 {
		return nil, errors.New("magic cookie not found, packet is not DHCP")
	}

	packet := DHCPPacket{
		Op:      data[0],
		Htype:   data[1],
		Hlen:    data[2],
		Hops:    data[3],
		Xid:     binary.BigEndian.Uint32(data[4:8]),
		Secs:    binary.BigEndian.Uint16(data[8:10]),
		Flags:   binary.BigEndian.Uint16(data[10:12]),
		Ciaddr:  net.IP(data[12:16]),
		Yiaddr:  net.IP(data[16:20]),
		Siaddr:  net.IP(data[20:24]),
		Giaddr:  net.IP(data[24:28]),
		Chaddr:  data[28:44],
		Options: make(map[byte][]byte),
	}

	i := 240
	for i < len(data) {
		code := data[i]
		if code == byte(255) {
			break
		}

		if code == byte(0) {
			i++
			continue
		}

		length := int(data[i+1])
		value := data[i+2 : i+2+length]

		packet.Options[code] = value
		i += 2 + length
	}

	return &packet, nil
}

func (r DHCPPacket) Serialize() []byte {
	b := make([]byte, 240)

	b[0] = r.Op
	b[1] = r.Htype
	b[2] = r.Hlen
	b[3] = r.Hops

	binary.BigEndian.PutUint32(b[4:8], r.Xid)
	binary.BigEndian.PutUint16(b[8:10], r.Secs)
	binary.BigEndian.PutUint16(b[10:12], r.Flags)

	copy(b[12:16], r.Ciaddr.To4())
	copy(b[16:20], r.Yiaddr.To4())
	copy(b[20:24], r.Siaddr.To4())
	copy(b[24:28], r.Giaddr.To4())

	copy(b[28:44], r.Chaddr)

	binary.BigEndian.PutUint32(b[236:240], 0x63825363)

	// option 53 should be the first options in any DHCP packet
	if msgType, ok := r.Options[53]; ok {
		b = append(b, 53)
		b = append(b, byte(len(msgType)))
		b = append(b, msgType...)
	}

	for key, value := range r.Options {
		if key == 53 {
			continue
		}

		b = append(b, key)

		length := len(value)
		b = append(b, byte(length))

		b = append(b, value...)
	}

	b = append(b, 255)

	return b
}
