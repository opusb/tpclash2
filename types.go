package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	clsconst "github.com/Dreamacro/clash/constant"
	"github.com/spf13/viper"
)

type Conf struct {
	Debug       bool
	DNSHost     string
	DNSPort     string
	TProxyPort  string
	FakeIPRange string
	ExternalUI  string
}

func parseConf() (*Conf, error) {
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

	return &Conf{
		Debug:       strings.ToLower(debug) == "debug",
		DNSHost:     dnsHost,
		DNSPort:     dnsPort,
		TProxyPort:  strconv.Itoa(tproxyPort),
		FakeIPRange: fakeIPRange,
		ExternalUI:  externalUI,
	}, nil
}
