package resolver

import (
	"context"
	"errors"
	"net"
)

func ResolveWired(host string) (*net.TCPAddr, error) {
	resolver := net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return net.Dial(network, "1.1.1.1:53")
		},
	}
	_, results, err := resolver.LookupSRV(context.Background(), "", "", "_wired._tcp."+host)
	if err != nil {
		return nil, err
	}

	if len(results) < 1 {
		return nil, errors.New("failed to resolve " + host)
	}

	addrs, err := resolver.LookupHost(context.Background(), results[0].Target)
	if err != nil {
		return nil, err
	}

	if len(addrs) < 1 {
		return nil, errors.New("failed to resolve " + results[0].Target)
	}

	return &net.TCPAddr{
		IP:   net.ParseIP(addrs[0]),
		Port: int(results[0].Port),
	}, nil
}
