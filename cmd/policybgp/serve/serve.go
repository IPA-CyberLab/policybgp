package serve

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/IPA-CyberLab/policybgp/asinfo"
	"github.com/osrg/gobgp/v4/api"
	"github.com/osrg/gobgp/v4/pkg/server"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/prototext"
)

type Policy struct {
	ASN        uint32
	IP4NextHop netip.Addr
	IP6NextHop netip.Addr

	ASInfo *asinfo.ASInfo
}

var Command = &cli.Command{
	Name:                      "serve",
	Usage:                     "Run BGP peer that injects the policies",
	DisableSliceFlagSeparator: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "dbpath",
			Usage:    "dbip-asn-lite csv file (or csv.gz)",
			Required: true,
		},
		&cli.Uint32Flag{
			Name:  "bgpASN",
			Usage: "BGP ASN of myself",
			Value: 64513,
		},
		&cli.StringFlag{
			Name:  "routerId",
			Usage: "Router ID of myself",
			Value: "10.64.51.3",
		},
		&cli.StringFlag{
			Name:     "peer",
			Usage:    "BGP peer address in the format <ip>:<port>",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:  "policy",
			Usage: "Policy routing policy to be distributed to the peer. Format: <asn>!<ip4_nexthop>[!<ip6_nexthop>]",
		},
		&cli.StringFlag{
			Name:  "listenGobgp",
			Usage: "Enable GoBGP gRPC server on the specified address",
			Value: ":50051",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		logger := zap.L()
		s := logger.Named("policybgp.serve").Sugar()

		bgpASN := cmd.Uint32("bgpASN")
		if bgpASN < 1 || bgpASN > 65535 {
			return cli.Exit(fmt.Errorf("BGP ASN %d invalid. It must be between 1 and 65535", bgpASN), 1)
		}

		routerId := cmd.String("routerId")
		if routerId == "" {
			return cli.Exit("routerId cannot be empty", 1)
		}

		peerS := cmd.String("peer")
		peerHost, peerPortS, err := net.SplitHostPort(peerS)
		if err != nil {
			return cli.Exit(fmt.Errorf("invalid peer format %q: %w", peerS, err), 1)
		}
		peerPort := 179 // default BGP port
		if peerPortS != "" {
			var err error
			peerPort, err = net.LookupPort("tcp", peerPortS)
			if err != nil {
				return cli.Exit(fmt.Errorf("invalid peer port %q: %w", peerPortS, err), 1)
			}
		}
		if peerPort < 1 || peerPort > 65535 {
			return cli.Exit(fmt.Errorf("peer port %d invalid. It must be between 1 and 65535", peerPort), 1)
		}

		policyStrings := cmd.StringSlice("policy")
		policies := make([]*Policy, 0, len(policyStrings))
		for _, policyStr := range policyStrings {
			parts := strings.SplitN(policyStr, ",", 3)
			if len(parts) < 2 || len(parts) > 3 {
				s.Debugf("parts: %+v", parts)
				return cli.Exit(fmt.Errorf("invalid policy format %q. Expected <asn>,<ip4_nexthop>[,<ip6_nexthop>]", policyStr), 1)
			}

			asn, err := strconv.Atoi(parts[0])
			if err != nil {
				return cli.Exit(fmt.Errorf("invalid ASN in policy %q: %w", policyStr, err), 1)
			}
			if asn == 0 || asn > 4294967295 { // Max ASN for 32-bit
				return cli.Exit(fmt.Errorf("ASN %d in policy %q is out of valid range", asn, policyStr), 1)
			}

			ip4NextHop, err := netip.ParseAddr(parts[1])
			if err != nil || !ip4NextHop.Is4() {
				return cli.Exit(fmt.Errorf("invalid IPv4 nexthop %q in policy %q: %w", parts[1], policyStr, err), 1)
			}

			var ip6NextHop netip.Addr
			if len(parts) == 3 && parts[2] != "" {
				ip6NextHop, err = netip.ParseAddr(parts[2])
				if err != nil || !ip6NextHop.Is6() {
					return cli.Exit(fmt.Errorf("invalid IPv6 nexthop %q in policy %q: %w", parts[2], policyStr, err), 1)
				}
			}

			policies = append(policies, &Policy{
				ASN:        uint32(asn),
				IP4NextHop: ip4NextHop,
				IP6NextHop: ip6NextHop,
			})
		}

		s.Infof("Parsed %d policies", len(policies))
		if len(policies) == 0 {
			return cli.Exit("No policies provided. Use --policy flag to specify at least one policy.", 1)
		}

		dbPath := cmd.String("dbpath")
		db, err := asinfo.ParseASInfoCSVFromFile(dbPath, s.Desugar())
		if err != nil {
			return err
		}

		for _, pol := range policies {
			info := db[int(pol.ASN)]
			if info == nil {
				return cli.Exit(fmt.Errorf("ASN %d not found in database %q", pol.ASN, dbPath), 1)
			}

			pol.ASInfo = info

			s.Infof("Configuring policy: %d prefixes to ASN %d (%s) nexthop v4 %s and v6 %s",
				len(info.Prefixes), pol.ASN, info.Organization, pol.IP4NextHop, pol.IP6NextHop)
		}

		peer := &api.Peer{
			Conf: &api.PeerConf{
				NeighborAddress: peerHost,
				PeerAsn:         bgpASN,
			},
			Transport: &api.Transport{
				RemotePort: uint32(peerPort),
			},
			Timers: &api.Timers{Config: &api.TimersConfig{
				ConnectRetry:      3,
				HoldTime:          90,
				KeepaliveInterval: 30,
			}},
			AfiSafis: []*api.AfiSafi{
				{Config: &api.AfiSafiConfig{Family: &api.Family{
					Afi:  api.Family_AFI_IP,
					Safi: api.Family_SAFI_UNICAST,
				}}},
				{Config: &api.AfiSafiConfig{Family: &api.Family{
					Afi:  api.Family_AFI_IP6,
					Safi: api.Family_SAFI_UNICAST,
				}}},
			},
		}

		sopts := []server.ServerOption{
			server.LoggerOption(&logAdapter{l: s.Named("gobgp")}),
		}
		if listenAddr := cmd.String("listenGobgp"); listenAddr != "" {
			sopts = append(sopts, server.GrpcListenAddress(listenAddr))
		}
		bgps := server.NewBgpServer(sopts...)
		go bgps.Serve()

		s.Infof("Starting BGP server with ASN %d and Router ID %s", bgpASN, routerId)
		if err := bgps.StartBgp(ctx, &api.StartBgpRequest{
			Global: &api.Global{
				Asn:        bgpASN,
				RouterId:   routerId,
				ListenPort: -1, // gobgp won't listen on tcp:179
			},
		}); err != nil {
			return err
		}

		// monitor the change of the peer state
		// TBD: is this really needed?
		if err := bgps.WatchEvent(ctx, &api.WatchEventRequest{Peer: &api.WatchEventRequest_Peer{}}, func(r *api.WatchEventResponse) {
			if p := r.GetPeer(); p != nil && p.Type == api.WatchEventResponse_PeerEvent_TYPE_STATE {
				s.Info(p)
			}
		}); err != nil {
			return err
		}

		peerText, err := prototext.Marshal(peer)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to marshal peer: %w", err), 1)
		}
		s.Infof("Adding peer: %s", peerText)

		if err := bgps.AddPeer(ctx, &api.AddPeerRequest{Peer: peer}); err != nil {
			return cli.Exit(fmt.Errorf("failed to add peer: %w", err), 1)
		}

		for _, pol := range policies {
			for _, pre := range pol.ASInfo.Prefixes {
				if pre.Addr().Is4() {
					nlri := &api.NLRI{Nlri: &api.NLRI_Prefix{Prefix: &api.IPAddressPrefix{
						Prefix:    pre.Addr().String(),
						PrefixLen: uint32(pre.Bits()),
					}}}
					attrs := []*api.Attribute{
						{Attr: &api.Attribute_Origin{Origin: &api.OriginAttribute{
							Origin: uint32(api.RouteOriginType_ORIGIN_IGP),
						}}},
						{Attr: &api.Attribute_NextHop{NextHop: &api.NextHopAttribute{
							NextHop: pol.IP4NextHop.String(),
						}}},
						{Attr: &api.Attribute_AsPath{AsPath: &api.AsPathAttribute{
							Segments: []*api.AsSegment{{
								Type:    api.AsSegment_TYPE_AS_SEQUENCE,
								Numbers: []uint32{pol.ASN},
							}},
						}}},
					}

					if _, err := bgps.AddPath(ctx, &api.AddPathRequest{
						Path: &api.Path{
							Family: &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
							Nlri:   nlri,
							Pattrs: attrs,
						},
					}); err != nil {
						return cli.Exit(fmt.Errorf("failed to add IPv4 path for ASN %d: %w", pol.ASN, err), 1)
					}
					s.Infof("Added IPv4 path %v for ASN %d (%s) with nexthop %s", pre, pol.ASN, pol.ASInfo.Organization, pol.IP4NextHop)
				} else if pre.Addr().Is6() {
					if !pol.IP6NextHop.IsValid() {
						s.Debugf("Skipping IPv6 path for ASN %d (%s) because IPv6 nexthop is not configured", pol.ASN, pol.ASInfo.Organization)
						continue
					}
					nlri := &api.NLRI{Nlri: &api.NLRI_Prefix{Prefix: &api.IPAddressPrefix{
						Prefix:    pre.Addr().String(),
						PrefixLen: uint32(pre.Bits()),
					}}}
					attrs := []*api.Attribute{
						{Attr: &api.Attribute_Origin{Origin: &api.OriginAttribute{
							Origin: uint32(api.RouteOriginType_ORIGIN_IGP),
						}}},
						{Attr: &api.Attribute_NextHop{NextHop: &api.NextHopAttribute{
							NextHop: pol.IP6NextHop.String(),
						}}},
						{Attr: &api.Attribute_AsPath{AsPath: &api.AsPathAttribute{
							Segments: []*api.AsSegment{{
								Type:    api.AsSegment_TYPE_AS_SEQUENCE,
								Numbers: []uint32{pol.ASN},
							}},
						}}},
					}

					if _, err := bgps.AddPath(ctx, &api.AddPathRequest{
						Path: &api.Path{
							Family: &api.Family{Afi: api.Family_AFI_IP6, Safi: api.Family_SAFI_UNICAST},
							Nlri:   nlri,
							Pattrs: attrs,
						},
					}); err != nil {
						return cli.Exit(fmt.Errorf("failed to add IPv6 path for ASN %d: %w", pol.ASN, err), 1)
					}
					s.Infof("Added IPv6 path %v for ASN %d (%s) with nexthop %s", pre, pol.ASN, pol.ASInfo.Organization, pol.IP6NextHop)
				} else {
					s.Errorf("unsupported prefix address family %s for ASN %d", pre.Addr().String(), pol.ASN)
				}
			}
		}

		<-ctx.Done()

		return nil
	},
}
