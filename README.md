# DHCipher

A custom DHCPv4 server built from scratch in Go, implementing the full DORA (Discover, Offer, Request, Acknowledge) state machine as defined in [RFC 2131](https://datatracker.ietf.org/doc/html/rfc2131) and [RFC 2132](https://datatracker.ietf.org/doc/html/rfc2132).

This project was built as a deep-dive into low-level network programming — every DHCP packet is manually parsed and serialized at the byte level using the RFC specification, with no third-party networking libraries.

## Features

- **Full DORA State Machine** — Handles `DHCPDISCOVER`, `DHCPOFFER`, `DHCPREQUEST`, and `DHCPACK` transactions for complete IP address assignment.
- **Manual Packet Parsing** — Raw UDP bytes are deserialized into Go structs using byte offsets defined in RFC 2131 (Section 2, Table 1).
- **Manual Packet Serialization** — Server responses are constructed byte-by-byte, including the 240-byte BOOTP-compatible header and variable-length DHCP options.
- **IP Address Pool Management** — Configurable IP range with automatic allocation and lease tracking via an in-memory map.
- **Client Requested IP Support (Option 50)** — Clients can request a specific IP address during discovery; the server honors it if available.
- **DNS Server Distribution (Option 6)** — Provides DNS resolver addresses to clients for internet connectivity.
- **Lease Expiry & Cleanup** — A background goroutine periodically sweeps the lease map and frees expired addresses.
- **Persistent Leases** — Lease state is saved to a JSON file on disk and restored on server restart, surviving crashes and reboots.
- **DHCPRELEASE Handling** — Clients can gracefully return their IP addresses to the pool.
- **DHCPINFORM Support** — Clients with static IPs can request network configuration without a lease.
- **YAML Configuration** — All server parameters (IP range, subnet mask, DNS servers, lease duration, log path) are configurable via a `config.yaml` file.
- **Thread Safety** — All lease map operations are protected by `sync.Mutex` to prevent race conditions between the main loop and background goroutines.
- **Graceful Shutdown** — Intercepts `SIGINT`/`SIGTERM` signals to safely close connections and persist lease state before exiting.
- **File Logging** — All server activity is logged to a configurable log file for auditing and debugging.

## RFC Implementation

This server implements the core protocol behavior described in:

- **[RFC 2131](https://datatracker.ietf.org/doc/html/rfc2131)** — Dynamic Host Configuration Protocol
  - Section 2: Protocol Summary (message format, Table 1)
  - Section 3.1: Client-server interaction — DHCPDISCOVER/OFFER/REQUEST/ACK
  - Section 3.4: DHCPRELEASE
  - Section 3.5: DHCPINFORM
  - Section 4.1: Constructing and sending DHCP messages
- **[RFC 2132](https://datatracker.ietf.org/doc/html/rfc2132)** — DHCP Options and BOOTP Vendor Extensions
  - Option 1 (Subnet Mask), Option 3 (Router), Option 6 (DNS), Option 50 (Requested IP), Option 51 (Lease Time), Option 53 (Message Type), Option 54 (Server Identifier)


## Getting Started

### Prerequisites

- Go 1.21+ installed
- Linux (required for network namespace testing)
- Root privileges (`sudo`) for binding to port 67

### Installation

```bash
git clone https://github.com/yourusername/DHCipher.git
cd DHCipher
go mod tidy
```

### Configuration

Copy the example configuration and edit it to match your network:

```bash
cp config.example.yaml config.yaml
```

```yaml
server_ip: "192.168.1.1"
starting_ip: "192.168.1.100"
ending_ip: "192.168.1.200"
subnet_mask: "255.255.255.0"
router_ip: "192.168.1.1"
lease_duration_hours: 24
dns_servers: ["8.8.8.8", "1.1.1.1"]
log_file: "/var/log/dhcipher.log"
lease_file: "/var/lib/dhcipher/leases.json"
```

### Running the Server

```bash
sudo go run main.go
```

## Testing with Linux Network Namespaces

The server was tested end-to-end against ISC's `dhclient` using isolated Linux network namespaces. This setup creates a virtual network cable between the host and a sandboxed environment, allowing safe DHCP testing without affecting the host network.

### Setting Up the Test Environment

```bash
# Create the namespace and virtual ethernet pair
sudo ip netns add dhcp_test_ns
sudo ip link add veth_host type veth peer name veth_ns

# Move one end into the namespace
sudo ip link set veth_ns netns dhcp_test_ns

# Configure the host side (must match server_ip in config.yaml)
sudo ip addr add 192.168.1.1/24 dev veth_host
sudo ip link set veth_host up

# Bring up the namespace side (no IP assigned — the DHCP server will provide one!)
sudo ip netns exec dhcp_test_ns ip link set veth_ns up

# Allow traffic through the firewall
sudo firewall-cmd --zone=trusted --add-interface=veth_host

# Fix checksum offloading for virtual interfaces
sudo ethtool -K veth_host tx off
```

### Running the DHCP Client

In a separate terminal:

```bash
sudo ip netns exec dhcp_test_ns dhclient -v -d veth_ns
```

### Expected Output

**Server logs:**
```
Starting DHCP Server...
Listening for DHCP packets on 0.0.0.0:67
DHCPDISCOVER packet received
Client requested specific IP: 192.168.1.100
Sent DHCPOFFER for IP 192.168.1.100 to MAC aa:bb:cc:dd:ee:ff
Sent DHCPACK for IP 192.168.1.100 to MAC aa:bb:cc:dd:ee:ff
```

**Client output:**
```
DHCPDISCOVER on veth_ns to 255.255.255.255 port 67
DHCPOFFER of 192.168.1.100 from 192.168.1.1
DHCPREQUEST for 192.168.1.100 on veth_ns to 255.255.255.255
DHCPACK of 192.168.1.100 from 192.168.1.1
bound to 192.168.1.100
```

### Verify the Assignment

```bash
sudo ip netns exec dhcp_test_ns ip addr show veth_ns
```

You should see `192.168.1.100/24` assigned to `veth_ns`.

### Cleanup

```bash
sudo ip netns delete dhcp_test_ns
sudo ip link delete veth_host
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    main.go                          │
│  LoadConfig() → SetupLogging → NewServer() → Start  │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                  server.go                          │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │ Signal   │  │ Cleanup  │  │   Main UDP Loop   │  │
│  │ Handler  │  │ Ticker   │  │                   │  │
│  │ Goroutine│  │ Goroutine│  │  ReadFromUDP()    │  │
│  │          │  │          │  │       │           │  │
│  │ SIGINT → │  │ Every 1m │  │       ▼           │  │
│  │ Save &   │  │ Sweep    │  │  ParsePacket()    │  │
│  │ Exit     │  │ Expired  │  │       │           │  │
│  └──────────┘  │ Leases   │  │       ▼           │  │
│                └──────────┘  │  DORA Switch      │  │
│                              │  Case 1 → Offer   │  │
│                              │  Case 3 → Ack/Nak │  │
│                              │  Case 7 → Release │  │
│                              │  Case 8 → Inform  │  │
│                              └───────────────────┘  │
└───────────────────────┬─────────────────────────────┘
                        │
           ┌────────────┼────────────┐
           ▼            ▼            ▼
      ┌─────────┐ ┌──────────┐ ┌──────────────┐
      │ dhcp.go │ │ pool.go  │ │persistence.go│
      │         │ │          │ │              │
      │ Parse   │ │ Allocate │ │ SaveLeases() │
      │ Packet  │ │ IP Pool  │ │ LoadLeases() │
      │         │ │ Mutex    │ │              │
      │Serialize│ │ Cleanup  │ │  leases.json │
      └─────────┘ └──────────┘ └──────────────┘
```

## License

This project is open source and available under the [MIT License](LICENSE).
