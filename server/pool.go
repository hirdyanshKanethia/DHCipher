package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

// IPPool defines the Pool of IP addresses available to be allocated
type IPPool struct {
	mu            sync.Mutex
	ServerIP      net.IP
	StartingIP    net.IP
	EndingIP      net.IP
	SubnetMask    net.IPMask
	RouterIP      net.IP
	LeaseDuration time.Duration
	LeaseMap      map[string]*Lease // key: IP leased to the client
	DNSServers    []net.IP
	BroadcastIP   net.IP
	LeaseJSONfile string
}

// NewIPPool is a constructor func for IPPool struct
func NewIPPool(srvrIP net.IP, strtIP net.IP, endIP net.IP, snmask net.IPMask, routerIP net.IP, DNSServers []string, leaseduration time.Duration, LeaseJSONfile string) *IPPool {
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
		LeaseMap:      make(map[string]*Lease),
		DNSServers:    DNSIPs,
		BroadcastIP:   getBroadcastIP(srvrIP, snmask),
		LeaseJSONfile: LeaseJSONfile,
	}
}

// AllocateIP is the public implementation of the AllocateIP function with the mutex locking mechanism
func (r *IPPool) AllocateIP(clientMAC net.HardwareAddr) (*Lease, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.allocateIP(clientMAC)
}

// allocateIP allocates IP address to the provided MAC address. If one already exists, returns the existing lease with renewed LeaseDuration
// it doesn't use mutex locking because it is a private function intended to be invoked by other methods that already use mutex locking
func (r *IPPool) allocateIP(clientMAC net.HardwareAddr) (*Lease, error) {
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

// CleanupExpiredLeases is used in a goroutine that runs every 1 minute to cleanup expired leases
func (r *IPPool) CleanupExpiredLeases() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	for ip, lease := range r.LeaseMap {
		if now.After(lease.ExpiresAt) {
			log.Printf("Lease expired for IP %s (MAC: %s)", lease.LeasedIP.String(), lease.ClientMAC.String())
			delete(r.LeaseMap, ip)
		}
	}
}

// AllocateRequestedIP is used to try and allocate the IP requested by the client.
// It can extend the current lease or allocate a new lease which may or may not assign the reqIP depending upon the availability of the reqIP
func (r *IPPool) AllocateRequestedIP(clientMAC net.HardwareAddr, reqIP net.IP) (*Lease, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if lease, ok := r.LeaseMap[reqIP.String()]; ok {
		// Case: reqIP is given to same client
		if bytes.Equal(clientMAC, lease.ClientMAC) {
			lease.ExpiresAt = time.Now().Add(r.LeaseDuration)
			return lease, nil
		}

		// Case: reqIP is given to another client with remaining lease time
		if time.Now().Before(lease.ExpiresAt) {
			return r.allocateIP(clientMAC)
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

	return r.allocateIP(clientMAC)
}

func (r *IPPool) releaseIP(mac net.HardwareAddr) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ipStr, lease := range r.LeaseMap {
		if bytes.Equal(lease.ClientMAC, mac) {
			log.Printf("Released IP %s from MAC %s", lease.LeasedIP.String(), mac.String())
			delete(r.LeaseMap, ipStr)
			return
		}
	}

	log.Printf("DHCPRELEASE failed: no lease found for MAC %s", mac.String())
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
