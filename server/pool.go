package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"time"
)

// IPPool defines the Pool of IP addresses available to be allocated
type IPPool struct {
	ServerIP      net.IP
	StartingIP    net.IP
	EndingIP      net.IP
	SubnetMask    net.IPMask
	RouterIP      net.IP
	LeaseDuration time.Duration
	LeaseMap      map[string]*Lease
	DNSServers    []net.IP
	BroadcastIP   net.IP
}

// NewIPPool is a constructor func for IPPool struct
func NewIPPool(srvrIP net.IP, strtIP net.IP, endIP net.IP, snmask net.IPMask, routerIP net.IP, DNSServers []string, leaseduration time.Duration) *IPPool {
	var DNSIPs []net.IP
	for _, serverStr := range DNSServers {
		parsedIP := net.ParseIP(serverStr)
		if parsedIP != nil {
			DNSIPs = append(DNSIPs, parsedIP)
		}
	}

	return &IPPool{
		ServerIP:      srvrIP,
		StartingIP:    strtIP,
		EndingIP:      endIP,
		SubnetMask:    snmask,
		RouterIP:      routerIP,
		LeaseDuration: leaseduration,
		LeaseMap:      make(map[string]*Lease), // key: IP leased to the client
		DNSServers:    DNSIPs,
		BroadcastIP:   getBroadcastIP(srvrIP, snmask),
	}
}

// AllocateIP allocates IP address to the provided MAC address. If one already exists, returns the existing lease with renewed LeaseDuration
func (r *IPPool) AllocateIP(clientMAC net.HardwareAddr) (*Lease, error) {
	for _, lease := range r.LeaseMap {
		if bytes.Equal(lease.ClientMAC, clientMAC) {
			lease.ExpiresAt = time.Now().Add(r.LeaseDuration)
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

// getBroadcastIP calculates the broadcast address for a given IP and Subnet Mask
func getBroadcastIP(ip net.IP, mask net.IPMask) net.IP {
	ip4 := ip.To4()
	broadcast := make(net.IP, 4)
	for i := range 4 {
		broadcast[i] = ip4[i] | ^mask[i]
	}
	return broadcast
}

func (r *IPPool) CleanupExpiredLeases() {
	now := time.Now()

	for ip, lease := range r.LeaseMap {
		if now.After(lease.ExpiresAt) {
			log.Printf("Lease expired for IP %s (MAC: %s)", lease.LeasedIP.String(), lease.ClientMAC.String())
			delete(r.LeaseMap, ip)
		}
	}
}

func (r *IPPool) AllocateRequestedIP(clientMAC net.HardwareAddr, reqIP net.IP) (*Lease, error) {
	if lease, ok := r.LeaseMap[reqIP.String()]; ok {
		// Case: reqIP is given to same client
		if bytes.Equal(clientMAC, lease.ClientMAC) {
			lease.ExpiresAt = time.Now().Add(r.LeaseDuration)
			return lease, nil
		}

		if time.Now().Before(lease.ExpiresAt) {
			return r.AllocateIP(clientMAC)
		}
	}

	reqIPInt := ip2int(reqIP)
	if reqIPInt >= ip2int(r.StartingIP) && reqIPInt <= ip2int(r.EndingIP) {
		newLease := &Lease{
			LeasedIP:  reqIP,
			ClientMAC: clientMAC,
			ExpiresAt: time.Now().Add(r.LeaseDuration),
		}
		r.LeaseMap[reqIP.String()] = newLease
		return newLease, nil
	}

	return r.AllocateIP(clientMAC)
}
