package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"time"
)

// Defines the Pool of IP addresses available to be allocated
type IPPool struct {
	ServerIP      net.IP
	StartingIP    net.IP
	EndingIP      net.IP
	SubnetMask    net.IPMask
	RouterIP      net.IP
	LeaseDuration time.Duration
	LeaseMap      map[string]*Lease
}

// Constructor func for IPPool struct
func NewIPPool(srvrIP net.IP, strtIP net.IP, endIP net.IP, snmask net.IPMask, routerIP net.IP, leaseduration time.Duration) *IPPool {
	return &IPPool{
		ServerIP:      srvrIP,
		StartingIP:    strtIP,
		EndingIP:      endIP,
		SubnetMask:    snmask,
		RouterIP:      routerIP,
		LeaseDuration: leaseduration,
		LeaseMap:      make(map[string]*Lease), // key: IP leased
	}
}

// Allocates IP address to the provided MAC address. If one already exists, returns the existing lease
func (r *IPPool) AllocateIP(clientMAC net.HardwareAddr) (*Lease, error) {
	for _, lease := range r.LeaseMap {
		if bytes.Equal(lease.ClientMAC, clientMAC) {
			return lease, nil
		}
	}

	for i := ip2int(r.StartingIP); i <= ip2int(r.EndingIP); i += 1 {
		ip := int2ip(i)

		_, ok := r.LeaseMap[ip.String()]

		if !ok {
			newLease := &Lease{
				LeasedIP:  ip,
				ClientMAC: clientMAC,
				ExpiresAt: time.Now().Add(r.LeaseDuration),
			}

			r.LeaseMap[ip.String()] = newLease

			return newLease, nil
		}
	}

	return nil, errors.New("IP pool exhausted")
}

// HELPER FUNCTIONS

// converts net.IP to a 32-bit integer value
func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}

	return binary.BigEndian.Uint32(ip)
}

// converts a 32-bit integer value to net.IP
func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
