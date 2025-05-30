# PolicyBGP

[![Go Version](https://img.shields.io/github/go-mod/go-version/IPA-CyberLab/policybgp)](https://golang.org/)
[![License](https://img.shields.io/github/license/IPA-CyberLab/policybgp)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/IPA-CyberLab/policybgp)](https://goreportcard.com/report/github.com/IPA-CyberLab/policybgp)

**PolicyBGP** enables policy-driven traffic engineering for network operators, especially those without their own Autonomous System (AS), by advertising BGP routes for specific ASes with user-defined nexthops. This allows for granular control over traffic paths, optimizing for cost, latency, or bandwidth.

## Use Cases

### 1. Multi-ISP Traffic Routing

PolicyBGP is ideal for small network operators who do not own their own AS, but have access to multiple ISPs.

With PolicyBGP, you can direct different types of traffic through the most appropriate ISP—for example:
- Route video streaming (e.g., Netflix) through an unmetered connection.
- Route gaming traffic through a low-latency ISP.
- Prioritize critical business traffic over a reliable, high-availability ISP.

### 2. Local Internet Breakout

In enterprise environments, it is common to route all Internet traffic from branch offices through a central data center via VPN. However, due to increasing bandwidth demands, this approach is often unsustainable.

A "Local Internet Breakout" strategy allows sites to send low-risk traffic (e.g., video conferencing) directly to the Internet via local ISPs while retaining VPN paths for sensitive or critical data.

PolicyBGP enables site routers to selectively route traffic to local ISPs while maintaining the default route through the data center.

## How It Works

PolicyBGP uses the [IP to ASN Lite database](https://db-ip.com/db/download/ip-to-asn-lite) from db-ip.com, along with a set of user-defined policy rules.

It establishes a BGP session with your router and advertises routes for the specified ASes using the configured nexthops.

> **Note:** Your router must support [BGP](https://en.wikipedia.org/wiki/Border_Gateway_Protocol). On Linux-based routers, you can use BGP daemons like [BIRD](https://bird.network.cz/) to receive and inject routes into the kernel routing table. Most commercial routers also support BGP.

## Installation

**Requirements:**
- Go 1.24.2 or later

Install using:

```bash
go install github.com/IPA-CyberLab/policybgp/cmd/policybgp@latest
```

## Usage

### Defining Policies

Specify policies via the command line using the following format: `--policy ASN,ip4-nexthop[,ip6-nexthop]`

Example policies:
- `--policy 15169,192.168.1.1` - Route traffic to Google (ASN 15169) via 192.168.1.1 (IPv4 only).
- `--policy 32934,10.0.0.1,2001:db8::1` - Route traffic to Facebook (ASN 32934) via both IPv4 and IPv6 nexthops.

### Running PolicyBGP

```bash
policybgp serve \
  --dbpath ./work/dbip-asn-lite.csv.gz \
  --peer 192.168.0.1:10179 \
  --policy 15169,192.168.1.1 \
  --policy 32934,10.0.0.1,2001:db8::1
```

## Development

### Setting up a test environment

1. **Start BIRD BGP server for testing:**
   ```bash
   bird -c hack/bird3.test.conf -s /tmp/bird.test.ctl -d
   ```

2. **Connect to BIRD CLI:**
   ```bash
   birdc -s /tmp/bird.test.ctl
   ```

3. **Run PolicyBGP against the test instance:**
   ```bash
   go run ./cmd/policybgp --verbose serve \
     --dbpath ./work/dbip-asn-lite.csv.gz \
     --peer localhost:10179 \
     --policy 15169,192.168.100.100
   ```

4. **Inspect the advertised routes using GoBGP CLI:**
   ```bash
   gobgp neighbor localhost
   gobgp global rib summary
   ```

### Building and Testing

```bash
# Build the binary
go build ./cmd/policybgp

# Run tests
go test ./...
```

## Contributing

We welcome contributions! Please feel free to submit issues, feature requests, or pull requests.

## License

This project is licensed under Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [GoBGP](https://github.com/osrg/gobgp) for BGP protocol implementation