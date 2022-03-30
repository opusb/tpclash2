package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	clsconst "github.com/Dreamacro/clash/constant"
	"github.com/spf13/viper"
)

type TPClashConf struct {
	ProxyMode string

	ClashHome   string
	ClashConfig string
	ClashUI     string
	ClashURL    string
	MMDB        bool
	LocalProxy  bool

	TproxyMark     string
	ClashUser      string
	DirectGroup    string
	HijackDNS      []string
	DisableExtract bool

	Debug bool
}

type ClashConf struct {
	Debug       bool
	DNSHost     string
	DNSPort     string
	TProxyPort  string
	FakeIPRange string
	ExternalUI  string
}

type ProxyMode interface {
	addForward() error
	delForward() error

	addForwardDNS() error
	delForwardDNS() error

	addLocal() error
	delLocal() error

	addLocalDNS() error
	delLocalDNS() error

	apply() error
	clean() error
}

func parseConf() (*ClashConf, error) {
	debug := viper.GetString("log-level")
	enhancedMode := viper.GetString("dns.enhanced-mode")
	tproxyPort := viper.GetInt("tproxy-port")
	dnsListen := viper.GetString("dns.listen")
	fakeIPRange := viper.GetString("dns.fake-ip-range")
	externalUI := viper.GetString("external-ui")

	if strings.ToLower(enhancedMode) != clsconst.DNSFakeIP.String() {
		return nil, fmt.Errorf("only support fake-ip dns mode")
	}

	if tproxyPort < 1 {
		return nil, fmt.Errorf("tproxy port in clash config is missing(tproxy-port)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(dnsListen)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config: %v", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config: %v", err)
	}

	if dport < 1 {
		return nil, fmt.Errorf("dns port in clash config is missing(dns.listen)")
	}

	if fakeIPRange == "" {
		fakeIPRange = "198.18.0.1/16"
	}

	if externalUI == "" {
		externalUI = "dashboard"
	}

	return &ClashConf{
		Debug:       strings.ToLower(debug) == "debug",
		DNSHost:     dnsHost,
		DNSPort:     dnsPort,
		TProxyPort:  strconv.Itoa(tproxyPort),
		FakeIPRange: fakeIPRange,
		ExternalUI:  externalUI,
	}, nil
}
