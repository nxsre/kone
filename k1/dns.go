package k1

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/miekg/dns/dnsutil"
)

const (
	dnsDefaultPort         = 53
	dnsDefaultTtl          = 600
	dnsDefaultPacketSize   = 4096
	dnsDefaultReadTimeout  = 5
	dnsDefaultWriteTimeout = 5
)

var resolveErr = errors.New("resolve error")

type Dns struct {
	one         *One
	server      *dns.Server
	clients     DnsClients
	nameservers []string
}

type DnsClient struct {
	client *dns.Client
	ns     NameServer
}

type DnsClients map[string]*DnsClient // map["tcp(8.8.8.8:53)"]*dns.Client

func (c DnsClients) Exchange(r *dns.Msg, ns string) (*dns.Msg, time.Duration, error) {
	n := c[ns]
	return n.client.Exchange(r, n.ns.String())
}

func (d *Dns) resolve(r *dns.Msg) (*dns.Msg, error) {
	var wg sync.WaitGroup
	msgCh := make(chan *dns.Msg, 1)

	qname := r.Question[0].Name

	Q := func(ns string) {
		defer wg.Done()
		logger.Debugf("nameserver:%s qname:%s", ns, qname)
		r, rtt, err := d.clients.Exchange(r, ns)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			logger.Debugf("[dns] resolve %s on %s failed: %v", qname, ns, err)
			return
		}

		if r.Rcode == dns.RcodeServerFailure {
			logger.Debugf("[dns] resolve %s on %s failed: code %d", qname, ns, r.Rcode)
			return
		}

		logger.Debugf("[dns] resolve %s on %s, code: %d, rtt: %d", qname, ns, r.Rcode, rtt)

		select {
		case msgCh <- r:
		default:
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for _, ns := range d.nameservers {
		wg.Add(1)
		go Q(ns)

		select {
		case r := <-msgCh:
			return r, nil
		case <-ticker.C:
			continue
		}
	}

	wg.Wait()

	select {
	case r := <-msgCh:
		return r, nil
	default:
		logger.Debugf("[dns] query %s failed", qname)
		return nil, resolveErr
	}
}

func (d *Dns) fillRealIP(record *DomainRecord, r *dns.Msg) {
	// resolve
	msg, err := d.resolve(r)
	if err != nil || len(msg.Answer) == 0 {
		return
	}
	record.SetRealIP(msg)
}

func (d *Dns) doIPv4Query(r *dns.Msg) (*dns.Msg, error) {
	one := d.one

	domain := dnsutil.TrimDomainName(r.Question[0].Name, ".")

	// if is a reject domain
	if one.rule.Reject(domain) {
		return nil, errors.New(domain + "is a reject domain")
	}

	// if is a non-proxy-domain
	if one.dnsTable.IsNonProxyDomain(domain) {
		logger.Infof("IsNonProxyDomain: %v", domain)
		return d.resolve(r)
	}

	// if have already hijacked
	record := one.dnsTable.Get(domain)
	if record != nil {
		logger.Infof("have already hijacked: %v", domain)
		return record.Answer(r), nil
	}

	// match by domain
	matched, proxy := one.rule.Proxy(domain)
	logger.Infof("matched:%v, proxy:%s", matched, proxy)

	// if domain use proxy
	if matched && proxy != "" {
		if record := one.dnsTable.Set(domain, proxy); record != nil {
			go d.fillRealIP(record, r)
			return record.Answer(r), nil
		}
	}

	// resolve
	msg, err := d.resolve(r)
	if err != nil || len(msg.Answer) == 0 {
		return msg, err
	}

	if !matched {
		// try match by cname and ip
	OuterLoop:
		for _, item := range msg.Answer {
			switch answer := item.(type) {
			case *dns.A:
				// test ip
				_, proxy = one.rule.Proxy(answer.A)
				break OuterLoop
			case *dns.CNAME:
				// test cname
				matched, proxy = one.rule.Proxy(answer.Target)
				if matched && proxy != "" {
					break OuterLoop
				}
			default:
				logger.Noticef("[dns] unexpected response %s -> %v", domain, item)
			}
		}
		// if ip use proxy
		if proxy != "" {
			if record := one.dnsTable.Set(domain, proxy); record != nil {
				record.SetRealIP(msg)
				logger.Infof("[dns] ---------- %s is a proxy-domain via %s by ip", domain, proxy)
				return record.Answer(r), nil
			}
		} else {
			logger.Infof("[dns] ---------- %s is a non-proxy-domain by ip", domain)
		}
	}

	// set domain as a non-proxy-domain
	one.dnsTable.SetNonProxyDomain(domain, msg.Answer[0].Header().Ttl)

	// final
	return msg, err
}

func isIPv4Query(q dns.Question) bool {
	if q.Qclass == dns.ClassINET && q.Qtype == dns.TypeA {
		return true
	}
	return false
}

func (d *Dns) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	isIPv4 := isIPv4Query(r.Question[0])
	logger.Infof("remote_addr:%s, r: %+v   isIPv4:%v", w.RemoteAddr(), r.Question, isIPv4)

	var msg *dns.Msg
	var err error

	if isIPv4 {
		msg, err = d.doIPv4Query(r)
	} else {
		msg, err = d.resolve(r)
	}

	if err != nil {
		logger.Errorf("%e", err)
		dns.HandleFailed(w, r)
	} else {
		w.WriteMsg(msg)
	}
}

func (d *Dns) Serve() error {
	logger.Infof("[dns] listen on %s", d.server.Addr)
	return d.server.ListenAndServe()
}

func NewDns(one *One, cfg DnsConfig) (*Dns, error) {
	d := new(Dns)
	d.one = one

	server := &dns.Server{
		Net:          "udp",
		Addr:         fmt.Sprintf("%s:%d", fixTunIP(one.ip), cfg.DnsPort),
		Handler:      dns.HandlerFunc(d.ServeDNS),
		UDPSize:      int(cfg.DnsPacketSize),
		ReadTimeout:  time.Duration(cfg.DnsReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.DnsWriteTimeout) * time.Second,
	}

	d.server = server
	d.nameservers = cfg.Nameserver
	d.clients = GetDnsClients(cfg)

	return d, nil
}

func GetDnsClients(cfg DnsConfig) ( DnsClients) {
	clients := make(DnsClients)
	for _, ns := range cfg.Nameserver {
		nameserver := parseNs(ns)
		clients[ns] = &DnsClient{
			client: &dns.Client{
				Net:          nameserver.Protocol,
				UDPSize:      cfg.DnsPacketSize,
				ReadTimeout:  time.Duration(cfg.DnsReadTimeout) * time.Second,
				WriteTimeout: time.Duration(cfg.DnsWriteTimeout) * time.Second,
			},
			ns: nameserver,
		}
	}
	return clients
}

type NameServer struct {
	Protocol string
	Addr     string
	Port     int
}

func (ns *NameServer) String() string {
	return fmt.Sprintf("%s:%d", ns.Addr, ns.Port)
}

func parseNs(ns string) NameServer {
	re := regexp.MustCompile(`(?P<protocol>[a-z\-]+)?\(?(?P<addr>[0-9\.]+)(:(?P<port>\d+))?\)?`)
	match := re.FindStringSubmatch(ns)
	groupNames := re.SubexpNames()

	result := make(map[string]string)

	// 转换为map
	for i, name := range groupNames {
		if i != 0 && name != "" { // 第一个分组为空（也就是整个匹配）
			result[name] = match[i]
		}
	}

	protocol := result["protocol"]
	if protocol != "tcp" && protocol != "tcp-tls" {
		protocol = "udp"
	}

	port, _ := strconv.Atoi(result["port"])
	if port == 0 {
		port = 53
	}

	return NameServer{
		Protocol: protocol,
		Addr:     result["addr"],
		Port:     port,
	}
}
