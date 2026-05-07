package server

import (
	"net"
	"time"
)

type Lease struct {
	LeasedIP  net.IP
	ClientMAC net.HardwareAddr
	ExpiresAt time.Time
}
