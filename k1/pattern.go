package k1

import (
	jsoniter "github.com/json-iterator/go"
	"net"
	"sort"
	"strings"

	"github.com/nxsre/kone/geoip"
	"github.com/nxsre/kone/tcpip"
)

const (
	schemeDomain        = "DOMAIN"
	schemeDomainSuffix  = "DOMAIN-SUFFIX"
	schemeDomainKeyword = "DOMAIN-KEYWORD"
	schemeIPCountry     = "IP-COUNTRY"
	schemeIPCIDR        = "IP-CIDR"
)

type Pattern interface {
	Name() string
	Scheme() string
	Policy() string
	Proxy() string
	Add(string)
	Remove(string)
	Match(val interface{}) bool
	MarshalJSON() ([]byte, error)
	//UnmarshalJSON(data []byte) error
}

// DOMAIN
type DomainPattern struct {
	name   string
	policy string
	proxy  string
	vals   map[string]bool
}

func (p *DomainPattern) Name() string {
	return p.name
}

func (p *DomainPattern) Policy() string {
	return p.policy
}

func (p *DomainPattern) Proxy() string {
	return p.proxy
}

func (p *DomainPattern) Match(val interface{}) bool {
	v, ok := val.(string)
	if !ok {
		return false
	}
	v = strings.ToLower(v)
	if p.vals[v] {
		return true
	}
	return false
}

func (p *DomainPattern) Add(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		p.vals[val] = true
	}
}

func (p *DomainPattern) Remove(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		delete(p.vals, val)
	}
}

func (p *DomainPattern) Scheme() string {
	return schemeDomain
}

func (p *DomainPattern) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(map[string]interface{}{
		"name":   p.name,
		"policy": p.policy,
		"proxy":  p.proxy,
		"vals":   p.vals,
		"schema": p.Scheme(),
	})
}

func NewDomainPattern(name, policy, proxy string, vals []string) Pattern {
	p := new(DomainPattern)
	p.name = name
	p.policy = policy
	p.proxy = proxy
	p.vals = make(map[string]bool)
	for _, val := range vals {
		val = strings.ToLower(val)
		if len(val) > 0 { // ignore empty
			p.vals[val] = true
		}
	}
	return p
}

// DOMAIN-SUFFIX
type DomainSuffixPattern struct {
	name   string
	policy string
	proxy  string
	vals   map[string]bool
}

func (p *DomainSuffixPattern) Name() string {
	return p.name
}

func (p *DomainSuffixPattern) Policy() string {
	return p.policy
}

func (p *DomainSuffixPattern) Proxy() string {
	return p.proxy
}

func (p *DomainSuffixPattern) Match(val interface{}) bool {
	v, ok := val.(string)
	if !ok {
		return false
	}

	v = strings.ToLower(v)
	for {
		if p.vals[v] {
			return true
		}

		pos := strings.Index(v, ".")
		if pos < 0 {
			break
		}
		v = v[pos+1:]
	}
	return false
}

func (p *DomainSuffixPattern) Add(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		p.vals[val] = true
	}
}

func (p *DomainSuffixPattern) Remove(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		delete(p.vals, val)
	}
}

func (p *DomainSuffixPattern) Scheme() string {
	return schemeDomainSuffix
}

func (p *DomainSuffixPattern) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(map[string]interface{}{
		"name":   p.name,
		"policy": p.policy,
		"proxy":  p.proxy,
		"vals":   p.vals,
		"schema": p.Scheme(),
	})
}

func NewDomainSuffixPattern(name, policy, proxy string, vals []string) Pattern {
	p := new(DomainSuffixPattern)
	p.name = name
	p.policy = policy
	p.proxy = proxy
	p.vals = make(map[string]bool)
	for _, val := range vals {
		p.Add(val)
	}
	return p
}

// DOMAIN-KEYWORD
type DomainKeywordPattern struct {
	name   string
	policy string
	proxy  string
	vals   map[string]bool
}

func (p *DomainKeywordPattern) Name() string {
	return p.name
}

func (p *DomainKeywordPattern) Policy() string {
	return p.policy
}

func (p *DomainKeywordPattern) Proxy() string {
	return p.proxy
}

func (p *DomainKeywordPattern) Match(val interface{}) bool {
	v, ok := val.(string)
	if !ok {
		return false
	}
	v = strings.ToLower(v)
	for k := range p.vals {
		if strings.Index(v, k) >= 0 {
			return true
		}
	}
	return false
}

func (p *DomainKeywordPattern) Add(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		p.vals[val] = true
	}
}

func (p *DomainKeywordPattern) Remove(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		delete(p.vals, val)
	}
}

func (p *DomainKeywordPattern) Scheme() string {
	return schemeDomainKeyword
}

func (p *DomainKeywordPattern) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(map[string]interface{}{
		"name":   p.name,
		"policy": p.policy,
		"proxy":  p.proxy,
		"vals":   p.vals,
		"schema": p.Scheme(),
	})
}

func NewDomainKeywordPattern(name, policy, proxy string, vals []string) Pattern {
	p := new(DomainKeywordPattern)
	p.name = name
	p.policy = policy
	p.proxy = proxy
	p.vals = make(map[string]bool)
	for _, val := range vals {
		p.Add(val)
	}
	return p
}

// IP-COUNTRY
type IPCountryPattern struct {
	name   string
	policy string
	proxy  string
	vals   map[string]bool
}

func (p *IPCountryPattern) Name() string {
	return p.name
}

func (p *IPCountryPattern) Policy() string {
	return p.policy
}

func (p *IPCountryPattern) Proxy() string {
	return p.proxy
}

func (p *IPCountryPattern) Match(val interface{}) bool {
	var country string
	switch ip := val.(type) {
	case uint32:
		country = geoip.QueryCountry(ip)
	case net.IP:
		country = geoip.QueryCountryByIP(ip)
	}
	logger.Debugf("proxy_name:%s,policy:%s,proxy:%s,values:%v,ip:%v,country:%s", p.Name(), p.Policy(), p.Proxy(), p.vals, val, country)

	return p.vals[country]
}

func (p *IPCountryPattern) Add(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		p.vals[val] = true
	}
}

func (p *IPCountryPattern) Remove(val string) {
	if len(val) > 0 { // ignore empty suffix
		val = strings.ToLower(val)
		delete(p.vals, val)
	}
}

func (p *IPCountryPattern) Scheme() string {
	return schemeIPCountry
}

func (p *IPCountryPattern) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(map[string]interface{}{
		"name":   p.name,
		"policy": p.policy,
		"proxy":  p.proxy,
		"vals":   p.vals,
		"schema": p.Scheme(),
	})
}

func NewIPCountryPattern(name, policy, proxy string, vals []string) Pattern {
	p := new(IPCountryPattern)
	p.name = name
	p.policy = policy
	p.proxy = proxy
	p.vals = make(map[string]bool)
	for _, val := range vals {
		if len(val) > 0 { // ignore empty country
			p.vals[val] = true
		}
	}
	return p
}

// IPRangeArray
type IPRange struct {
	Start uint32
	End   uint32
}

func (p *IPRange) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(map[string]interface{}{
		"start": tcpip.ConvertUint32ToIPv4(p.Start),
		"end":   tcpip.ConvertUint32ToIPv4(p.End),
	})
}

type IPRangeArray []IPRange

func (a IPRangeArray) Len() int           { return len(a) }
func (a IPRangeArray) Less(i, j int) bool { return a[i].End < a[j].End }
func (a IPRangeArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (a IPRangeArray) Contains(ip uint32) bool {
	l := len(a)
	i := sort.Search(l, func(i int) bool {
		n := a[i]
		return n.End >= ip
	})

	if i < l {
		n := a[i]
		if n.Start <= ip {
			return true
		}
	}
	return false
}

func (a IPRangeArray) ContainsIP(ip net.IP) bool {
	return a.Contains(tcpip.ConvertIPv4ToUint32(ip))
}

// IP-CIDR
type IPCIDRPattern struct {
	name   string
	policy string
	proxy  string
	vals   IPRangeArray
}

func (p *IPCIDRPattern) Name() string {
	return p.name
}

func (p *IPCIDRPattern) Policy() string {
	return p.policy
}

func (p *IPCIDRPattern) Proxy() string {
	return p.proxy
}

func (p *IPCIDRPattern) Match(val interface{}) bool {
	switch ip := val.(type) {
	case uint32:
		return p.vals.Contains(ip)
	case net.IP:
		return p.vals.ContainsIP(ip)
	}

	return false
}

func (p *IPCIDRPattern) Add(val string) {
	if len(val) > 0 { // ignore empty suffix
		if _, ipNet, err := net.ParseCIDR(val); err == nil {
			start := tcpip.ConvertIPv4ToUint32(ipNet.IP)
			_end := start + ^tcpip.ConvertIPv4ToUint32(net.IP(ipNet.Mask))
			// 判断是否已存在，不能重复添加记录
			if p.vals.Contains(start+1) || p.vals.Contains(_end-1) {
				return
			}
			p.vals = append(p.vals, IPRange{
				Start: start,
				End:   _end,
			})
		}
	}
}

func (p *IPCIDRPattern) Remove(val string) {
	if len(val) > 0 { // ignore empty suffix
		if _, ipNet, err := net.ParseCIDR(val); err == nil {
			start := tcpip.ConvertIPv4ToUint32(ipNet.IP)
			_end := start + ^tcpip.ConvertIPv4ToUint32(net.IP(ipNet.Mask))
			for k, v := range p.vals {
				if v.Start == start && v.End == _end {
					p.vals = append(p.vals[:k], p.vals[k+1:]...)
				}
			}
		}
	}
}

func (p *IPCIDRPattern) Scheme() string {
	return schemeIPCIDR
}

func (p *IPCIDRPattern) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(map[string]interface{}{
		"name":   p.name,
		"policy": p.policy,
		"proxy":  p.proxy,
		"vals":   p.vals,
		"schema": p.Scheme(),
	})
}

func NewIPCIDRPattern(name, policy, proxy string, vals []string) Pattern {
	p := new(IPCIDRPattern)
	p.name = name
	p.policy = policy
	p.proxy = proxy
	for _, val := range vals {
		p.Add(val)
	}

	sort.Sort(p.vals)
	return p
}

var patternSchemes map[string]func(string, string, string, []string) Pattern

func init() {
	patternSchemes = make(map[string]func(string, string, string, []string) Pattern)
	patternSchemes[schemeDomain] = NewDomainPattern
	patternSchemes[schemeDomainSuffix] = NewDomainSuffixPattern
	patternSchemes[schemeDomainKeyword] = NewDomainKeywordPattern
	patternSchemes[schemeIPCountry] = NewIPCountryPattern
	patternSchemes[schemeIPCIDR] = NewIPCIDRPattern
}

func IsExistPatternScheme(scheme string) bool {
	_, ok := patternSchemes[scheme]
	return ok
}

func CreatePattern(name string, config *PatternConfig) Pattern {
	if f := patternSchemes[config.Scheme]; f != nil {
		logger.Infof("[pattern] name: %s, Scheme: %s, Policy: %s, Proxy: %s", name, config.Scheme, config.Policy, config.Proxy)
		return f(name, config.Policy, config.Proxy, config.V)
	}
	return nil
}
