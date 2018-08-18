package k1

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/FlowerWrong/proxy"
)

var errNoProxy = errors.New("no proxy")

type Proxies struct {
	proxies map[string]*proxy.Proxy
	dft     string // default proxy name
}

func (p *Proxies) Dial(network, proxy, addr string) (net.Conn, error) {
	if proxy == "" {
		return p.DefaultDial(network, addr)
	}

	dialer := p.proxies[proxy]
	if dialer != nil {
		return dialer.Dial(network, addr)
	}
	return nil, fmt.Errorf("invalid proxy: %s", proxy)
}

func (p *Proxies) DefaultDial(network, addr string) (net.Conn, error) {
	dialer := p.proxies[p.dft]
	if dialer == nil {
		return nil, errNoProxy
	}
	return dialer.Dial(network, addr)
}

func NewProxies(one *One, config map[string]*ProxyConfig) (*Proxies, error) {
	p := &Proxies{}

	proxies := make(map[string]*proxy.Proxy)
	for name, item := range config {
		proxyDialer, err := proxy.FromUrl(item.Url)
		if err != nil {
			return nil, err
		}

		if item.Default || p.dft == "" {
			p.dft = name
		}
		proxies[name] = proxyDialer

		// don't hijack proxyDialer domain
		host := proxyDialer.Url.Host
		index := strings.IndexByte(proxyDialer.Url.Host, ':')
		if index > 0 {
			host = proxyDialer.Url.Host[:index]
		}
		one.rule.DirectDomain(host)
	}
	p.proxies = proxies
	logger.Infof("[proxies] default proxy: %q", p.dft)
	return p, nil
}
