package rss

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// newFeedHTTPClient returns the client used to fetch feeds. Unless allowPrivate
// is set, it refuses to connect to loopback, private, link-local and
// unspecified addresses. The check runs in the dialer, so it sees the address
// after DNS resolution and applies to every redirect hop.
func newFeedHTTPClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if !allowPrivate {
		dialer.Control = blockPrivateAddresses
	}

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			// no proxy, so the dialed address is always the feed host itself
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          20,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

func blockPrivateAddresses(network, address string, _ syscall.RawConn) error {
	addrPort, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("refusing to connect to %s: %w", address, err)
	}
	if isBlockedAddr(addrPort.Addr()) {
		return fmt.Errorf("refusing to connect to %s: address is not public", addrPort.Addr())
	}
	return nil
}

func isBlockedAddr(addr netip.Addr) bool {
	// ::ffff:127.0.0.1 and friends are judged as the IPv4 address they carry
	addr = addr.Unmap().WithZone("")
	return !addr.IsValid() ||
		addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}
