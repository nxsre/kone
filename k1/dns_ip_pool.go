package k1

import (
	"hash/adler32"
	"net"

	"github.com/nxsre/kone/tcpip"
)

const DnsIPPoolMaxSpace = 0x3ffff // 4*65535

type DnsIPPool struct {
	base  uint32
	space uint32
	flags []bool
}

func (pool *DnsIPPool) Capacity() int {
	return int(pool.space)
}

func (pool *DnsIPPool) Contains(ip net.IP) bool {
	index := tcpip.ConvertIPv4ToUint32(ip) - pool.base
	if index < pool.space {
		return true
	}
	return false
}

func (pool *DnsIPPool) Release(ip net.IP) {
	index := tcpip.ConvertIPv4ToUint32(ip) - pool.base
	if index < pool.space {
		pool.flags[index] = false
	}
}

// use tips as a hint to find a stable index
func (pool *DnsIPPool) Alloc(tips string) net.IP {
	index := adler32.Checksum([]byte(tips)) % pool.space
	if pool.flags[index] {
		logger.Debugf("[dns] %s is not in main index: %d", tips, index)
		for i, used := range pool.flags {
			if !used {
				index = uint32(i)
				break
			}
		}
	}

	if pool.flags[index] {
		return nil
	}
	pool.flags[index] = true
	return tcpip.ConvertUint32ToIPv4(pool.base + index)
}

func NewDnsIPPool(ip net.IP, subnet *net.IPNet) *DnsIPPool {
	// 地址池的起始地址为网段的第一个IP+1
	base := tcpip.ConvertIPv4ToUint32(subnet.IP) + 1
	// 地址池的结束地址为网段的广播IP+1
	max := base + ^tcpip.ConvertIPv4ToUint32(net.IP(subnet.Mask)) - 1

	// space should not over 0x3ffff
	space := max - base
	if space > DnsIPPoolMaxSpace {
		space = DnsIPPoolMaxSpace
	}
	flags := make([]bool, space)

	// ip is used by tun
	index := tcpip.ConvertIPv4ToUint32(ip) - base
	if index < space {
		flags[index] = true
	}

	return &DnsIPPool{
		base:  base,
		space: space,
		flags: flags,
	}
}
