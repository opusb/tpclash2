package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type TPClashConf struct {
	ProxyMode string

	ClashHome   string
	ClashConfig string
	ClashUI     string
	LocalProxy  bool

	TproxyMark     string
	ClashUser      string
	DirectGroup    string
	HijackIP       []net.IP
	HijackDNS      []string
	DisableExtract bool

	Debug bool
}

type ClashConf struct {
	Debug         bool
	EnhancedMode  string
	DNSHost       string
	DNSPort       string
	TProxyPort    string
	FakeIPRange   string
	InterfaceName string
}

// ParseClashConf Parses clash configuration and performs necessary checks
// based on proxy mode
func ParseClashConf() (*ClashConf, error) {
	debug := viper.GetString("log-level")
	enhancedMode := viper.GetString("dns.enhanced-mode")
	dnsListen := viper.GetString("dns.listen")
	tproxyPort := viper.GetInt("tproxy-port")
	fakeIPRange := viper.GetString("dns.fake-ip-range")
	interfaceName := viper.GetString("interface-name")
	tunEnabled := viper.GetBool("tun.enable")
	routingMark := viper.GetInt("routing-mark")

	// common check
	if strings.ToLower(enhancedMode) != "fake-ip" {
		return nil, fmt.Errorf("only support fake-ip dns mode(dns.enhanced-mode)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(dnsListen)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config(dns.listen): %v", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse clash dns listen config(dns.listen): %v", err)
	}
	if dport < 1 {
		return nil, fmt.Errorf("dns port in clash config is missing(dns.listen)")
	}

	if interfaceName == "" {
		return nil, fmt.Errorf("failed to parse clash interface name(interface-name): interface-name must be set")
	}

	if fakeIPRange == "" {
		fakeIPRange = "198.18.0.1/16"
	}

	switch conf.ProxyMode {
	case "tproxy":
		if tproxyPort < 1 {
			return nil, fmt.Errorf("tproxy port in clash config is missing(tproxy-port)")
		}
		if tunEnabled {
			return nil, fmt.Errorf("tun must be disabled in tproxy mode(tun.enable)")
		}
		if routingMark > 0 {
			return nil, fmt.Errorf("routing-mark cannot be set in tproxy mode(routing-mark)")
		}
	case "tun":
		if tproxyPort > 0 {
			return nil, fmt.Errorf("please delete the tproxy port in tun mode(tproxy-port)")
		}
		if !tunEnabled {
			return nil, fmt.Errorf("tun must be enabled in tun mode(tun.enable)")
		}
		if routingMark < 1 {
			return nil, fmt.Errorf("routing-mark must be set in tun mode(routing-mark)")
		}
	}

	return &ClashConf{
		Debug:         strings.ToLower(debug) == "debug",
		EnhancedMode:  enhancedMode,
		DNSHost:       dnsHost,
		DNSPort:       dnsPort,
		TProxyPort:    strconv.Itoa(tproxyPort),
		FakeIPRange:   fakeIPRange,
		InterfaceName: interfaceName,
	}, nil
}
