package geoip

import (
	"github.com/mitchellh/mapstructure"
	. "github.com/nxsre/kone/internal"
	"github.com/oschwald/maxminddb-golang"
	"net"
	"sort"
)

var (
	geoIPLen = len(geoIP)
	logger   = GetLogger()
)

func QueryCountry(ip uint32) string {
	i := sort.Search(geoIPLen, func(i int) bool {
		n := geoIP[i]
		return n.End >= ip
	})

	var country string
	if i < geoIPLen {
		n := geoIP[i]
		if n.Start <= ip {
			country = n.Name
		}
	}
	return country
}

var mmdb *maxminddb.Reader

func init() {
	db, err := maxminddb.Open("mmdbs/GeoLite2-Country.mmdb")
	if err != nil {
		logger.Errorf("open maxminddb failed:%s", err.Error())
	}
	mmdb = db
}

func QueryConuntryByIPDetails(ip net.IP) GeoLite2Country {
	record := make(map[string]interface{})
	country := GeoLite2Country{}

	mmdb.Lookup(ip, &record)
	mapstructure.Decode(record, &country)
	return country
}

func QueryCountryByIP(ip net.IP) string {
	ip = ip.To4()
	if ip == nil {
		return ""
	}

	//v := uint32(ip[0]) << 24
	//v += uint32(ip[1]) << 16
	//v += uint32(ip[2]) << 8
	//v += uint32(ip[3])
	//return QueryCountry(v)

	return QueryConuntryByIPDetails(ip).Country.IsoCode
}

func QueryCountryByString(v string) string {
	ip := net.ParseIP(v)
	if ip == nil {
		return ""
	}
	return QueryCountryByIP(ip)
}

type GeoLite2Country struct {
	Continent struct {
		Code      string `json:"code" yaml:"code" mapstructure:"code"`
		GeonameID int64  `json:"geoname_id" yaml:"geoname_id" mapstructure:"geoname_id"`
		Names     struct {
			De    string `json:"de" yaml:"de" mapstructure:"de"`
			En    string `json:"en" yaml:"en" mapstructure:"en"`
			Es    string `json:"es" yaml:"es" mapstructure:"es"`
			Fr    string `json:"fr" yaml:"fr" mapstructure:"fr"`
			Ja    string `json:"ja" yaml:"ja" mapstructure:"ja"`
			Pt_BR string `json:"pt-BR" yaml:"pt-BR" mapstructure:"pt-BR"`
			Ru    string `json:"ru" yaml:"ru" mapstructure:"ru"`
			Zh_CN string `json:"zh-CN" yaml:"zh-CN" mapstructure:"zh-CN"`
		} `json:"names" yaml:"names" mapstructure:"names"`
	} `json:"continent" yaml:"continent" mapstructure:"continent"`
	Country struct {
		GeonameID int64  `json:"geoname_id" yaml:"geoname_id" mapstructure:"geoname_id"`
		IsoCode   string `json:"iso_code" yaml:"iso_code" mapstructure:"iso_code"`
		Names     struct {
			De    string `json:"de" yaml:"de" mapstructure:"de"`
			En    string `json:"en" yaml:"en" mapstructure:"en"`
			Es    string `json:"es" yaml:"es" mapstructure:"es"`
			Fr    string `json:"fr" yaml:"fr" mapstructure:"fr"`
			Ja    string `json:"ja" yaml:"ja" mapstructure:"ja"`
			Pt_BR string `json:"pt-BR" yaml:"pt-BR" mapstructure:"pt-BR"`
			Ru    string `json:"ru" yaml:"ru" mapstructure:"ru"`
			Zh_CN string `json:"zh-CN" yaml:"zh-CN" mapstructure:"zh-CN"`
		} `json:"names" yaml:"names" mapstructure:"names"`
	} `json:"country" yaml:"country" mapstructure:"country"`
	RegisteredCountry struct {
		GeonameID         int64  `json:"geoname_id" yaml:"geoname_id" mapstructure:"geoname_id"`
		IsInEuropeanUnion bool   `json:"is_in_european_union" yaml:"is_in_european_union" mapstructure:"is_in_european_union"`
		IsoCode           string `json:"iso_code" yaml:"iso_code" mapstructure:"iso_code"`
		Names             struct {
			De    string `json:"de" yaml:"de" mapstructure:"de"`
			En    string `json:"en" yaml:"en" mapstructure:"en"`
			Es    string `json:"es" yaml:"es" mapstructure:"es"`
			Fr    string `json:"fr" yaml:"fr" mapstructure:"fr"`
			Ja    string `json:"ja" yaml:"ja" mapstructure:"ja"`
			Pt_BR string `json:"pt-BR" yaml:"pt-BR" mapstructure:"pt-BR"`
			Ru    string `json:"ru" yaml:"ru" mapstructure:"ru"`
			Zh_CN string `json:"zh-CN" yaml:"zh-CN" mapstructure:"zh-CN"`
		} `json:"names" yaml:"names" mapstructure:"names"`
	} `json:"registered_country" yaml:"registered_country" mapstructure:"registered_country"`
	RepresentedCountry struct {
		GeonameID int64  `json:"geoname_id" yaml:"geoname_id" mapstructure:"geoname_id"`
		IsoCode   string `json:"iso_code" yaml:"iso_code" mapstructure:"iso_code"`
		Names     struct {
			De    string `json:"de" yaml:"de" mapstructure:"de"`
			En    string `json:"en" yaml:"en" mapstructure:"en"`
			Es    string `json:"es" yaml:"es" mapstructure:"es"`
			Fr    string `json:"fr" yaml:"fr" mapstructure:"fr"`
			Ja    string `json:"ja" yaml:"ja" mapstructure:"ja"`
			Pt_BR string `json:"pt-BR" yaml:"pt-BR" mapstructure:"pt-BR"`
			Ru    string `json:"ru" yaml:"ru" mapstructure:"ru"`
			Zh_CN string `json:"zh-CN" yaml:"zh-CN" mapstructure:"zh-CN"`
		} `json:"names" yaml:"names" mapstructure:"names"`
		Type string `json:"type" yaml:"type" mapstructure:"type"`
	} `json:"represented_country" yaml:"represented_country" mapstructure:"represented_country"`
}
