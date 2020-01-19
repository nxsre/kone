package k1

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/soopsio/gosocks"
	"github.com/soopsio/kone/tcpip"
	socks5Proxy "github.com/soopsio/proxy"
)

type UDPTunnel struct {
	session        *NatSession
	clientAddr     *net.UDPAddr
	localConn      *net.UDPConn
	remoteUDPConn  *net.UDPConn
	remoteTCPConn  net.Conn
	remoteHostType byte
	remoteHost     string
	remotePort     uint16
	BndHost        string
	BndPort        uint16
}

func (tunnel *UDPTunnel) SetDeadline(duration time.Duration) error {
	err := tunnel.remoteTCPConn.SetDeadline(time.Now().Add(duration))
	if err != nil {
		return err
	}
	return tunnel.remoteUDPConn.SetDeadline(time.Now().Add(duration))
}

func (tunnel *UDPTunnel) Pump() error {
	b := make([]byte, MTU)
	for {
		n, err := tunnel.remoteUDPConn.Read(b)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return nil
			}
			return err
		}
		udpReq, err := gosocks.ParseUDPRequest(b[:n])
		if err != nil {
			return err
		}

		_, err = tunnel.localConn.WriteToUDP(udpReq.Data, tunnel.clientAddr)
		if err != nil {
			return err
		}

		// close if it is a dns query
		if tunnel.remotePort == 53 {
			return nil
		}
	}
}

func (tunnel *UDPTunnel) Write(b []byte) (int, error) {
	req := &gosocks.UDPRequest{
		Frag:     0,
		HostType: tunnel.remoteHostType,
		DstHost:  tunnel.remoteHost,
		DstPort:  tunnel.remotePort,
		Data:     b,
	}

	n, err := tunnel.remoteUDPConn.WriteTo(gosocks.PackUDPRequest(req), gosocks.SocksAddrToNetAddr("udp", tunnel.BndHost, tunnel.BndPort).(*net.UDPAddr))
	if err != nil {
		logger.Errorf("[udp] write to socks5 failed: %s", err)
	}
	return n, err
}

type UDPRelay struct {
	one       *One
	nat       *Nat
	relayIP   net.IP
	relayPort uint16

	lock    sync.Mutex
	tunnels map[string]*UDPTunnel
}

func (r *UDPRelay) grabTunnel(localConn *net.UDPConn, clientAddr *net.UDPAddr) *UDPTunnel {
	r.lock.Lock()
	defer r.lock.Unlock()
	addr := clientAddr.String()
	tunnel := r.tunnels[addr]
	if tunnel == nil {
		port := uint16(clientAddr.Port)
		session := r.nat.getSession(port)
		if session == nil {
			logger.Errorf("[udp] %s:%d > %s:%d no session", session.srcIP, session.srcPort, session.dstIP, session.dstPort)
			return nil
		}

		one := r.one
		var host, proxy string
		if record := one.dnsTable.GetByIP(session.dstIP); record != nil {
			host = record.Hostname
			proxy = record.Proxy
		} else if one.dnsTable.Contains(session.dstIP) {
			logger.Debugf("[udp] %s:%d > %s:%d dns expired", session.srcIP, session.srcPort, session.dstIP, session.dstPort)
			return nil
		} else {
			host = session.dstIP.String()
		}
		remoteAddr := fmt.Sprintf("%s:%d", host, session.dstPort)
		logger.Debugf("[udp] %s:%d > %s proxy %q", session.srcIP, session.srcPort, remoteAddr, proxy)
		if remoteAddr == "" {
			return nil
		}

		proxies := one.proxies
		socks5TCPConn, err := proxies.Dial("udp", proxy, remoteAddr)
		if err != nil {
			logger.Errorf("[udp] dial %s by proxy %q failed: %s", remoteAddr, proxy, err)
			return nil
		}

		socks5UDPListen, socks5Reply, err := socks5Proxy.Socks5UDPRequest(socks5TCPConn, "0.0.0.0", 0)

		var hostType byte
		if ip := net.ParseIP(host); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				hostType = socks5Proxy.Socks5AtypIP4
			} else {
				hostType = socks5Proxy.Socks5AtypIP6
			}
		} else {
			hostType = socks5Proxy.Socks5AtypDomain
		}

		tunnel = &UDPTunnel{
			session:        session,
			clientAddr:     clientAddr,
			localConn:      localConn,
			remoteUDPConn:  socks5UDPListen,
			remoteTCPConn:  socks5TCPConn,
			remoteHostType: hostType,
			remoteHost:     host,
			remotePort:     session.dstPort,
			BndHost:        socks5Reply.BndHost,
			BndPort:        socks5Reply.BndPort,
		}

		logger.Debugf("[udp] %s:%d > %v: new tunnel", session.srcIP, session.srcPort, remoteAddr)

		r.tunnels[addr] = tunnel
		go func() {
			err := tunnel.Pump()
			if err != nil {
				logger.Errorf("[udp] pump to %v failed: %v", tunnel.remoteUDPConn.RemoteAddr(), err)
			}
			logger.Debugf("[udp] %s:%d > %v: destroy tunnel", tunnel.session.srcIP, tunnel.session.srcPort, remoteAddr)
			r.close(tunnel, addr)
		}()
	}
	tunnel.SetDeadline(NatSessionLifeSeconds * time.Second)
	return tunnel
}

func (r *UDPRelay) handlePacket(localConn *net.UDPConn, cliaddr *net.UDPAddr, packet []byte) {
	tunnel := r.grabTunnel(localConn, cliaddr)
	if tunnel == nil {
		logger.Errorf("[udp] %v > %v: grap tunnel failed", cliaddr, localConn.LocalAddr())
		return
	}
	_, err := tunnel.Write(packet)
	if err != nil {
		logger.Errorf("[udp] %v", err)
	}
}

func (r *UDPRelay) close(tunnel *UDPTunnel, addr string) {
	tunnel.remoteUDPConn.Close()
	tunnel.remoteTCPConn.Close()

	r.lock.Lock()
	delete(r.tunnels, addr)
	r.lock.Unlock()
}

func (r *UDPRelay) Serve() error {
	addr := &net.UDPAddr{IP: r.relayIP, Port: int(r.relayPort)}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}

	for {
		b := make([]byte, MTU)
		n, clientAddr, err := conn.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go r.handlePacket(conn, clientAddr, b[:n])
	}
}

// redirect udp packet to relay
func (r *UDPRelay) Filter(wr io.Writer, ipPacket tcpip.IPv4Packet) {
	udpPacket := tcpip.UDPPacket(ipPacket.Payload())

	srcIP := ipPacket.SourceIP()
	dstIP := ipPacket.DestinationIP()
	srcPort := udpPacket.SourcePort()
	dstPort := udpPacket.DestinationPort()

	one := r.one

	if bytes.Equal(srcIP, r.relayIP) && srcPort == r.relayPort {
		// from remote
		session := r.nat.getSession(dstPort)
		if session == nil {
			logger.Debugf("[udp] %s:%d > %s:%d: no session", srcIP, srcPort, dstIP, dstPort)
			return
		}
		ipPacket.SetSourceIP(session.dstIP)
		ipPacket.SetDestinationIP(session.srcIP)
		udpPacket.SetSourcePort(session.dstPort)
		udpPacket.SetDestinationPort(session.srcPort)
	} else if one.dnsTable.Contains(dstIP) { // is fake ip
		// redirect to relay
		isNew, port := r.nat.allocSession(srcIP, dstIP, srcPort, dstPort)

		ipPacket.SetSourceIP(dstIP)
		udpPacket.SetSourcePort(port)
		ipPacket.SetDestinationIP(r.relayIP)
		udpPacket.SetDestinationPort(r.relayPort)

		if isNew {
			logger.Debugf("[udp] %s:%d > %s:%d: shape to %s:%d > %s:%d",
				srcIP, srcPort, dstIP, dstPort, dstIP, port, r.relayIP, r.relayPort)
		}
	} else {
		// redirect to relay
		isNew, port := r.nat.allocSession(srcIP, dstIP, srcPort, dstPort)

		ipPacket.SetSourceIP(dstIP)
		udpPacket.SetSourcePort(port)
		ipPacket.SetDestinationIP(r.relayIP)
		udpPacket.SetDestinationPort(r.relayPort)

		if isNew {
			logger.Debugf("[udp] %s:%d > %s:%d: shape to %s:%d > %s:%d",
				srcIP, srcPort, dstIP, dstPort, dstIP, port, r.relayIP, r.relayPort)
		}
	}

	// write back packet
	udpPacket.ResetChecksum(ipPacket.PseudoSum())
	ipPacket.ResetChecksum()
	wr.Write(ipPacket)
}

func NewUDPRelay(one *One, cfg NatConfig) *UDPRelay {
	r := new(UDPRelay)
	r.one = one
	r.nat = NewNat(cfg.NatPortStart, cfg.NatPortEnd)
	r.relayIP = one.ip
	r.relayPort = cfg.ListenPort
	r.tunnels = make(map[string]*UDPTunnel)
	return r
}
