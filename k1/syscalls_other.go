// +build !linux,!darwin,!windows

package k1

import (
	"errors"
	"net"
)

var errOS = errors.New("unsupported os")

func initTun(tun string, ipNet *net.IPNet, mtu int) error {
	return errOS
}

func addRoute(tun string, subnet *net.IPNet) error {
	return errOS
}

func fixTunIP(ip net.IP) net.IP {
	return ip
}
